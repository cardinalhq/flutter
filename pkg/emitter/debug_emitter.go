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

package emitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/cardinalhq/flutter/pkg/state"
)

type DebugEmitter struct {
	out io.Writer
}

func NewDebugEmitter(out io.Writer) *DebugEmitter {
	return &DebugEmitter{
		out: out,
	}
}

type DebugMessage struct {
	Now      string    `json:"now"`
	Walltime time.Time `json:"walltime"`
	Metrics  any       `json:"metrics,omitempty"`
	Traces   any       `json:"traces,omitempty"`
}

func (e *DebugEmitter) EmitMetrics(_ context.Context, rs *state.RunState, md pmetric.Metrics) error {
	if md.DataPointCount() == 0 {
		return nil
	}

	marshaller := pmetric.JSONMarshaler{}

	msgBody, err := marshaller.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal otel metric payload: %w", err)
	}

	var anyBody any
	if err := json.Unmarshal(msgBody, &anyBody); err != nil {
		return fmt.Errorf("failed to unmarshal otel metric payload: %w", err)
	}

	msg := DebugMessage{
		Now:      rs.Tick.String(),
		Walltime: rs.Wallclock,
		Metrics:  anyBody,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}
	_, _ = e.out.Write(b)
	_, _ = e.out.Write([]byte("\n"))

	return nil
}

func (e *DebugEmitter) EmitTraces(_ context.Context, rs *state.RunState, td ptrace.Traces) error {
	if td.SpanCount() == 0 {
		return nil
	}

	marshaller := ptrace.JSONMarshaler{}

	msgBody, err := marshaller.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal otel metric payload: %w", err)
	}

	var anyBody any
	if err := json.Unmarshal(msgBody, &anyBody); err != nil {
		return fmt.Errorf("failed to unmarshal otel metric payload: %w", err)
	}

	msg := DebugMessage{
		Now:      rs.Tick.String(),
		Walltime: rs.Wallclock,
		Traces:   anyBody,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}
	_, _ = e.out.Write(b)
	_, _ = e.out.Write([]byte("\n"))

	return nil
}
