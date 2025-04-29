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
	"math/rand"

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
	Seed       int64   `mapstructure:"seed" yaml:"seed" json:"seed"`
	Target     float64 `mapstructure:"target" yaml:"target" json:"target"`
	Elasticity float64 `mapstructure:"elasticity" yaml:"elasticity" json:"elasticity"`
	StepSize   float64 `mapstructure:"step_size" yaml:"step_size" json:"step_size"`
	Variation  float64 `mapstructure:"variation" yaml:"variation" json:"variation"`
}

type MetricRandomWalk struct {
	spec     MetricRandomWalkSpec
	rng      *rand.Rand // seeded source
	current  float64
	min, max float64
}

var _ MetricEmitter = (*MetricRandomWalk)(nil)

func NewMetricRandomWalk(is map[string]any) (*MetricRandomWalk, error) {
	spec := MetricRandomWalkSpec{
		Seed:       0,
		Target:     0,
		Elasticity: 0,
		StepSize:   0,
	}
	if err := mapstructure.Decode(is, &spec); err != nil {
		return nil, err
	}
	state := MetricRandomWalk{
		spec:    spec,
		current: spec.Target,
		min:     spec.Target - spec.Variation,
		max:     spec.Target + spec.Variation,
	}
	state.rng = rand.New(rand.NewSource(spec.Seed))
	return &state, nil
}

func (m *MetricRandomWalk) Reconfigure(is map[string]any) error {
	if err := mapstructure.Decode(is, &m.spec); err != nil {
		return err
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

// Emit advances the mean-reverting walk, clamps it, then adds to incoming.
func (m *MetricRandomWalk) Emit(incoming float64) float64 {
	// uniform noise in [−stepSize, +stepSize]
	noise := (m.rng.Float64()*2 - 1) * m.spec.StepSize

	// mean reversion toward target
	pull := m.spec.Elasticity * (m.spec.Target - m.current)

	// step
	m.current += pull + noise

	// clamp within [min, max]
	if m.current < m.min {
		m.current = m.min
	} else if m.current > m.max {
		m.current = m.max
	}

	return incoming + m.current
}
