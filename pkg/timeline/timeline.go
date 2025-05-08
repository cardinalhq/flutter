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

package timeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/exporters"
	"github.com/cardinalhq/flutter/pkg/generator"
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
	StartTs config.Duration `json:"start_ts"` // optional on segments other than first
	EndTs   config.Duration `json:"end_ts"`
	Start   float64         `json:"start,omitempty"` // optional
	Target  float64         `json:"target"`
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
		frequency := getFrequency(metric.Frequency)
		generators := generateGeneratorIDs(id, len(variant.Timeline))
		if len(variant.Timeline) == 0 {
			return nil
		}
		firstAt := variant.Timeline[0].StartTs.Get()
		lastAt := variant.Timeline[len(variant.Timeline)-1].EndTs.Get()

		if err := addMetricToConfig(cfg, id, metric, variant, frequency, generators, firstAt, lastAt); err != nil {
			return err
		}

		if err := addNoiseGenerator(cfg, id); err != nil {
			return err
		}

		if err := addTimelineToConfig(cfg, id, variant.Timeline); err != nil {
			return err
		}
	}
	return nil
}

func getFrequency(frequency config.Duration) time.Duration {
	if frequency.Get() == 0 {
		return exporters.DefaultFrequency
	}
	return frequency.Get()
}

func generateGeneratorIDs(id string, timelineLength int) []string {
	generators := []string{id + "_noise"}
	for i := range timelineLength {
		generators = append(generators, id+"_ramp_"+strconv.Itoa(i))
	}
	return generators
}

func addMetricToConfig(cfg *config.Config, id string, metric Metric, variant Variant, frequency time.Duration, generators []string, startAt, endAt time.Duration) error {
	action := config.ScriptAction{
		At:   startAt,
		To:   endAt,
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
	return nil
}

func addNoiseGenerator(cfg *config.Config, id string) error {
	action := config.ScriptAction{
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
	return nil
}

func addTimelineToConfig(cfg *config.Config, id string, timeline []Segment) error {
	if len(timeline) == 0 {
		return nil
	}

	startAt := timeline[0].StartTs.Get()
	startValue := timeline[0].Start
	dpCount := len(timeline)
	for i, dp := range timeline {
		duration := dp.EndTs.Get() - startAt
		if duration <= 0 {
			duration = time.Second
		}
		action := config.ScriptAction{
			Name: id + "_ramp_" + strconv.Itoa(i),
			Type: "metricGenerator",
			At:   startAt,
			Spec: specToMap(generator.MetricRampSpec{
				MetricGeneratorSpec: generator.MetricGeneratorSpec{
					Type: "ramp",
				},
				Start:        startValue,
				Target:       dp.Target,
				Duration:     duration,
				PrestartZero: i != 0,
				PostEndZero:  i > 0 && i != dpCount-1,
			}),
		}
		startValue = dp.Target
		startAt = dp.EndTs.Get()
		cfg.Script = append(cfg.Script, action)
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
