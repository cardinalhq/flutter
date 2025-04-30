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

package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/exporters"
	"github.com/cardinalhq/flutter/internal/generator"
	"github.com/cardinalhq/flutter/internal/state"
)

type RunConfig struct {
	Script     []config.ScriptAction
	Generators map[string]generator.MetricGenerator
	Exporters  map[string]exporters.MetricExporter
	Duration   time.Duration
}

var SimulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate a load test",
	Long:  `Simulate a load test using the provided configuration files.`,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("no config files provided")
		}
		return Simulate(args)
	},
}

func Simulate(args []string) error {
	// load the config files in order, merging as we go
	cfg, err := config.LoadConfigs(args)
	if err != nil {
		return fmt.Errorf("error loading config files: %w", err)
	}

	runConfig, err := makeRunningConfig(cfg)
	if err != nil {
		return fmt.Errorf("error creating running config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.OTLPDestination.Timeout,
	}

	return run(cfg, runConfig, httpClient)
}

func makeRunningConfig(cfg *config.Config) (*RunConfig, error) {
	rc := RunConfig{
		Script:     cfg.Script,
		Generators: make(map[string]generator.MetricGenerator),
		Exporters:  make(map[string]exporters.MetricExporter),
	}

	if len(cfg.Script) == 0 {
		return nil, errors.New("no script actions found in config")
	}
	slices.SortFunc(rc.Script, func(a, b config.ScriptAction) int {
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

func run(cfg *config.Config, rc *RunConfig, client *http.Client) error {
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
		if !cfg.Dryrun {
			fmt.Printf("TICK! %d, %s\r", now, rs.Wallclock.Format(time.RFC3339))
		}
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
					_, ok := rc.Exporters[action.Name]
					if ok {
						return fmt.Errorf("metric exporter already exists: %s", action.Name)
					}
					metric, err := exporters.CreateMetricExporter(rc.Generators, action.Name, action)
					if err != nil {
						return fmt.Errorf("error creating metric exporter: %s", action.Name)
					}
					rc.Exporters[action.Name] = metric
				}
			}
		}

		metricNames := make([]string, 0, len(rc.Exporters))
		for name := range rc.Exporters {
			metricNames = append(metricNames, name)
		}
		mb := signalbuilder.NewMetricsBuilder()
		for _, name := range metricNames {
			err := rc.Exporters[name].Emit(rc.Generators, rs, mb)
			if err != nil {
				return fmt.Errorf("error emitting metric: %s", name)
			}
		}
		mm := mb.Build()

		if !cfg.Dryrun && cfg.OTLPDestination.Endpoint != "" {
			if err := sendOTLPMetric(context.Background(), client, cfg.OTLPDestination.Headers, mm, cfg.OTLPDestination.Endpoint); err != nil {
				return fmt.Errorf("error sending OTLP metric: %w", err)
			}
		}

		if !cfg.Dryrun && rs.Now < rc.Duration {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

func sendOTLPMetric(ctx context.Context, client *http.Client, headers map[string]string, md pmetric.Metrics, endpoint string) error {
	if md.MetricCount() == 0 {
		return nil
	}

	req := pmetricotlp.NewExportRequestFromMetrics(md)

	body, err := req.MarshalProto()
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to protobuf: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/v1/metrics"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("collector returned %s: %s", resp.Status, string(respBody))
	}

	return nil
}
