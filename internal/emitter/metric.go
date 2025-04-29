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
	"errors"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/state"
)

type MetricEmitter interface {
	Emit(state *state.RunState, initial float64) float64
	Reconfigure(spec map[string]any) error
}

type MetricEmitterSpec struct {
	Type string `mapstructure:"type" yaml:"type" json:"type"`
}

func CreateMetricEmitter(mes config.ScriptAction) (MetricEmitter, error) {
	if mes.Spec == nil {
		return nil, errors.New("missing spec in metric emitter")
	}
	emitterTypeAny, ok := mes.Spec["type"]
	if !ok {
		return nil, errors.New("missing type in metric emitter spec")
	}
	emitterType, ok := emitterTypeAny.(string)
	if !ok {
		return nil, errors.New("type in metric emitter spec is not a string")
	}
	switch emitterType {
	case "constant":
		return NewMetricConstant(mes.Spec)
	case "randomWalk":
		return NewMetricRandomWalk(mes.Spec)
	default:
		return nil, errors.New("unknown metric emitter type: " + mes.Type)
	}
}
