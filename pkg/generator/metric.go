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

package generator

import (
	"errors"
	"time"

	"github.com/cardinalhq/flutter/pkg/scriptaction"
	"github.com/cardinalhq/flutter/pkg/state"
)

type MetricGenerator interface {
	Emit(state *state.RunState, initial float64) float64
	Reconfigure(at time.Duration, spec map[string]any) error
}

type MetricGeneratorSpec struct {
	Type string `mapstructure:"type" yaml:"type" json:"type"`
}

func CreateMetricGenerator(mes scriptaction.ScriptAction) (MetricGenerator, error) {
	if mes.Spec == nil {
		return nil, errors.New("missing spec in metric generator")
	}
	generatorTypeAny, ok := mes.Spec["type"]
	if !ok {
		return nil, errors.New("missing type in metric generator spec")
	}
	generatorType, ok := generatorTypeAny.(string)
	if !ok {
		return nil, errors.New("type in metric generator spec is not a string")
	}
	switch generatorType {
	case "constant":
		return NewMetricConstant(mes.At, mes.Spec)
	case "normalNoise":
		return NewMetricNormalNoise(mes.At, mes.Spec)
	case "poissonNoise":
		return NewMetricPoissonNoise(mes.At, mes.Spec)
	case "randomWalk":
		return NewMetricRandomWalk(mes.At, mes.Spec)
	case "ramp":
		return NewMetricRamp(mes.At, mes.Spec)
	case "spikyNoise":
		return NewMetricSpikyNoise(mes.At, mes.Spec)
	default:
		return nil, errors.New("unknown metricGenerator type: " + generatorType)
	}
}
