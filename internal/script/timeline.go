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
	Name               string          `json:"name"`
	Type               string          `json:"type"`
	Frequency          config.Duration `json:"frequency,omitempty"` // optional, defaults to DefaultFrequency (10s)
	ResourceAttributes map[string]any  `json:"resourceAttributes"`
	Variants           []Variant       `json:"variants"`
}

type Variant struct {
	Attributes map[string]any `json:"attributes"`
	Timeline   []Segment      `json:"timeline"`
}

type Segment struct {
	StartTs config.Duration `json:"start_ts"`
	EndTs   config.Duration `json:"end_ts"`
	Start   float64         `json:"start,omitempty"` // optional
	Median  float64         `json:"median"`
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
		if frequency.Get() == 0 {
			frequency = config.Duration{Duration: exporters.DefaultFrequency}
		}
		generators := []string{
			id + "_noise",
		}
		for i := range len(variant.Timeline) {
			generators = append(generators, id+"_ramp_"+strconv.Itoa(i))
		}
		// Add the metric to the config
		action := config.ScriptAction{
			Name: id,
			Type: "metric",
			Spec: specToMap(exporters.MetricGaugeSpec{
				MetricExporterSpec: exporters.MetricExporterSpec{
					Name:      metric.Name,
					Type:      metric.Type,
					Frequency: frequency.Get(),
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
		dpCount := len(variant.Timeline)
		for i, dp := range variant.Timeline {
			duration := dp.EndTs.Get() - dp.StartTs.Get()
			if duration <= 0 {
				duration = time.Second
			}
			action = config.ScriptAction{
				Name: id + "_ramp_" + strconv.Itoa(i),
				Type: "metricGenerator",
				At:   dp.StartTs.Get(),
				Spec: specToMap(generator.MetricRampSpec{
					MetricGeneratorSpec: generator.MetricGeneratorSpec{
						Type: "ramp",
					},
					Start:        prevStart,
					Target:       dp.Median,
					Duration:     duration,
					PrestartZero: true,
					PostEndZero:  i > 0 && i != dpCount-1,
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
