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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/emitter"
	"github.com/cardinalhq/flutter/internal/exporters"
	"github.com/cardinalhq/flutter/internal/state"
)

type RunConfig struct {
	Script         []config.ScriptAction
	MetricEmitters map[string]emitter.MetricEmitter
	Metrics        map[string]exporters.MetricExporter
	Duration       int64
}

func main() {
	if len(os.Args) < 2 {
		panic("Usage: flutter <config_file>")
	}

	// load the config files in order, merging as we go
	cfg, err := config.LoadConfigs(os.Args[1:])
	if err != nil {
		panic(err)
	}

	runConfig, err := makeRunningConfig(cfg)
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.OTLPDestination.Timeout),
	}

	run(cfg, runConfig, httpClient)
}

func makeRunningConfig(cfg *config.Config) (*RunConfig, error) {
	rc := RunConfig{
		Script:         cfg.Script,
		MetricEmitters: make(map[string]emitter.MetricEmitter),
		Metrics:        make(map[string]exporters.MetricExporter),
	}

	if len(cfg.Script) == 0 {
		panic("No script actions found in config")
	}
	slices.SortFunc(rc.Script, func(a, b config.ScriptAction) int {
		if a.At != b.At {
			return int(a.At - b.At)
		}
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.Name, b.Name)
	})
	if cfg.Duration == 0 {
		cfg.Duration = rc.Script[len(rc.Script)-1].At
	}
	if cfg.Duration < rc.Script[len(rc.Script)-1].At {
		panic("Duration must be greater than last script action time")
	}
	rc.Duration = cfg.Duration

	// Create the metric emitters
	for _, action := range rc.Script {
		switch action.Type {
		case "metricEmitter":
			metricEmitter, err := emitter.CreateMetricEmitter(action)
			if err != nil {
				panic("Error creating metric emitter: " + err.Error())
			}
			rc.MetricEmitters[action.Name] = metricEmitter
		default:
			// Ignore other types of actions for now
		}
	}

	return &rc, nil
}

func run(cfg *config.Config, rc *RunConfig, client *http.Client) {
	seed := cfg.Seed
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	rs := &state.RunState{
		Duration: rc.Duration,
		RND:      rand.New(rand.NewPCG(seed, seed)),
	}

	starttime := cfg.WallclockStart
	if starttime.IsZero() {
		starttime = time.Now()
	}
	for now := range rs.Duration {
		rs.Now = now
		rs.Wallclock = starttime.Add(time.Second * time.Duration(now))
		if !cfg.Dryrun {
			fmt.Printf("TICK! %d, %s\r", now, rs.Wallclock.Format(time.RFC3339))
		}
		//slog.Info("Running at time", slog.Int64("time", now), slog.Int("currentAction", rs.CurrentAction), slog.Int("scriptLength", len(rc.Script)))
		if len(rc.Script) > rs.CurrentAction {
			if rc.Script[rs.CurrentAction].At <= now {
				action := rc.Script[rs.CurrentAction]
				rs.CurrentAction++
				switch action.Type {
				case "metricEmitter":
					metricEmitter, ok := rc.MetricEmitters[action.Name]
					if !ok {
						panic("Metric emitter not found: " + action.Name)
					}
					err := metricEmitter.Reconfigure(action.Spec)
					if err != nil {
						panic("Error reconfiguring metric emitter: " + err.Error())
					}
				case "metric":
					metric, ok := rc.Metrics[action.Name]
					if ok {
						panic("Metric already exists: " + action.Name)
					}
					metric, err := exporters.CreateMetricExporter(rc.MetricEmitters, action.Name, action)
					if err != nil {
						panic("Error creating metric exporter: " + err.Error())
					}
					rc.Metrics[action.Name] = metric
				}
			}
		}

		metricNames := make([]string, 0, len(rc.Metrics))
		for name := range rc.Metrics {
			metricNames = append(metricNames, name)
		}
		mb := signalbuilder.NewMetricsBuilder()
		for _, name := range metricNames {
			err := rc.Metrics[name].Emit(rc.MetricEmitters, rs, mb)
			if err != nil {
				panic("Error emitting metric: " + err.Error())
			}
		}
		mm := mb.Build()

		if !cfg.Dryrun && cfg.OTLPDestination.Endpoint != "" {
			if err := sendOTLPMetric(context.Background(), client, cfg.OTLPDestination.Headers, mm, cfg.OTLPDestination.Endpoint); err != nil {
				panic("Error sending OTLP metric: " + err.Error())
			}
		}

		if !cfg.Dryrun && rs.Now < rc.Duration {
			time.Sleep(1 * time.Second)
		}
	}
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
