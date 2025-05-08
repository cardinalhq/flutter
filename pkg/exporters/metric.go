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
	"time"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/state"
	"github.com/cardinalhq/oteltools/signalbuilder"
)

type MetricExporter interface {
	Emit(generators map[string]generator.MetricGenerator, state *state.RunState, mb *signalbuilder.MetricsBuilder) error
	Reconfigure(generators map[string]generator.MetricGenerator, spec map[string]any) error
}

type Attributes struct {
	Resource  map[string]any `mapstructure:"resource" yaml:"resource" json:"resource"`
	Scope     map[string]any `mapstructure:"scope" yaml:"scope" json:"scope"`
	Datapoint map[string]any `mapstructure:"datapoint" yaml:"datapoint" json:"datapoint"`
}

type MetricExporterSpec struct {
	To         time.Duration `mapstructure:"to" yaml:"to" json:"to"`
	Attributes Attributes    `mapstructure:"attributes" yaml:"attributes" json:"attributes"`
	Generators []string      `mapstructure:"generators" yaml:"generators" json:"generators"`
	Frequency  time.Duration `mapstructure:"frequency" yaml:"frequency" json:"frequency"`
	Type       string        `mapstructure:"type" yaml:"type" json:"type"`
	Name       string        `mapstructure:"name" yaml:"name" json:"name"`

	lastEmitted time.Duration
}

const (
	// DefaultFrequency is the default frequency for metric exporters.
	DefaultFrequency = 10 * time.Second
)

func CreateMetricExporter(generators map[string]generator.MetricGenerator, name string, mes config.ScriptAction) (MetricExporter, error) {
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
		return NewMetricGauge(generators, name, mes.To, mes.Spec)
	case "sum":
		return NewMetricSum(generators, name, mes.To, mes.Spec)
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
