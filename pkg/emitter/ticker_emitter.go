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
	"fmt"
	"io"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/cardinalhq/flutter/pkg/state"
)

type TickerEmitter struct {
	out io.Writer
}

func NewTickerEmitter(out io.Writer) *TickerEmitter {
	return &TickerEmitter{
		out: out,
	}
}

func (e *TickerEmitter) EmitMetrics(_ context.Context, rs *state.RunState, _ pmetric.Metrics) error {
	percent := rs.Tick.Seconds() / rs.Duration.Seconds() * 100
	fmt.Fprintf(e.out, "Tick %d %.2f%% %s\r", int(rs.Tick.Seconds()), percent, rs.Wallclock.Format("2006-01-02 15:04:05"))
	return nil
}

func (e *TickerEmitter) EmitTraces(_ context.Context, rs *state.RunState, _ ptrace.Traces) error {
	percent := rs.Tick.Seconds() / rs.Duration.Seconds() * 100
	fmt.Fprintf(e.out, "Tick %d %.2f%% %s\r", int(rs.Tick.Seconds()), percent, rs.Wallclock.Format("2006-01-02 15:04:05"))
	return nil
}
