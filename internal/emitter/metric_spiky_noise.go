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

	"slices"

	"github.com/mitchellh/mapstructure"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/state"
)

var validSpikyDirs = []string{"positive", "negative", "both"}

// MetricSpikyNoiseSpec configures a mostly‐zero emitter that randomly
// spikes with Poisson‐distributed counts when “ON”.
type MetricSpikyNoiseSpec struct {
	MetricEmitterSpec `mapstructure:",squash"`

	// PStart: chance per interval to transition from OFF→ON (0–1).
	PStart float64 `mapstructure:"pStart" yaml:"pStart" json:"pStart"`
	// PEnd:   chance per interval to transition from ON→OFF (0–1).
	PEnd float64 `mapstructure:"pEnd"   yaml:"pEnd"   json:"pEnd"`
	// PeakTarget: mean count while in a spike.
	PeakTarget float64 `mapstructure:"peakTarget" yaml:"peakTarget" json:"peakTarget"`
	// Variation: max ± deviation around PeakTarget (clamped).
	Variation float64 `mapstructure:"variation"  yaml:"variation"  json:"variation"`
	// Direction: "positive" (default), "negative", or "both"
	Direction string `mapstructure:"direction"  yaml:"direction"  json:"direction"`
}

type MetricSpikyNoise struct {
	spec    MetricSpikyNoiseSpec
	spiking bool
}

var _ MetricEmitter = (*MetricSpikyNoise)(nil)

func NewMetricSpikyNoise(_ time.Duration, is map[string]any) (*MetricSpikyNoise, error) {
	spec := MetricSpikyNoiseSpec{
		PStart:     0.02,
		PEnd:       0.20,
		PeakTarget: 10,
		Variation:  3,
		Direction:  "positive",
	}
	decoder, err := config.NewMapstructureDecoder(&spec)
	if err != nil {
		return nil, fmt.Errorf("spiky: failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return nil, err
	}
	if spec.PStart < 0 || spec.PStart > 1 {
		return nil, fmt.Errorf("spiky: invalid pStart %v", spec.PStart)
	}
	if spec.PEnd < 0 || spec.PEnd > 1 {
		return nil, fmt.Errorf("spiky: invalid pEnd %v", spec.PEnd)
	}
	if spec.Variation < 0 {
		return nil, fmt.Errorf("spiky: invalid variation %v", spec.Variation)
	}
	if !slices.Contains(validSpikyDirs, spec.Direction) {
		return nil, fmt.Errorf("spiky: invalid direction %q", spec.Direction)
	}
	return &MetricSpikyNoise{spec: spec}, nil
}

func (m *MetricSpikyNoise) Reconfigure(_ time.Duration, is map[string]any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &m.spec,
		ErrorUnused: true,
	})
	if err != nil {
		return fmt.Errorf("spiky: failed to create decoder: %w", err)
	}
	if err := decoder.Decode(is); err != nil {
		return err
	}
	// same validation as New…
	if m.spec.PStart < 0 || m.spec.PStart > 1 {
		return fmt.Errorf("spiky: invalid pStart %v", m.spec.PStart)
	}
	if m.spec.PEnd < 0 || m.spec.PEnd > 1 {
		return fmt.Errorf("spiky: invalid pEnd %v", m.spec.PEnd)
	}
	if m.spec.Variation < 0 {
		return fmt.Errorf("spiky: invalid variation %v", m.spec.Variation)
	}
	if !slices.Contains(validSpikyDirs, m.spec.Direction) {
		return fmt.Errorf("spiky: invalid direction %q", m.spec.Direction)
	}
	// you may choose to reset m.spiking = false here if desired
	return nil
}

func (m *MetricSpikyNoise) Emit(st *state.RunState, incoming float64) float64 {
	// OFF→ON?
	if !m.spiking && st.RND.Float64() < m.spec.PStart {
		m.spiking = true
	}

	var noise float64
	if m.spiking {
		// sample Poisson spike
		noise = samplePoisson(m.spec.PeakTarget, st.RND)

		// clamp [0 … PeakTarget+Variation]
		max := m.spec.PeakTarget + m.spec.Variation
		if noise < 0 {
			noise = 0
		} else if noise > max {
			noise = max
		}

		// directionality
		switch m.spec.Direction {
		case "positive":
			if noise < 0 {
				noise = -noise
			}
		case "negative":
			if noise > 0 {
				noise = -noise
			}
		}

		// ON→OFF?
		if st.RND.Float64() < m.spec.PEnd {
			m.spiking = false
		}
	} else {
		// OFF state: baseline = 0
		noise = 0
	}

	// add to incoming baseline (usually 0)
	return incoming + noise
}
