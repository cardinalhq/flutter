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
	"strconv"
	"time"

	"github.com/cespare/xxhash"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/metricproducer"
	"github.com/cardinalhq/flutter/pkg/script"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
)

func mergeMetric(rs *script.Script, metric Metric) error {
	for _, variant := range metric.Variants {
		if len(variant.Timeline) == 0 {
			return fmt.Errorf("no timeline for metric %s", metric.Name)
		}

		id := makeMetricID(metric, variant)
		frequency := getMetricFrequency(metric.Frequency)
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

		if err := addMetricNoiseGenerator(rs, id); err != nil {
			return err
		}

		if err := addMetricTimelineToScript(rs, id, variant.Timeline); err != nil {
			return err
		}
	}
	return nil
}

func addMetricToConfig(rs *script.Script, id string, metric Metric, variant Variant, frequency time.Duration, generators []string, startAt, endAt time.Duration) error {
	action := scriptaction.ScriptAction{
		At:   startAt,
		To:   endAt,
		ID:   id,
		Type: "metric",
		Spec: specToMap(metricproducer.MetricGauge{
			MetricProducerSpec: metricproducer.MetricProducerSpec{
				Name:      metric.Name,
				Type:      metric.Type,
				To:        endAt,
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

func addMetricNoiseGenerator(rs *script.Script, id string) error {
	action := scriptaction.ScriptAction{
		ID:   id + "_noise",
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

func addMetricTimelineToScript(rs *script.Script, id string, timeline []Segment) error {
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

	// count the number of ramps so we can do something special with the last one
	nRamps := 0
	for _, dp := range timeline {
		if dp.Type == "segment" {
			nRamps++
		}
	}

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
				ID:   id,
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
				ID:   id,
				Type: "enableMetric",
			}
			rs.AddAction(action)
			disabled = false
		}
		action := scriptaction.ScriptAction{
			ID:   id + "_ramp_" + strconv.Itoa(rampCounter),
			Type: "metricGenerator",
			At:   startAt,
			Spec: specToMap(generator.MetricRampSpec{
				MetricGeneratorSpec: generator.MetricGeneratorSpec{
					Type: "ramp",
				},
				Start:       startValue,
				Target:      dp.Target,
				Duration:    duration,
				PostEndZero: rampCounter < nRamps-1,
			}),
		}
		rampCounter++
		startValue = dp.Target
		startAt = dp.EndTs.Get()
		rs.AddAction(action)
	}
	return nil
}

func getMetricFrequency(frequency config.Duration) time.Duration {
	if frequency.Get() == 0 {
		return metricproducer.DefaultFrequency
	}
	return frequency.Get()
}

func makeMetricID(metric Metric, variant Variant) string {
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
