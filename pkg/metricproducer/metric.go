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
	"errors"
	"time"

	"github.com/cardinalhq/oteltools/signalbuilder"

	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
	"github.com/cardinalhq/flutter/pkg/state"
)

type MetricProducer interface {
	MetricProducerInterface
	Emit(generators map[string]generator.MetricGenerator, state *state.RunState, mb *signalbuilder.MetricsBuilder) error
	Reconfigure(generators map[string]generator.MetricGenerator, spec map[string]any) error
}

type Attributes struct {
	Resource  map[string]any `mapstructure:"resource,omitempty" yaml:"resource,omitempty" json:"resource,omitempty"`
	Scope     map[string]any `mapstructure:"scope,omitempty" yaml:"scope,omitempty" json:"scope,omitempty"`
	Datapoint map[string]any `mapstructure:"datapoint,omitempty" yaml:"datapoint,omitempty" json:"datapoint,omitempty"`
}

type MetricProducerSpec struct {
	To         time.Duration `mapstructure:"to,omitempty" yaml:"to,omitempty" json:"to,omitempty"`
	Attributes Attributes    `mapstructure:"attributes" yaml:"attributes" json:"attributes"`
	Generators []string      `mapstructure:"generators" yaml:"generators" json:"generators"`
	Frequency  time.Duration `mapstructure:"frequency,omitempty" yaml:"frequency,omitempty" json:"frequency,omitempty"`
	Type       string        `mapstructure:"type" yaml:"type" json:"type"`
	Name       string        `mapstructure:"name" yaml:"name" json:"name"`
	Disabled   bool          `mapstructure:"disabled,omitempty" yaml:"disabled,omitempty" json:"disabled,omitempty"`

	lastEmitted time.Duration
}

type MetricProducerInterface interface {
	GetAttributes() Attributes
	ShouldEmit(state *state.RunState) bool
	Enable()
	Disable()
	IsDisabled() bool
}

func (m *MetricProducerSpec) GetAttributes() Attributes {
	return m.Attributes
}

func (m *MetricProducerSpec) ShouldEmit(state *state.RunState) bool {
	return !m.IsDisabled() && m.emitDueToFrequency(state) && m.emitDueToTo(state)
}

func (m *MetricProducerSpec) emitDueToFrequency(state *state.RunState) bool {
	return state.Tick >= m.lastEmitted+m.Frequency
}

func (m *MetricProducerSpec) emitDueToTo(state *state.RunState) bool {
	return m.To == 0 || state.Tick <= m.To
}

func (m *MetricProducerSpec) Enable() {
	m.Disabled = false
}

func (m *MetricProducerSpec) Disable() {
	m.Disabled = true
}

func (m *MetricProducerSpec) IsDisabled() bool {
	return m.Disabled
}

const (
	// DefaultFrequency is the default frequency for metric exporters.
	DefaultFrequency = 10 * time.Second
)

func CreateMetricExporter(generators map[string]generator.MetricGenerator, name string, mes scriptaction.ScriptAction) (MetricProducer, error) {
	exporterTypeAny, ok := mes.Spec["type"]
	if !ok {
		return nil, errors.New("missing type in metric exporter spec")
	}
	exporterType, ok := exporterTypeAny.(string)
	if !ok {
		return nil, errors.New("type in metric exporter spec is not a string")
	}

	switch exporterType {
	case "gauge":
		return NewMetricGauge(generators, name, mes)
	case "sum":
		return NewMetricSum(generators, name, mes)
	default:
		return nil, errors.New("unknown metric exporter type: " + exporterType)
	}
}

func calculateValue(generators map[string]generator.MetricGenerator, generatorNames []string, state *state.RunState) (float64, error) {
	value := 0.0
	for _, generatorName := range generatorNames {
		if _, ok := generators[generatorName]; !ok {
			return 0, errors.New("unknown generator: " + generatorName)
		}
		value = generators[generatorName].Emit(state, value)
	}
	return value, nil
}
