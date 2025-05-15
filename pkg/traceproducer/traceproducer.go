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
	SetRate(at time.Duration, to time.Duration, now time.Duration, rate float64)
	SetStart(start float64)
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
	return &exemplar{
		TraceProducerSpec: spec,
		start:             spec.Rate,
	}, nil
}

type exemplar struct {
	TraceProducerSpec

	start float64
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

// intrerpolate linearly interpolates from start → target over the given duration,
// beginning at offset startAt, and evaluated at offset at.
func intrerpolate(start, target float64, startAt, now, duration time.Duration) float64 {
	if duration <= 0 {
		return target
	}
	elapsed := now - startAt
	if elapsed <= 0 {
		return start
	}
	if elapsed >= duration {
		return target
	}
	frac := float64(elapsed) / float64(duration)
	return start + (target-start)*frac
}

func (t *exemplar) Emit(rs *state.RunState, tb *signalbuilder.TracesBuilder) error {
	if t.Disabled || rs.Tick < t.At || rs.Tick > t.To {
		return nil
	}

	rate := intrerpolate(t.start, t.Rate, t.At, rs.Tick, t.To-t.At)
	if rate <= 0 {
		return nil
	}
	for range int(rate) {
		offset := rs.Wallclock.Add(-time.Second)
		offset = offset.Add(time.Duration(rs.RND.Int64N(int64(time.Second))))
		jitter0 := time.Duration(scaledKindaNormal(rs.RND)*2) * time.Millisecond
		jitter1 := time.Duration(scaledKindaNormal(rs.RND)*2) * time.Millisecond
		if err := emitSpan(offset, jitter0, jitter1, tb, t.Exemplar, randomTraceID(rs.RND), pcommon.NewSpanIDEmpty()); err != nil {
			return err
		}
	}

	return nil
}

func scaledKindaNormal(r *rand.Rand) float64 {
	const maxSigma = 3.0
	for {
		x := min(r.NormFloat64(), maxSigma)
		if x >= 0 {
			return x / maxSigma
		}
	}
}

func emitSpan(now time.Time, jitter0, jitter1 time.Duration, tb *signalbuilder.TracesBuilder, s Span, traceID pcommon.TraceID, parentSpanID pcommon.SpanID) error {
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
	scale := len(s.Children) + 1
	j0ms := jitter0 * time.Duration(scale)
	sts := stime.Add(-j0ms)
	ospan.SetStartTimestamp(pcommon.NewTimestampFromTime(sts))

	j1ms := jitter1 * time.Duration(scale)
	ets := stime.Add(s.Duration.Get() + j1ms*time.Duration(scale))
	ospan.SetEndTimestamp(pcommon.NewTimestampFromTime(ets))

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
		if err := emitSpan(now, jitter0, jitter1, tb, child, traceID, spanID); err != nil {
			return err
		}
	}

	return nil
}

func (t *exemplar) SetRate(at time.Duration, to time.Duration, now time.Duration, rate float64) {
	current := intrerpolate(t.start, t.Rate, t.At, now, t.To-t.At)
	t.start = current
	t.At = at
	t.To = to
	t.Rate = rate
}

func (t *exemplar) SetStart(start float64) {
	t.start = start
}
