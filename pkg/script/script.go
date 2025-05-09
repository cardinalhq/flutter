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
	"errors"
	"fmt"
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

func Simulate(ctx context.Context, cfg *config.Config, script *Script, from time.Duration) error {
	if err := prepareScript(cfg, script); err != nil {
		return fmt.Errorf("error creating running config: %w", err)
	}
	script.from = from
	return run(ctx, cfg, script)
}

func prepareScript(cfg *config.Config, script *Script) error {
	if len(script.actions) == 0 {
		return errors.New("no script actions found in config")
	}

	slices.SortFunc(script.actions, func(a, b scriptaction.ScriptAction) int {
		if v := int(a.At - b.At); v != 0 {
			return v
		}
		if v := strings.Compare(a.Type, b.Type); v != 0 {
			return v
		}
		return strings.Compare(a.Name, b.Name)
	})
	if cfg.Duration == 0 {
		cfg.Duration = script.actions[len(script.actions)-1].At
	}
	if cfg.Duration < script.actions[len(script.actions)-1].At {
		return errors.New("Duration must be greater than or equal to the last script action time, or set to 0")
	}
	script.duration = cfg.Duration

	// Create the metric generators
	for _, action := range script.actions {
		switch action.Type {
		case "metricGenerator":
			g, err := generator.CreateMetricGenerator(action)
			if err != nil {
				return errors.New("Error creating metric generator: " + err.Error())
			}
			script.generators[action.Name] = g
		default:
			// Ignore other types of actions for now
		}
	}

	return nil
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
		rs.Now = time.Duration(now) * time.Second
		rs.Wallclock = cfg.WallclockStart.Add(rs.Now)
		err := tick(ctx, rscript, rs)
		if err != nil {
			return fmt.Errorf("error running script: %w", err)
		}
		if !cfg.Dryrun && rs.Now < rscript.duration {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func tick(ctx context.Context, rscript *Script, rs *state.RunState) error {
	if len(rscript.actions) > rs.CurrentAction {
		if rscript.actions[rs.CurrentAction].At <= rs.Now {
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
				_, ok := rscript.metricProducers[action.Name]
				if ok {
					return fmt.Errorf("metric exporter already exists: %s", action.Name)
				}
				metric, err := metricproducer.CreateMetricExporter(rscript.generators, action.Name, action)
				if err != nil {
					return fmt.Errorf("error creating metric exporter: %v", err)
				}
				rscript.metricProducers[action.Name] = metric
			}
		}
	}

	metricNames := make([]string, 0, len(rscript.metricProducers))
	for name := range rscript.metricProducers {
		metricNames = append(metricNames, name)
	}
	mb := signalbuilder.NewMetricsBuilder()
	for _, name := range metricNames {
		err := rscript.metricProducers[name].Emit(rscript.generators, rs, mb)
		if err != nil {
			return fmt.Errorf("error emitting metric: %s", name)
		}
	}
	md := mb.Build()

	if rs.Now >= rscript.from {
		for _, emitter := range rscript.emitters {
			if err := emitter.Emit(ctx, rs, md); err != nil {
				return fmt.Errorf("error emitting metric: %w", err)
			}
		}
	}

	return nil
}
