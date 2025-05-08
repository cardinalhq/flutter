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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/exporters"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/state"
)

type Script struct {
	Script     []config.ScriptAction
	Generators map[string]generator.MetricGenerator
	Exporters  map[string]exporters.MetricExporter
	Duration   time.Duration
}

func Simulate(cfg *config.Config, from time.Duration) error {
	s, err := makeRunningConfig(cfg)
	if err != nil {
		return fmt.Errorf("error creating running config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.OTLPDestination.Timeout,
	}

	return run(cfg, s, httpClient, from)
}

func makeRunningConfig(cfg *config.Config) (*Script, error) {
	rc := Script{
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

func run(cfg *config.Config, rc *Script, client *http.Client, from time.Duration) error {
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
		if !cfg.Dryrun && rs.Now >= from {
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
						return fmt.Errorf("error creating metric exporter: %v", err)
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

		shouldEmit := rs.Now >= from && !cfg.Dryrun

		if shouldEmit && cfg.OTLPDestination.Endpoint != "" {
			if err := sendOTLPMetric(context.Background(), client, cfg.OTLPDestination.Headers, mm, cfg.OTLPDestination.Endpoint); err != nil {
				slog.Warn("failed to send OTLP metrics", "error", err)
			}
		}

		if rs.Now >= from && cfg.DumpJSON {
			if err := dumpJSONMetric(rs, mm); err != nil {
				return fmt.Errorf("error dumping JSON metrics: %w", err)
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

type jsonWrapper struct {
	Timestamp       time.Time       `json:"timestamp"`
	MetricsProtobuf string          `json:"metricsProtobuf"`
	At              config.Duration `json:"at"`
}

func dumpJSONMetric(rs *state.RunState, md pmetric.Metrics) error {
	if md.MetricCount() == 0 {
		return nil
	}

	marshaller := pmetric.ProtoMarshaler{}

	j := jsonWrapper{
		Timestamp: rs.Wallclock,
		At:        config.Duration{Duration: rs.Now},
	}

	msgBody, err := marshaller.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	msgBody, err = gzipBytes(msgBody)
	if err != nil {
		return fmt.Errorf("failed to gzip metrics: %w", err)
	}
	j.MetricsProtobuf = base64.StdEncoding.EncodeToString(msgBody)

	jsonData, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
