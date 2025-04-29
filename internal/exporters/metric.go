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

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/emitter"
	"github.com/cardinalhq/flutter/internal/state"
	"github.com/cardinalhq/oteltools/signalbuilder"
)

type MetricExporter interface {
	Emit(emitters map[string]emitter.MetricEmitter, state *state.RunState, mb *signalbuilder.MetricsBuilder) error
	Reconfigure(emitters map[string]emitter.MetricEmitter, spec map[string]any) error
}

type Attributes struct {
	Resource  map[string]any `mapstructure:"resource" yaml:"resource" json:"resource"`
	Scope     map[string]any `mapstructure:"scope" yaml:"scope" json:"scope"`
	Datapoint map[string]any `mapstructure:"datapoint" yaml:"datapoint" json:"datapoint"`
}

type MetricExporterSpec struct {
	Attributes Attributes    `mapstructure:"attributes" yaml:"attributes" json:"attributes"`
	Emitters   []string      `mapstructure:"emitters" yaml:"emitters" json:"emitters"`
	Frequency  time.Duration `mapstructure:"frequency" yaml:"frequency" json:"frequency"`
	Type       string        `mapstructure:"type" yaml:"type" json:"type"`

	name        string
	lastEmitted time.Duration
}

func CreateMetricExporter(emitters map[string]emitter.MetricEmitter, name string, mes config.ScriptAction) (MetricExporter, error) {
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
		return NewMetricGauge(emitters, name, mes.Spec)
	default:
		return nil, errors.New("unknown metric exporter type: " + mes.Type)
	}
}

func calculateValue(emitters map[string]emitter.MetricEmitter, names []string, state *state.RunState) (float64, error) {
	value := 0.0
	for _, emitterName := range names {
		if _, ok := emitters[emitterName]; !ok {
			return 0, errors.New("unknown emitter: " + emitterName)
		}
		value = emitters[emitterName].Emit(state, value)
	}
	return value, nil
}
