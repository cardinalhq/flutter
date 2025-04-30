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

package exporters

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/emitter"
	"github.com/cardinalhq/flutter/internal/state"
	"github.com/cardinalhq/oteltools/signalbuilder"
	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type MetricGaugeSpec struct {
	MetricExporterSpec `mapstructure:",squash"`
}

type MetricGauge struct {
	spec MetricGaugeSpec
}

var _ MetricExporter = (*MetricGauge)(nil)

func NewMetricGauge(emitters map[string]emitter.MetricEmitter, name string, spec map[string]any) (*MetricGauge, error) {
	gaugeSpec := MetricGaugeSpec{
		MetricExporterSpec: MetricExporterSpec{
			Frequency: 10 * time.Second,
			Name:      name,
		},
	}
	if name == "" {
		return nil, errors.New("invalid metric name: " + name)
	}

	decoder, err := config.NewMapstructureDecoder(&gaugeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(spec); err != nil {
		// if there are unknown fields, err will mention them
		return nil, fmt.Errorf("unable to decode MetricGaugeSpec for %q: %w", name, err)
	}

	if len(gaugeSpec.Emitters) == 0 {
		return nil, errors.New("no emitters specified for metric gauge: " + name)
	}
	for _, emitterName := range gaugeSpec.Emitters {
		if _, ok := emitters[emitterName]; !ok {
			return nil, errors.New("unknown emitter: " + emitterName)
		}
	}

	return &MetricGauge{
		spec: gaugeSpec,
	}, nil
}

func (m *MetricGauge) Reconfigure(emitters map[string]emitter.MetricEmitter, spec map[string]any) error {
	if err := mapstructure.Decode(spec, &m.spec); err != nil {
		return err
	}
	for _, emitterName := range m.spec.Emitters {
		if _, ok := emitters[emitterName]; !ok {
			return errors.New("unknown emitter: " + emitterName)
		}
	}
	return nil
}

func (m *MetricGauge) Emit(emitters map[string]emitter.MetricEmitter, state *state.RunState, mb *signalbuilder.MetricsBuilder) error {
	if state.Now < m.spec.lastEmitted+m.spec.Frequency {
		return nil
	}
	m.spec.lastEmitted = state.Now

	value, err := calculateValue(emitters, m.spec.Emitters, state)
	if err != nil {
		return err
	}

	rattr := pcommon.NewMap()
	if err := rattr.FromRaw(m.spec.Attributes.Resource); err != nil {
		return fmt.Errorf("failed to create resource attributes: %w", err)
	}
	r := mb.Resource(rattr)

	sattr := pcommon.NewMap()
	if err := sattr.FromRaw(m.spec.Attributes.Scope); err != nil {
		return fmt.Errorf("failed to create scope attributes: %w", err)
	}
	s := r.Scope(sattr)

	mm, err := s.Metric(m.spec.Name, "unit", pmetric.MetricTypeGauge)
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}

	dattr := pcommon.NewMap()
	if err := dattr.FromRaw(m.spec.Attributes.Datapoint); err != nil {
		return fmt.Errorf("failed to create datapoint attributes: %w", err)
	}

	dp, _, _ := mm.Datapoint(dattr, pcommon.NewTimestampFromTime(state.Wallclock))
	dp.SetDoubleValue(value)

	slog.Info("MetricGauge Emit", slog.Duration("ts", state.Now), slog.String("metricName", m.spec.Name), slog.Float64("value", value))

	return nil
}
