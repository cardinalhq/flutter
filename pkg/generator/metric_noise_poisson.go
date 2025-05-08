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
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/state"
)

var validPoissonDirs = []string{"positive", "negative", "both"}

// MetricPoissonNoiseSpec drives a discrete-event noise generator.
// On each Emit(), it draws a Poisson count with mean = Target,
// then clamps it to [max(0,Target−Variation) … Target+Variation]
// and applies directionality.
type MetricPoissonNoiseSpec struct {
	MetricGeneratorSpec `mapstructure:",squash"`

	// Target is the expected events per Emit() interval.
	Target float64 `mapstructure:"target" yaml:"target" json:"target"`
	// Variation is the absolute max deviation from Target.
	Variation float64 `mapstructure:"variation" yaml:"variation" json:"variation"`
	// Direction: "positive" (default), "negative", or "both".
	Direction string `mapstructure:"direction" yaml:"direction" json:"direction"`
}

type MetricPoissonNoise struct {
	spec MetricPoissonNoiseSpec
}

var _ MetricGenerator = (*MetricPoissonNoise)(nil)

func NewMetricPoissonNoise(_ time.Duration, is map[string]any) (*MetricPoissonNoise, error) {
	spec := MetricPoissonNoiseSpec{
		Direction: "positive",
	}
	decoder, err := config.NewMapstructureDecoder(&spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return nil, err
	}

	if spec.Variation < 0 {
		return nil, fmt.Errorf("invalid variation: %v", spec.Variation)
	}
	if !slices.Contains(validPoissonDirs, spec.Direction) {
		return nil, fmt.Errorf("invalid direction: %q", spec.Direction)
	}

	return &MetricPoissonNoise{spec: spec}, nil
}

func (m *MetricPoissonNoise) Reconfigure(_ time.Duration, is map[string]any) error {
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

	if m.spec.Variation < 0 {
		return fmt.Errorf("invalid variation: %v", m.spec.Variation)
	}
	if !slices.Contains(validPoissonDirs, m.spec.Direction) {
		return fmt.Errorf("invalid direction: %q", m.spec.Direction)
	}
	return nil
}

func (m *MetricPoissonNoise) Emit(st *state.RunState, _ float64) float64 {
	λ := m.spec.Target
	sample := samplePoisson(λ, st.RND)

	// clamp to [low…high]
	low := max(λ-m.spec.Variation, 0)
	high := λ + m.spec.Variation

	switch {
	case sample < low:
		sample = low
	case sample > high:
		sample = high
	}

	// enforce directionality
	switch m.spec.Direction {
	case "positive":
		if sample < 0 {
			sample = 0
		}
	case "negative":
		if sample > 0 {
			sample = -sample
		}
	}

	return sample
}

// samplePoisson returns a Poisson(λ) variate.
// Uses Knuth’s algorithm when λ<30, otherwise a Normal approx.
func samplePoisson(λ float64, r *rand.Rand) float64 {
	if λ <= 0 {
		return 0
	}
	if λ < 30 {
		L := math.Exp(-λ)
		k, p := 0, 1.0
		for p > L {
			k++
			p *= r.Float64()
		}
		return float64(k - 1)
	}
	// approximation: N(λ,λ) rounded to nearest int
	return math.Round(r.NormFloat64()*math.Sqrt(λ) + λ)
}
