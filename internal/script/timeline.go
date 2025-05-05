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
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/exporters"
	"github.com/cardinalhq/flutter/internal/generator"
)

type Timeline struct {
	Metrics []Metric `json:"metrics"`
}

type Metric struct {
	Name               string         `json:"name"`
	Type               string         `json:"type"`
	Frequency          time.Duration  `json:"frequency,omitempty"` // optional, defaults to DefaultFrequency (10s)
	ResourceAttributes map[string]any `json:"resourceAttributes"`
	Variants           []Variant      `json:"variants"`
}

type Variant struct {
	Attributes map[string]any `json:"attributes"`
	Timeline   []DataPoint    `json:"timeline"`
}

type DataPoint struct {
	StartTs int64   `json:"start_ts"`
	EndTs   int64   `json:"end_ts"`
	Start   float64 `json:"start,omitempty"` // optional
	Median  float64 `json:"median"`
}

func ParseTimeline(b []byte) (*Timeline, error) {
	var timeline Timeline
	if err := json.Unmarshal(b, &timeline); err != nil {
		return nil, err
	}
	return &timeline, nil
}

func (t *Timeline) MergeIntoConfig(cfg *config.Config) error {
	for _, metric := range t.Metrics {
		if err := mergeMetric(cfg, metric); err != nil {
			return err
		}
	}
	return nil
}

func mergeMetric(cfg *config.Config, metric Metric) error {
	for _, variant := range metric.Variants {
		id := makeID(metric, variant)
		frequency := metric.Frequency
		if frequency == 0 {
			frequency = exporters.DefaultFrequency
		}
		generators := []string{
			id + "_noise",
		}
		for _, dp := range variant.Timeline {
			generators = append(generators, id+"_ramp_"+strconv.FormatInt(dp.StartTs, 10))
		}
		// Add the metric to the config
		action := config.ScriptAction{
			Name: id,
			Type: "metric",
			Spec: specToMap(exporters.MetricGaugeSpec{
				MetricExporterSpec: exporters.MetricExporterSpec{
					Name:      metric.Name,
					Type:      metric.Type,
					Frequency: frequency,
					Attributes: exporters.Attributes{
						Resource:  metric.ResourceAttributes,
						Datapoint: variant.Attributes,
					},
					Generators: generators,
				},
			}),
		}
		cfg.Script = append(cfg.Script, action)

		// add a noise generator
		action = config.ScriptAction{
			Name: id + "_noise",
			Type: "metricGenerator",
			Spec: specToMap(generator.MetricGaussianNoiseSpec{
				MetricGeneratorSpec: generator.MetricGeneratorSpec{
					Type: "gaussianNoise",
				},
				Variation: 5,
				Direction: "positive",
			}),
		}
		cfg.Script = append(cfg.Script, action)

		// Add the timeline to the config
		prevStart := float64(0)
		for _, dp := range variant.Timeline {
			action = config.ScriptAction{
				Name: id + "_ramp_" + strconv.FormatInt(dp.StartTs, 10),
				Type: "metricGenerator",
				At:   time.Duration(dp.StartTs) * time.Second,
				Spec: specToMap(generator.MetricRampSpec{
					MetricGeneratorSpec: generator.MetricGeneratorSpec{
						Type: "ramp",
					},
					Start:        prevStart,
					Target:       dp.Median,
					Duration:     time.Duration(dp.EndTs-dp.StartTs) * time.Second,
					PrestartZero: true,
					PostEndZero:  true,
				}),
			}
			prevStart = dp.Median
			cfg.Script = append(cfg.Script, action)
		}
	}

	return nil
}

func makeMapID(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	id := ""
	for k := range keys {
		id += keys[k] + "=" + fmt.Sprintf("%v", m[keys[k]]) + "|"
	}
	return id
}

func makeID(metric Metric, variant Variant) string {
	id := metric.Name + "|"
	id += metric.Type + "|"
	id += makeMapID(metric.ResourceAttributes) + "|"
	id += makeMapID(variant.Attributes) + "|"

	x := xxhash.Sum64([]byte(id))
	return strconv.FormatUint(x, 32)
}

func specToMap(spec any) map[string]any {
	b, err := json.Marshal(spec)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
