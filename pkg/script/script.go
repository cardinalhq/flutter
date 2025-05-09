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
	Script          []scriptaction.ScriptAction
	Generators      map[string]generator.MetricGenerator
	MetricProducers map[string]metricproducer.MetricExporter
	Duration        time.Duration
}

func Simulate(ctx context.Context, cfg *config.Config, actions []scriptaction.ScriptAction, emitters []metricemitter.Emitter, from time.Duration) error {
	s, err := makeRunningConfig(cfg, actions)
	if err != nil {
		return fmt.Errorf("error creating running config: %w", err)
	}

	return run(ctx, cfg, s, emitters, from)
}

func makeRunningConfig(cfg *config.Config, actions []scriptaction.ScriptAction) (*Script, error) {
	if len(actions) == 0 {
		return nil, errors.New("no script actions found in config")
	}

	rc := Script{
		Script:          actions,
		Generators:      make(map[string]generator.MetricGenerator),
		MetricProducers: make(map[string]metricproducer.MetricExporter),
	}

	slices.SortFunc(rc.Script, func(a, b scriptaction.ScriptAction) int {
		if v := int(a.At - b.At); v != 0 {
			return v
		}
		if v := strings.Compare(a.Type, b.Type); v != 0 {
			return v
		}
		return strings.Compare(a.Name, b.Name)
	})
	if cfg.Duration == 0 {
		cfg.Duration = rc.Script[len(rc.Script)-1].At
	}
	if cfg.Duration < rc.Script[len(rc.Script)-1].At {
		return nil, errors.New("Duration must be greater than or equal to the last script action time, or set to 0")
	}
	rc.Duration = cfg.Duration

	// Create the metric generators
	for _, action := range rc.Script {
		switch action.Type {
		case "metricGenerator":
			g, err := generator.CreateMetricGenerator(action)
			if err != nil {
				return nil, errors.New("Error creating metric generator: " + err.Error())
			}
			rc.Generators[action.Name] = g
		default:
			// Ignore other types of actions for now
		}
	}

	return &rc, nil
}

func run(ctx context.Context, cfg *config.Config, rc *Script, emitters []metricemitter.Emitter, from time.Duration) error {
	seed := cfg.Seed
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	rs := &state.RunState{
		Duration: rc.Duration,
		RND:      state.MakeRNG(seed),
	}

	starttime := cfg.WallclockStart
	if starttime.IsZero() {
		starttime = time.Now()
	}
	seconds := int64(rs.Duration.Seconds())
	for now := range seconds + 1 {
		rs.Now = time.Duration(now) * time.Second
		rs.Wallclock = starttime.Add(rs.Now)
		if len(rc.Script) > rs.CurrentAction {
			if rc.Script[rs.CurrentAction].At <= rs.Now {
				action := rc.Script[rs.CurrentAction]
				rs.CurrentAction++
				switch action.Type {
				case "metricGenerator":
					g, ok := rc.Generators[action.Name]
					if !ok {
						return fmt.Errorf("metric generator not found: %s", action.Name)
					}
					err := g.Reconfigure(action.At, action.Spec)
					if err != nil {
						return fmt.Errorf("error reconfiguring metric generator: %s", action.Name)
					}
				case "metric":
					_, ok := rc.MetricProducers[action.Name]
					if ok {
						return fmt.Errorf("metric exporter already exists: %s", action.Name)
					}
					metric, err := metricproducer.CreateMetricExporter(rc.Generators, action.Name, action)
					if err != nil {
						return fmt.Errorf("error creating metric exporter: %v", err)
					}
					rc.MetricProducers[action.Name] = metric
				}
			}
		}

		metricNames := make([]string, 0, len(rc.MetricProducers))
		for name := range rc.MetricProducers {
			metricNames = append(metricNames, name)
		}
		mb := signalbuilder.NewMetricsBuilder()
		for _, name := range metricNames {
			err := rc.MetricProducers[name].Emit(rc.Generators, rs, mb)
			if err != nil {
				return fmt.Errorf("error emitting metric: %s", name)
			}
		}
		md := mb.Build()

		if rs.Now >= from {
			for _, emitter := range emitters {
				if err := emitter.Emit(ctx, rs, md); err != nil {
					return fmt.Errorf("error emitting metric: %w", err)
				}
			}
		}

		if !cfg.Dryrun && rs.Now < rc.Duration {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}
