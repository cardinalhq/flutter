// Copyright 2025 CardinalHQ, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package script

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/emitter"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/metricproducer"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
	"github.com/cardinalhq/flutter/pkg/state"
	"github.com/cardinalhq/flutter/pkg/traceproducer"
)

type Script struct {
	actions          []scriptaction.ScriptAction
	metricGenerators map[string]generator.MetricGenerator
	metricProducers  map[string]metricproducer.MetricProducer
	traceProducers   map[string]traceproducer.TraceProducer
	emitters         []emitter.Emitter
	duration         time.Duration
	from             time.Duration
}

func NewScript() *Script {
	return &Script{
		actions:          []scriptaction.ScriptAction{},
		metricGenerators: map[string]generator.MetricGenerator{},
		metricProducers:  map[string]metricproducer.MetricProducer{},
		traceProducers:   map[string]traceproducer.TraceProducer{},
	}
}

func (s *Script) AddAction(action scriptaction.ScriptAction) {
	s.actions = append(s.actions, action)
}

func (s *Script) AddEmitter(emitter emitter.Emitter) {
	s.emitters = append(s.emitters, emitter)
}

func (s *Script) AddTraceProducer(id string, producer traceproducer.TraceProducer) {
	s.traceProducers[id] = producer
}

func (s *Script) Duration() time.Duration {
	return s.duration
}

func (s *Script) Dump(out io.Writer) error {
	if len(s.actions) == 0 {
		return errors.New("no script actions found in config")
	}

	for _, action := range s.actions {
		if err := json.NewEncoder(out).Encode(action); err != nil {
			return fmt.Errorf("error encoding action: %w", err)
		}
	}
	return nil
}

func Simulate(ctx context.Context, cfg *config.Config, rscript *Script, from time.Duration) error {
	if err := rscript.Prepare(cfg); err != nil {
		return fmt.Errorf("error creating running config: %w", err)
	}
	rscript.from = from
	return run(ctx, cfg, rscript)
}

func (s *Script) Prepare(cfg *config.Config) error {
	if len(s.actions) == 0 {
		return errors.New("no script actions found in config")
	}

	slices.SortFunc(s.actions, func(a, b scriptaction.ScriptAction) int {
		if v := int(a.At - b.At); v != 0 {
			return v
		}
		if v := strings.Compare(a.Type, b.Type); v != 0 {
			return v
		}
		return strings.Compare(a.ID, b.ID)
	})

	var err error
	s.duration, err = calculateDuration(cfg.Duration, s.actions)
	if err != nil {
		return fmt.Errorf("error calculating duration: %w", err)
	}

	// Create the metric generators
	for _, action := range s.actions {
		switch action.Type {
		case "metricGenerator":
			g, err := generator.CreateMetricGenerator(action)
			if err != nil {
				return errors.New("Error creating metric generator: " + err.Error())
			}
			s.metricGenerators[action.ID] = g
		default:
			// Ignore other types of actions for now
		}
	}

	return nil
}

func calculateDuration(cd time.Duration, actions []scriptaction.ScriptAction) (time.Duration, error) {
	if len(actions) == 0 {
		return 0, errors.New("no actions provided")
	}
	checkvalue := actions[0].At
	for _, action := range actions {
		checkvalue = max(checkvalue, action.At)
		checkvalue = max(checkvalue, action.To)
	}
	if cd == 0 {
		return checkvalue, nil
	}
	if cd < checkvalue {
		return 0, errors.New("Duration must be greater than or equal to the last script action time, or set to 0")
	}
	return cd, nil
}

