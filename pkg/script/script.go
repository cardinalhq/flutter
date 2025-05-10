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
	"slices"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/metricemitter"
	"github.com/cardinalhq/flutter/pkg/metricproducer"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
	"github.com/cardinalhq/flutter/pkg/state"
)

type Script struct {
	actions         []scriptaction.ScriptAction
	generators      map[string]generator.MetricGenerator
	metricProducers map[string]metricproducer.MetricExporter
	emitters        []metricemitter.Emitter
	duration        time.Duration
	from            time.Duration
}

func NewScript() *Script {
	return &Script{
		actions:         []scriptaction.ScriptAction{},
		generators:      map[string]generator.MetricGenerator{},
		metricProducers: map[string]metricproducer.MetricExporter{},
	}
}

func (s *Script) AddAction(action scriptaction.ScriptAction) {
	s.actions = append(s.actions, action)
}

func (s *Script) AddEmitter(emitter metricemitter.Emitter) {
	s.emitters = append(s.emitters, emitter)
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
		return strings.Compare(a.Name, b.Name)
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
			s.generators[action.Name] = g
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
				g, ok := rscript.generators[action.Name]
				if !ok {
					return fmt.Errorf("metric generator not found: %s", action.Name)
				}
				err := g.Reconfigure(action.At, action.Spec)
				if err != nil {
					return fmt.Errorf("error reconfiguring metric generator: %s", action.Name)
				}
			case "metric":
				if producer, ok := rscript.metricProducers[action.Name]; ok {
					if err := producer.Reconfigure(rscript.generators, action.Spec); err != nil {
						return fmt.Errorf("error reconfiguring metric exporter: %s", action.Name)
					}
				}
				metric, err := metricproducer.CreateMetricExporter(rscript.generators, action.Name, action)
				if err != nil {
					return fmt.Errorf("error creating metric exporter: %v", err)
				}
				rscript.metricProducers[action.Name] = metric
			case "disableMetric":
				if producer, ok := rscript.metricProducers[action.Name]; ok {
					producer.Disable()
				} else {
					return fmt.Errorf("metric producer not found: %s", action.Name)
				}
			case "enableMetric":
				if producer, ok := rscript.metricProducers[action.Name]; ok {
					producer.Enable()
				} else {
					return fmt.Errorf("metric producer not found: %s", action.Name)
				}
			}
		}
	}

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
		err := producer.Emit(rscript.generators, rs, mb)
		if err != nil {
			return fmt.Errorf("error emitting metric: %s", name)
		}
	}
	md := mb.Build()

	if rs.Tick >= rscript.from {
		for _, emitter := range rscript.emitters {
			if err := emitter.Emit(ctx, rs, md); err != nil {
				return fmt.Errorf("error emitting metric: %w", err)
			}
		}
	}

	return nil
}
