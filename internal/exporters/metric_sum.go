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
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/generator"
	"github.com/cardinalhq/flutter/internal/state"
)

type MetricSumSpec struct {
	MetricExporterSpec `mapstructure:",squash"`
}

type MetricSum struct {
	spec MetricSumSpec
}

var _ MetricExporter = (*MetricSum)(nil)

func NewMetricSum(generators map[string]generator.MetricGenerator, name string, to time.Duration, spec map[string]any) (*MetricSum, error) {
	SumSpec := MetricSumSpec{
		MetricExporterSpec: MetricExporterSpec{
			Frequency: DefaultFrequency,
			Name:      name,
			To:        to,
		},
	}
	if name == "" {
		return nil, errors.New("invalid metric name: " + name)
	}

	decoder, err := config.NewMapstructureDecoder(&SumSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("unable to decode MetricSumSpec for %q: %w", name, err)
	}

	if len(SumSpec.Generators) == 0 {
		return nil, errors.New("no generators specified for metric sum: " + name)
	}
	for _, generatorName := range SumSpec.Generators {
		if _, ok := generators[generatorName]; !ok {
			return nil, errors.New("unknown generator: " + generatorName)
		}
	}

	return &MetricSum{
		spec: SumSpec,
	}, nil
}

func (m *MetricSum) Reconfigure(generators map[string]generator.MetricGenerator, spec map[string]any) error {
	if err := mapstructure.Decode(spec, &m.spec); err != nil {
		return err
	}
	for _, generatorName := range m.spec.Generators {
		if _, ok := generators[generatorName]; !ok {
			return errors.New("unknown generator: " + generatorName)
		}
	}
	return nil
}

func shouldEmitMetric(state *state.RunState, to time.Duration) bool {
	return to == 0 || state.Now <= to
}

func (m *MetricSum) Emit(generators map[string]generator.MetricGenerator, state *state.RunState, mb *signalbuilder.MetricsBuilder) error {
	if !shouldEmitMetric(state, m.spec.To) {
		return nil
	}
	if state.Now < m.spec.lastEmitted+m.spec.Frequency {
		return nil
	}
	m.spec.lastEmitted = state.Now

	value, err := calculateValue(generators, m.spec.Generators, state)
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

	mm, err := s.Metric(m.spec.Name, "unit", pmetric.MetricTypeSum)
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}

	dattr := pcommon.NewMap()
	if err := dattr.FromRaw(m.spec.Attributes.Datapoint); err != nil {
		return fmt.Errorf("failed to create datapoint attributes: %w", err)
	}

	dp, _, _ := mm.Datapoint(dattr, pcommon.NewTimestampFromTime(state.Wallclock))
	dp.SetDoubleValue(value)

	return nil
}