func run(ctx context.Context, cfg *config.Config, rscript *Script) error {
	seed := cfg.Seed
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	rs := state.NewRunState(rscript.duration, seed)
	if cfg.WallclockStart.IsZero() {
		cfg.WallclockStart = time.Now()
	}
	seconds := int64(rs.Duration.Seconds())
	slog.Info("Running simulation", "duration", rs.Duration, "seed", seed, "wallclockStart", cfg.WallclockStart)
	for now := range seconds + 1 {
		rs.Tick = time.Duration(now) * time.Second
		rs.Wallclock = cfg.WallclockStart.Add(rs.Tick)
		err := tick(ctx, rscript, rs)
		if err != nil {
			return fmt.Errorf("error running script: %w", err)
		}
		if !cfg.Dryrun && rs.Tick < rscript.duration {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func tick(ctx context.Context, rscript *Script, rs *state.RunState) error {
	if len(rscript.actions) > rs.CurrentAction {
		if rscript.actions[rs.CurrentAction].At <= rs.Tick {
			action := rscript.actions[rs.CurrentAction]
			rs.CurrentAction++
			switch action.Type {
			case "metricGenerator":
				g, ok := rscript.metricGenerators[action.ID]
				if !ok {
					return fmt.Errorf("metric generator not found: %s", action.ID)
				}
				err := g.Reconfigure(action.At, action.Spec)
				if err != nil {
					return fmt.Errorf("error reconfiguring metric generator: %s", action.ID)
				}
			case "metric":
				if producer, ok := rscript.metricProducers[action.ID]; ok {
					if err := producer.Reconfigure(rscript.metricGenerators, action.Spec); err != nil {
						return fmt.Errorf("error reconfiguring metric exporter: %s", action.ID)
					}
				}
				producer, err := metricproducer.CreateMetricExporter(rscript.metricGenerators, action.ID, action)
				if err != nil {
					return fmt.Errorf("error creating metric exporter: %v", err)
				}
				rscript.metricProducers[action.ID] = producer
			case "disableMetric":
				if producer, ok := rscript.metricProducers[action.ID]; ok {
					producer.Disable()
				} else {
					return fmt.Errorf("disableMetric producer not found: %s", action.ID)
				}
			case "enableMetric":
				if producer, ok := rscript.metricProducers[action.ID]; ok {
					producer.Enable()
				} else {
					return fmt.Errorf("enableMetric producer not found: %s", action.ID)
				}
			case "traceRate":
				slog.Info("trace rate", "at", action.At, "to", action.To, "rate", action.Spec["rate"])
				producer, ok := rscript.traceProducers[action.ID]
				if !ok {
					return fmt.Errorf("trace producer not found: %s", action.ID)
				}
				rate, ok := action.Spec["rate"].(float64)
				if !ok {
					return fmt.Errorf("trace rate not found in action spec: %s", action.ID)
				}
				producer.SetRate(action.At, action.To, rs.Tick, rate)
				if start, ok := action.Spec["start"].(float64); ok {
					producer.SetStart(start)
				}
			default:
				return fmt.Errorf("unknown action type: %s", action.Type)
			}
		}
	}

	if err := emitMetrics(ctx, rscript, rs); err != nil {
		return fmt.Errorf("error emitting metrics: %w", err)
	}

	if err := emitTraces(ctx, rscript, rs); err != nil {
		return fmt.Errorf("error emitting traces: %w", err)
	}

	return nil
}

func emitMetrics(ctx context.Context, rscript *Script, rs *state.RunState) error {
	metricNames := make([]string, 0, len(rscript.metricProducers))
	for name := range rscript.metricProducers {
		metricNames = append(metricNames, name)
	}
	mb := signalbuilder.NewMetricsBuilder()
	for _, name := range metricNames {
		producer, ok := rscript.metricProducers[name]
		if !ok {
			return fmt.Errorf("metric producer not found: %s", name)
		}
		err := producer.Emit(rscript.metricGenerators, rs, mb)
		if err != nil {
			return fmt.Errorf("error emitting metric: %s", name)
		}
	}
	md := mb.Build()
	// if md.DataPointCount() > 0 {
	// 	slog.Info("Emitting metrics", "count", md.DataPointCount())
	// }

	if rs.Tick >= rscript.from {
		for _, emitter := range rscript.emitters {
			if err := emitter.EmitMetrics(ctx, rs, md); err != nil {
				return fmt.Errorf("error emitting metric: %w", err)
			}
		}
	}

	return nil
}

func emitTraces(ctx context.Context, rscript *Script, rs *state.RunState) error {
	tb := signalbuilder.NewTracesBuilder()
	for name, producer := range rscript.traceProducers {
		err := producer.Emit(rs, tb)
		if err != nil {
			return fmt.Errorf("error emitting trace: %s", name)
		}
	}
	td := tb.Build()
	// if td.SpanCount() > 0 {
	// 	rootCount := 0
	// 	for _, rspan := range td.ResourceSpans().All() {
	// 		for _, ilspan := range rspan.ScopeSpans().All() {
	// 			for _, span := range ilspan.Spans().All() {
	// 				if span.ParentSpanID().IsEmpty() {
	// 					rootCount++
	// 				}
	// 			}
	// 		}
	// 	}
	// 	slog.Info("Emitting traces", "spanCount", td.SpanCount(), "rootSpanCount", rootCount)
	// }

	if rs.Tick >= rscript.from {
		for _, emitter := range rscript.emitters {
			if err := emitter.EmitTraces(ctx, rs, td); err != nil {
				return fmt.Errorf("error emitting trace: %w", err)
			}
		}
	}

	return nil
}
