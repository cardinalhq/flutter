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
	"fmt"
	"maps"
	"time"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/script"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
)

func mergeTrace(rs *script.Script, trace Trace) error {
	for _, variant := range trace.Variants {
		if len(variant.Timeline) == 0 {
			return fmt.Errorf("no segments for trace %s", trace.Name)
		}

		id := makeTraceID(trace, variant)
		firstAt := variant.Timeline[0].StartTs.Get()
		lastAt := variant.Timeline[len(variant.Timeline)-1].EndTs.Get()

		span := duplicateSpans(trace.Exemplar, variant)
		if err := addTraceToConfig(rs, id, span, firstAt, lastAt); err != nil {
			return err
		}

		// if err := addTraceTimelineToScript(rs, id, variant.Timeline); err != nil {
		// 	return err
		// }
	}
	return nil
}

func makeTraceID(trace Trace, variant TraceVariant) string {
	return fmt.Sprintf("%s-%s", trace.Name, variant.Name)
}

type TraceProducerSpec struct {
	At       config.Duration `mapstructure:"at,omitempty" yaml:"at,omitempty" json:"at,omitempty"`
	To       config.Duration `mapstructure:"to,omitempty" yaml:"to,omitempty" json:"to,omitempty"`
	Rate     float64         `mapstructure:"rate,omitempty" yaml:"rate,omitempty" json:"rate,omitempty"`
	Exemplar Span            `mapstructure:"exemplar,omitempty" yaml:"exemplar,omitempty" json:"exemplar,omitempty"`
	Disabled bool            `mapstructure:"disabled,omitempty" yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

func duplicateSpans(span Span, variant TraceVariant) Span {
	spanCopy := span
	spanCopy.ResourceAttributes = make(map[string]any)
	maps.Copy(spanCopy.ResourceAttributes, span.ResourceAttributes)
	spanCopy.Attributes = make(map[string]any)
	maps.Copy(spanCopy.Attributes, span.Attributes)

	if override, ok := variant.Overrides[span.Ref]; ok {
		applySpanOverride(&spanCopy, override)
	}

	spanCopy.Children = make([]Span, len(span.Children))
	for i, child := range span.Children {
		spanCopy.Children[i] = duplicateSpans(child, variant)
	}
	return spanCopy
}

func applySpanOverride(span *Span, override SpanOverride) {
	if override.Duration != nil {
		span.Duration = *override.Duration
	}
	if override.Error != nil {
		span.Error = *override.Error
	}
	if override.Attributes != nil {
		span.Attributes = ApplyMap(span.Attributes, override.Attributes)
	}
}

func addTraceToConfig(rs *script.Script, id string, span Span, firstAt, endAt time.Duration, rate float64) error {
	action := scriptaction.ScriptAction{
		At:   firstAt,
		To:   endAt,
		Name: id,
		Type: "trace",
		Spec: specToMap(TraceProducerSpec{
			At:       config.DurationFromDuration(firstAt),
			To:       config.DurationFromDuration(endAt),
			Exemplar: span,
			Rate:     rate,
		}),
	}
	rs.AddAction(action)
	return nil
}
