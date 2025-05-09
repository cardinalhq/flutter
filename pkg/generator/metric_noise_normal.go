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
	"slices"
	"time"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/state"
)

// MetricNormalNoise emits independent normal noise centered on Target.
// On each Emit(), it samples:
//
//	x ~ Normal(Target, StdDev²)
//
// then clamps x into [Target-Variation, Target+Variation].
// Emit(in) returns in + x.
type MetricNormalNoiseSpec struct {
	MetricGeneratorSpec `mapstructure:",squash"`

	// Target is the mean around which Normal noise is drawn.
	Target float64 `mapstructure:"target" yaml:"target" json:"target"`
	// StdDev is the standard deviation of the normal distribution.
	StdDev float64 `mapstructure:"stdDev"   yaml:"stdDev"   json:"stdDev"`
	// Variation is the allowed max deviation from Target (for clamping).
	Variation float64 `mapstructure:"variation" yaml:"variation" json:"variation"`
	// Direction is the direction of the noise. Can be "positive", "negative", or
	// "both".  "both" is the default.
	Direction string `mapstructure:"direction" yaml:"direction" json:"direction"`
}

type MetricNormalNoise struct {
	spec   MetricNormalNoiseSpec
	stdDev float64
}

var _ MetricGenerator = (*MetricNormalNoise)(nil)

var validNormalDirs = []string{"positive", "negative", "both"}

func NewMetricNormalNoise(_ time.Duration, is map[string]any) (*MetricNormalNoise, error) {
	spec := MetricNormalNoiseSpec{
		StdDev:    -1,
		Direction: "both",
	}

	decoder, err := config.NewMapstructureDecoder(&spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return nil, err
	}

	if spec.Variation < 0 {
		return nil, fmt.Errorf("invalid variation: %f", spec.Variation)
	}

	if !slices.Contains(validNormalDirs, spec.Direction) {
		return nil, fmt.Errorf("invalid direction: %s", spec.Direction)
	}

	m := &MetricNormalNoise{
		spec:   spec,
		stdDev: calcStdDev(spec.StdDev, spec.Variation),
	}
	return m, nil
}

func calcStdDev(desired, variation float64) float64 {
	stdDev := desired
	if stdDev < 0 {
		stdDev = variation / 3
	}
	return stdDev
}

func (m *MetricNormalNoise) Reconfigure(_ time.Duration, is map[string]any) error {
	decoder, err := config.NewMapstructureDecoder(&m.spec)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return err
	}
	if m.spec.Variation < 0 {
		return fmt.Errorf("invalid variation: %f", m.spec.Variation)
	}

	if !slices.Contains(validNormalDirs, m.spec.Direction) {
		return fmt.Errorf("invalid direction: %s", m.spec.Direction)
	}

	m.stdDev = calcStdDev(m.spec.StdDev, m.spec.Variation)
	return nil
}

func (m *MetricNormalNoise) Emit(st *state.RunState, incoming float64) float64 {
	sample := getNormalSample(st, m.spec, m.stdDev)
	return incoming + sample
}

// getNormalNoise returns a noise sample drawn from the distribution,
// using truncated-normal rejection sampling for directional modes.
func getNormalNoise(st *state.RunState, spec MetricNormalNoiseSpec, stdDev float64) float64 {
	if spec.Direction == "both" {
		return st.RND.NormFloat64() * stdDev
	}

	if stdDev <= 0 {
		return 0
	}

	var noise float64
	for {
		noise = st.RND.NormFloat64() * stdDev
		if spec.Direction == "positive" && noise >= 0 {
			break
		}
		if spec.Direction == "negative" && noise <= 0 {
			break
		}
	}
	return noise
}

// getNormalSample adds Target and then clamps to [Target±Variation].
func getNormalSample(st *state.RunState, spec MetricNormalNoiseSpec, stdDev float64) float64 {
	noise := getNormalNoise(st, spec, stdDev)
	sample := spec.Target + noise

	// clamp to [Target–Variation, Target+Variation]
	if sample < spec.Target-spec.Variation {
		sample = spec.Target - spec.Variation
	} else if sample > spec.Target+spec.Variation {
		sample = spec.Target + spec.Variation
	}

	return sample
}
