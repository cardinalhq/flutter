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
	"fmt"
	"time"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/state"
	"github.com/mitchellh/mapstructure"
)

// MetricRandomWalk emits an additive, mean-reverting noise term.
// On each Emit(), the internal state steps like:
//
//	x ← x + elasticity*(target - x) + Uniform(−stepSize,+stepSize)
//
// then clamps into [target−variation, target+variation].
// Emit(in) returns in + x.
type MetricRandomWalkSpec struct {
	MetricEmitterSpec `mapstructure:",squash"`
	Target            float64 `mapstructure:"target" yaml:"target" json:"target"`
	Elasticity        float64 `mapstructure:"elasticity" yaml:"elasticity" json:"elasticity"`
	StepSize          float64 `mapstructure:"stepSize" yaml:"stepSize" json:"stepSize"`
	Variation         float64 `mapstructure:"variation" yaml:"variation" json:"variation"`
}

type MetricRandomWalk struct {
	spec     MetricRandomWalkSpec
	current  float64
	min, max float64
}

var _ MetricEmitter = (*MetricRandomWalk)(nil)

func NewMetricRandomWalk(_ time.Duration, is map[string]any) (*MetricRandomWalk, error) {
	spec := MetricRandomWalkSpec{}
	decoder, err := config.NewMapstructureDecoder(&spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return nil, err
	}
	if spec.StepSize <= 0 {
		return nil, fmt.Errorf("invalid stepSize: %f", spec.StepSize)
	}
	state := MetricRandomWalk{
		spec:    spec,
		current: spec.Target,
		min:     spec.Target - spec.Variation,
		max:     spec.Target + spec.Variation,
	}
	return &state, nil
}

func (m *MetricRandomWalk) Reconfigure(_ time.Duration, is map[string]any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &m.spec,
		ErrorUnused: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return err
	}
	if m.spec.StepSize <= 0 {
		return fmt.Errorf("invalid stepSize: %f", m.spec.StepSize)
	}

	m.current = m.spec.Target
	m.min = m.spec.Target - m.spec.Variation
	m.max = m.spec.Target + m.spec.Variation

	if m.current < m.min {
		m.current = m.min
	} else if m.current > m.max {
		m.current = m.max
	}

	return nil
}

func (m *MetricRandomWalk) Emit(state *state.RunState, incoming float64) float64 {
	noise := (state.RND.Float64()*2 - 1) * m.spec.StepSize
	pull := m.spec.Elasticity * (m.spec.Target - m.current)
	m.current += pull + noise

	if m.current < m.min {
		m.current = m.min
	} else if m.current > m.max {
		m.current = m.max
	}

	return incoming + m.current
}
