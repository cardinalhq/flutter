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
	"github.com/cardinalhq/flutter/pkg/script"
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
	Type    string          `json:"type"`
	StartTs config.Duration `json:"start_ts"` // optional on segments other than first
	EndTs   config.Duration `json:"end_ts"`
	Start   *float64        `json:"start,omitempty"` // optional
	Target  float64         `json:"target"`
}

func ParseTimeline(b []byte) (*Timeline, error) {
	var timeline Timeline
	if err := json.Unmarshal(b, &timeline); err != nil {
		return nil, err
	}

	for _, metric := range timeline.Metrics {
		for _, variant := range metric.Variants {
			for i := range variant.Timeline {
				if variant.Timeline[i].Type == "" {
					variant.Timeline[i].Type = "segment"
				}
			}
		}
	}

	return &timeline, nil
}

func (t *Timeline) MergeIntoScript(rs *script.Script) error {
	for _, metric := range t.Metrics {
		if err := mergeMetric(rs, metric); err != nil {
			return err
		}
	}
	return nil
}

func mergeMetric(rs *script.Script, metric Metric) error {
	for _, variant := range metric.Variants {
		if len(variant.Timeline) == 0 {
			return fmt.Errorf("no timeline for metric %s", metric.Name)
		}

		id := makeID(metric, variant)
		frequency := getFrequency(metric.Frequency)
		generators := generateGeneratorIDs(id, variant.Timeline)
		firstAt := variant.Timeline[0].StartTs.Get()
		lastAt := time.Duration(0)
		for _, tl := range variant.Timeline {
			if tl.Type == "segment" && tl.EndTs.Get() > lastAt {
				lastAt = tl.EndTs.Get()
			}
		}
		if lastAt == 0 {
			return fmt.Errorf("lastAt is 0 for metric %s", id)
		}

		if err := addMetricToConfig(rs, id, metric, variant, frequency, generators, firstAt, lastAt); err != nil {
			return err
		}

		if err := addNoiseGenerator(rs, id); err != nil {
			return err
		}

		if err := addTimelineToScript(rs, id, variant.Timeline); err != nil {
			return err
		}
	}
	return nil
}

func getFrequency(frequency config.Duration) time.Duration {
	if frequency.Get() == 0 {
		return metricproducer.DefaultFrequency
	}
	return frequency.Get()
}

func generateGeneratorIDs(id string, timeline []Segment) []string {
	generators := []string{id + "_noise"}
	counter := 0
	for _, tl := range timeline {
		if tl.Type != "segment" {
			continue
		}
		generators = append(generators, id+"_ramp_"+strconv.Itoa(counter))
		counter++
	}
	return generators
}

func addMetricToConfig(rs *script.Script, id string, metric Metric, variant Variant, frequency time.Duration, generators []string, startAt, endAt time.Duration) error {
	action := scriptaction.ScriptAction{
		At:   startAt,
		To:   endAt,
		Name: id,
		Type: "metric",
		Spec: specToMap(metricproducer.MetricGauge{
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
	rs.AddAction(action)
	return nil
}

func addNoiseGenerator(rs *script.Script, id string) error {
	action := scriptaction.ScriptAction{
		Name: id + "_noise",
		Type: "metricGenerator",
		Spec: specToMap(generator.MetricNormalNoiseSpec{
			MetricGeneratorSpec: generator.MetricGeneratorSpec{
				Type: "normalNoise",
			},
			Variation: 5,
			Direction: "both",
			StdDev:    -1,
		}),
	}
	rs.AddAction(action)
	return nil
}

func addTimelineToScript(rs *script.Script, id string, timeline []Segment) error {
	if len(timeline) == 0 {
		return nil
	}

	startAt := timeline[0].StartTs.Get()
	startValue := 0.0
	if timeline[0].Start != nil {
		startValue = *(timeline[0].Start)
	}
	disabled := false

	rampCounter := 0

	for _, dp := range timeline {
		if dp.StartTs.Get() != 0 {
			startAt = dp.StartTs.Get()
		}
		if dp.Start != nil {
			startValue = *(dp.Start)
		}
		if dp.Type == "disable" {
			action := scriptaction.ScriptAction{
				At:   dp.StartTs.Get(),
				Name: id,
				Type: "disableMetric",
			}
			disabled = true
			rs.AddAction(action)
			continue
		}
		if dp.Type != "segment" {
			return fmt.Errorf("unknown segment type %s for metric %s", dp.Type, id)
		}
		duration := dp.EndTs.Get() - startAt
		if duration <= 0 {
			duration = time.Second
		}
		if disabled {
			action := scriptaction.ScriptAction{
				At:   startAt,
				Name: id,
				Type: "enableMetric",
			}
			rs.AddAction(action)
			disabled = false
		}
		action := scriptaction.ScriptAction{
			Name: id + "_ramp_" + strconv.Itoa(rampCounter),
			Type: "metricGenerator",
			At:   startAt,
			Spec: specToMap(generator.MetricRampSpec{
				MetricGeneratorSpec: generator.MetricGeneratorSpec{
					Type: "ramp",
				},
				Start:    startValue,
				Target:   dp.Target,
				Duration: duration,
			}),
		}
		rampCounter++
		startValue = dp.Target
		startAt = dp.EndTs.Get()
		rs.AddAction(action)
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
