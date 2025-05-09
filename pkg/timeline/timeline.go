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
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/metricproducer"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
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

func (t *Timeline) MergeIntoConfig(cfg *config.Config, actions []scriptaction.ScriptAction) ([]scriptaction.ScriptAction, error) {
	for _, metric := range t.Metrics {
		var err error
		actions, err = mergeMetric(cfg, actions, metric)
		if err != nil {
			return actions, err
		}
	}
	return actions, nil
}

func mergeMetric(cfg *config.Config, actions []scriptaction.ScriptAction, metric Metric) ([]scriptaction.ScriptAction, error) {
	for _, variant := range metric.Variants {
		id := makeID(metric, variant)
		frequency := getFrequency(metric.Frequency)
		generators := generateGeneratorIDs(id, len(variant.Timeline))
		if len(variant.Timeline) == 0 {
			return actions, nil
		}

		// Ensure the timeline is sorted by start time
		slices.SortFunc(variant.Timeline, func(a, b Segment) int {
			if a.StartTs.Get() < b.StartTs.Get() {
				return -1
			}
			if a.StartTs.Get() > b.StartTs.Get() {
				return 1
			}
			return 0
		})

		firstAt := variant.Timeline[0].StartTs.Get()
		lastAt := variant.Timeline[len(variant.Timeline)-1].EndTs.Get()

		var err error
		actions, err = addMetricToConfig(cfg, actions, id, metric, variant, frequency, generators, firstAt, lastAt)
		if err != nil {
			return actions, err
		}

		actions, err = addNoiseGenerator(cfg, actions, id)
		if err != nil {
			return actions, err
		}

		actions, err = addTimelineToConfig(cfg, actions, id, variant.Timeline)
		if err != nil {
			return actions, err
		}
	}
	return actions, nil
}

func getFrequency(frequency config.Duration) time.Duration {
	if frequency.Get() == 0 {
		return metricproducer.DefaultFrequency
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

func addMetricToConfig(cfg *config.Config, actions []scriptaction.ScriptAction, id string, metric Metric, variant Variant, frequency time.Duration, generators []string, startAt, endAt time.Duration) ([]scriptaction.ScriptAction, error) {
	action := scriptaction.ScriptAction{
		At:   startAt,
		To:   endAt,
		Name: id,
		Type: "metric",
		Spec: specToMap(metricproducer.MetricGaugeSpec{
			MetricProducerSpec: metricproducer.MetricProducerSpec{
				Name:      metric.Name,
				Type:      metric.Type,
				Frequency: frequency,
				Attributes: metricproducer.Attributes{
					Resource:  metric.ResourceAttributes,
					Datapoint: variant.Attributes,
				},
				Generators: generators,
			},
		}),
	}
	actions = append(actions, action)
	return actions, nil
}

func addNoiseGenerator(cfg *config.Config, actions []scriptaction.ScriptAction, id string) ([]scriptaction.ScriptAction, error) {
	action := scriptaction.ScriptAction{
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
	actions = append(actions, action)
	return actions, nil
}

func addTimelineToConfig(cfg *config.Config, actions []scriptaction.ScriptAction, id string, timeline []Segment) ([]scriptaction.ScriptAction, error) {
	if len(timeline) == 0 {
		return actions, nil
	}

	startAt := timeline[0].StartTs.Get()
	startValue := timeline[0].Start
	dpCount := len(timeline)
	for i, dp := range timeline {
		duration := dp.EndTs.Get() - startAt
		if duration <= 0 {
			duration = time.Second
		}
		action := scriptaction.ScriptAction{
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
		actions = append(actions, action)
	}
	return actions, nil
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
