package metricemitter

import (
	"context"
	"fmt"
	"io"

	"github.com/cardinalhq/flutter/pkg/state"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type DebugMetricEmitter struct {
	out io.Writer
}

func NewDebugMetricEmitter(out io.Writer) *DebugMetricEmitter {
	return &DebugMetricEmitter{
		out: out,
	}
}

func (e *DebugMetricEmitter) Emit(_ context.Context, rs *state.RunState, md pmetric.Metrics) error {
	if md.DataPointCount() == 0 {
		return nil
	}

	marshaller := pmetric.JSONMarshaler{}

	msgBody, err := marshaller.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	_, err = e.out.Write(msgBody)
	if err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}
	_, err = e.out.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}
