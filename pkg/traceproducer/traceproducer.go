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

package traceproducer

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/state"
)

type Attributes struct {
	Resource map[string]any `json:"resource"`
	Scope    map[string]any `json:"scope"`
	Item     map[string]any `json:"item"`
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

type TraceProducer interface {
	Emit(state *state.RunState, tb *signalbuilder.TracesBuilder) error
	SetRate(rate float64)
}

type TraceProducerSpec struct {
	At       time.Duration `mapstructure:"at,omitempty" yaml:"at,omitempty" json:"at,omitempty"`
	To       time.Duration `mapstructure:"to,omitempty" yaml:"to,omitempty" json:"to,omitempty"`
	Exemplar Span          `mapstructure:"exemplar" yaml:"exemplar" json:"exemplar"`
	Disabled bool          `mapstructure:"disabled,omitempty" yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Rate     float64       `mapstructure:"rate,omitempty" yaml:"rate,omitempty" json:"rate,omitempty"`
}

var idRNG = state.MakeRNG(0)

func NewTraceProducer(spec TraceProducerSpec) (TraceProducer, error) {
	return &exemplar{spec}, nil
}

type exemplar struct {
	TraceProducerSpec
}

func randomTraceID(r *rand.Rand) pcommon.TraceID {
	traceidBytes := make([]byte, 16)
	for i := range 16 {
		traceidBytes[i] = byte(r.IntN(256))
	}

	return pcommon.TraceID(traceidBytes)
}

func randomSpanID(r *rand.Rand) pcommon.SpanID {
	spanidBytes := make([]byte, 8)
	for i := range 8 {
		spanidBytes[i] = byte(r.IntN(256))
	}

	return pcommon.SpanID(spanidBytes)
}

func (t *exemplar) Emit(state *state.RunState, tb *signalbuilder.TracesBuilder) error {
	if t.Disabled || t.Rate == 0 {
		return nil
	}

	if state.Tick < t.At || state.Tick > t.To {
		return nil
	}

	parentSpanID := pcommon.NewSpanIDEmpty()

	for range int(t.Rate / 60) {
		traceID := randomTraceID(state.RND)
		offset := state.Wallclock.Add(-time.Second)
		offset = offset.Add(time.Duration(state.RND.Int64N(int64(time.Second))))
		if err := emitSpan(offset, tb, t.Exemplar, traceID, parentSpanID); err != nil {
			return err
		}
	}

	return nil
}

func emitSpan(now time.Time, tb *signalbuilder.TracesBuilder, s Span, traceID pcommon.TraceID, parentSpanID pcommon.SpanID) error {
	rattr := pcommon.NewMap()
	if err := rattr.FromRaw(s.ResourceAttributes); err != nil {
		return err
	}

	sattr := pcommon.NewMap()

	ospan := tb.Resource(rattr).Scope(sattr).AddSpan()

	if err := ospan.Attributes().FromRaw(s.Attributes); err != nil {
		return err
	}

	spanID := randomSpanID(idRNG)

	ospan.SetTraceID(traceID)
	ospan.SetSpanID(spanID)
	ospan.SetParentSpanID(parentSpanID)
	ospan.SetName(s.Name)
	stime := now.Add(s.StartTs.Get())
	ospan.SetStartTimestamp(pcommon.NewTimestampFromTime(stime))
	ospan.SetEndTimestamp(pcommon.NewTimestampFromTime(stime.Add(s.Duration.Get())))

	if s.Error {
		ospan.Status().SetCode(ptrace.StatusCodeError)
		ospan.Status().SetMessage("error")
	} else {
		ospan.Status().SetCode(ptrace.StatusCodeOk)
		ospan.Status().SetMessage("")
	}

	switch strings.ToLower(s.Kind) {
	case "internal":
		ospan.SetKind(ptrace.SpanKindInternal)
	case "server":
		ospan.SetKind(ptrace.SpanKindServer)
	case "client":
		ospan.SetKind(ptrace.SpanKindClient)
	case "producer":
		ospan.SetKind(ptrace.SpanKindProducer)
	case "consumer":
		ospan.SetKind(ptrace.SpanKindConsumer)
	default:
		ospan.SetKind(ptrace.SpanKindUnspecified)
	}

	for _, child := range s.Children {
		if err := emitSpan(now, tb, child, traceID, spanID); err != nil {
			return err
		}
	}

	return nil
}

func (t *exemplar) SetRate(rate float64) {
	t.Rate = rate
}
