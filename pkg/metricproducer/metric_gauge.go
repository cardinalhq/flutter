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

package metricproducer

import (
	"fmt"

	"github.com/cardinalhq/oteltools/signalbuilder"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/cardinalhq/flutter/pkg/brokenwing"
	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
	"github.com/cardinalhq/flutter/pkg/state"
)

type MetricGauge struct {
	MetricProducerSpec `mapstructure:",squash" yaml:",inline" json:",inline"`
}

var _ MetricProducer = (*MetricGauge)(nil)

func NewMetricGauge(generators map[string]generator.MetricGenerator, name string, mes scriptaction.ScriptAction) (*MetricGauge, error) {
	gaugeSpec := MetricGauge{
		MetricProducerSpec: MetricProducerSpec{
			Frequency: DefaultFrequency,
			Name:      name,
			To:        mes.To,
		},
	}
	if name == "" {
		return nil, fmt.Errorf("%w: %s", brokenwing.ErrInvalidMetricName, name)
	}

	decoder, err := config.NewMapstructureDecoder(&gaugeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(mes.Spec); err != nil {
		return nil, &brokenwing.DecodeError{Name: name, Err: err}
	}

	if len(gaugeSpec.Generators) == 0 {
		return nil, fmt.Errorf("%w: %s", brokenwing.ErrNoGenerators, name)
	}
	for _, generatorName := range gaugeSpec.Generators {
		if _, ok := generators[generatorName]; !ok {
			return nil, fmt.Errorf("%w: %s", brokenwing.ErrUnknownGenerator, generatorName)
		}
	}

	return &gaugeSpec, nil
}

func (m *MetricGauge) Reconfigure(generators map[string]generator.MetricGenerator, spec map[string]any) error {
	decoder, err := config.NewMapstructureDecoder(m)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(spec); err != nil {
		return &brokenwing.DecodeError{Name: m.Name, Err: err}
	}
	for _, generatorName := range m.Generators {
		if _, ok := generators[generatorName]; !ok {
			return fmt.Errorf("%w: %s", brokenwing.ErrUnknownGenerator, generatorName)
		}
	}
	return nil
}

func (m *MetricGauge) Emit(generators map[string]generator.MetricGenerator, state *state.RunState, mb *signalbuilder.MetricsBuilder) error {
	if !m.ShouldEmit(state) {
		return nil
	}
	m.lastEmitted = state.Tick

	value, err := calculateValue(generators, m.Generators, state)
	if err != nil {
		return err
	}

	rattr := pcommon.NewMap()
	if err := rattr.FromRaw(m.Attributes.Resource); err != nil {
		return fmt.Errorf("failed to create resource attributes: %w", err)
	}
	r := mb.Resource(rattr)

	sattr := pcommon.NewMap()
	if err := sattr.FromRaw(m.Attributes.Scope); err != nil {
		return fmt.Errorf("failed to create scope attributes: %w", err)
	}
	s := r.Scope(sattr)

	mm, err := s.Metric(m.Name, "unit", pmetric.MetricTypeGauge)
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}

	dattr := pcommon.NewMap()
	if err := dattr.FromRaw(m.Attributes.Datapoint); err != nil {
		return fmt.Errorf("failed to create datapoint attributes: %w", err)
	}

	dp, _, _ := mm.Datapoint(dattr, pcommon.NewTimestampFromTime(state.Wallclock))
	dp.SetDoubleValue(value)

	return nil
}
