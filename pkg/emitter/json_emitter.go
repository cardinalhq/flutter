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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/cardinalhq/flutter/pkg/compression"
	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/state"
)

type JSONEmitter struct {
	out io.Writer
}

func NewJSONEmitter(out io.Writer) *JSONEmitter {
	return &JSONEmitter{
		out: out,
	}
}

type jsonWrapper struct {
	Timestamp       time.Time       `json:"timestamp"`
	MetricsProtobuf string          `json:"metricsProtobuf,omitempty"`
	TracesProtobuf  string          `json:"tracesProtobuf,omitempty"`
	At              config.Duration `json:"at"`
}

func (e *JSONEmitter) EmitMetrics(ctx context.Context, rs *state.RunState, md pmetric.Metrics) error {
	if md.DataPointCount() == 0 {
		return nil
	}

	marshaller := pmetric.ProtoMarshaler{}

	j := jsonWrapper{
		Timestamp: rs.Wallclock,
		At:        config.Duration{Duration: rs.Tick},
	}

	msgBody, err := marshaller.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	msgBody, err = compression.GZipBytes(msgBody)
	if err != nil {
		return fmt.Errorf("failed to gzip metrics: %w", err)
	}
	j.MetricsProtobuf = base64.StdEncoding.EncodeToString(msgBody)

	jsonData, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(e.out, string(jsonData))
	return nil
}

func (e *JSONEmitter) EmitTraces(ctx context.Context, rs *state.RunState, td ptrace.Traces) error {
	if td.SpanCount() == 0 {
		return nil
	}

	marshaller := ptrace.ProtoMarshaler{}

	j := jsonWrapper{
		Timestamp: rs.Wallclock,
		At:        config.Duration{Duration: rs.Tick},
	}

	msgBody, err := marshaller.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	msgBody, err = compression.GZipBytes(msgBody)
	if err != nil {
		return fmt.Errorf("failed to gzip metrics: %w", err)
	}
	j.TracesProtobuf = base64.StdEncoding.EncodeToString(msgBody)

	jsonData, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(e.out, string(jsonData))
	return nil
}
