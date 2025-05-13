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
	"bytes"
	"fmt"
	"slices"
	"strconv"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/script"
)

type Timeline struct {
	Metrics []Metric `json:"metrics"`
	Traces  []Trace  `json:"traces,omitempty"`
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

type Trace struct {
	Ref      string         `json:"ref"`
	Name     string         `json:"name"`
	Exemplar Span           `json:"exemplar"`
	Variants []TraceVariant `json:"variants"`
}

type Span struct {
	Ref                string          `json:"ref"`
	Name               string          `json:"name"`
	Kind               string          `json:"kind"`
	StartTs            config.Duration `json:"start_ts"`
	Duration           config.Duration `json:"duration"`
	Error              bool            `json:"error"`
	ResourceAttributes map[string]any  `json:"resourceAttributes"`
	Attributes         map[string]any  `json:"attributes"`
	Children           []Span          `json:"children"`
}

type TraceVariant struct {
	Ref       string                   `json:"ref"`
	Name      string                   `json:"name"`
	Timeline  []Segment                `json:"timeline"`
	Overrides map[string]TraceOverride `json:"overrides,omitempty"`
}

type TraceOverride struct {
	Duration   *config.Duration `json:"duration,omitempty"`
	Error      *bool            `json:"error,omitempty"`
	Attributes map[string]any   `json:"attributes,omitempty"`
}

func ParseTimeline(b []byte) (*Timeline, error) {
	var timeline Timeline
	if err := config.JSONDecode(bytes.NewReader(b), &timeline); err != nil {
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
	for _, trace := range t.Traces {
		if err := mergeTrace(rs, trace); err != nil {
			return err
		}
	}
	return nil
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
