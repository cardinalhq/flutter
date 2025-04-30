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
	"testing"
	"time"

	"github.com/cardinalhq/flutter/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestMetricRamp_Emit(t *testing.T) {
	tests := []struct {
		name       string
		spec       MetricRampSpec
		runState   state.RunState
		initialVal float64
		expected   float64
	}{
		{
			name: "Emit with valid progression",
			spec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			runState: state.RunState{
				Now: 5 * time.Minute,
			},
			initialVal: 0,
			expected:   50, // Halfway through the duration
		},
		{
			name: "Emit clamps to target when elapsed equals duration",
			spec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			runState: state.RunState{
				Now: 10 * time.Minute,
			},
			initialVal: 0,
			expected:   100, // Reaches target
		},
		{
			name: "Emit clamps to target when elapsed exceeds duration",
			spec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			runState: state.RunState{
				Now: 15 * time.Minute,
			},
			initialVal: 0,
			expected:   100, // Exceeds duration, clamps to target
		},
		{
			name: "Emit with reverse progression",
			spec: MetricRampSpec{
				Start:    100,
				Target:   0,
				Duration: 10 * time.Minute,
			},
			runState: state.RunState{
				Now: 5 * time.Minute,
			},
			initialVal: 0,
			expected:   50, // Halfway through the duration, reverse direction
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetricRamp{
				spec: tt.spec,
			}
			result := m.Emit(&tt.runState, tt.initialVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		start    float64
		target   float64
		startAt  time.Duration
		now      time.Duration
		duration time.Duration
		expected float64
	}{
		{
			name:     "Interpolate at start",
			start:    0,
			target:   100,
			startAt:  0,
			now:      0,
			duration: 10 * time.Minute,
			expected: 0, // At the start, value should be the start value
		},
		{
			name:     "Interpolate halfway",
			start:    0,
			target:   100,
			startAt:  0,
			now:      5 * time.Minute,
			duration: 10 * time.Minute,
			expected: 50, // Halfway through the duration
		},
		{
			name:     "Interpolate at target",
			start:    0,
			target:   100,
			startAt:  0,
			now:      10 * time.Minute,
			duration: 10 * time.Minute,
			expected: 100, // At the end, value should be the target value
		},
		{
			name:     "Interpolate beyond target",
			start:    0,
			target:   100,
			startAt:  0,
			now:      15 * time.Minute,
			duration: 10 * time.Minute,
			expected: 100, // Beyond duration, clamps to target
		},
		{
			name:     "Interpolate with reverse progression",
			start:    100,
			target:   0,
			startAt:  0,
			now:      5 * time.Minute,
			duration: 10 * time.Minute,
			expected: 50, // Halfway through, reverse direction
		},
		{
			name:     "Interpolate with zero duration",
			start:    0,
			target:   100,
			startAt:  0,
			now:      5 * time.Minute,
			duration: 0,
			expected: 100, // Zero duration, directly returns target
		},
		{
			name:     "Interpolate with negative elapsed time",
			start:    0,
			target:   100,
			startAt:  5 * time.Minute,
			now:      0,
			duration: 10 * time.Minute,
			expected: 0, // Negative elapsed time, clamps to start
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intrerpolate(tt.start, tt.target, tt.startAt, tt.now, tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricRamp_Reconfigure(t *testing.T) {
	tests := []struct {
		name          string
		initialSpec   MetricRampSpec
		initialAt     time.Duration
		newAt         time.Duration
		newConfig     map[string]any
		expectedSpec  MetricRampSpec
		expectedError string
	}{
		{
			name: "Reconfigure with valid new spec and earlier timestamp",
			initialSpec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			initialAt: 5 * time.Minute,
			newAt:     3 * time.Minute,
			newConfig: map[string]any{
				"start":    0,
				"target":   200,
				"duration": 15 * time.Minute,
			},
			expectedSpec: MetricRampSpec{
				Start:    0,
				Target:   200,
				Duration: 15 * time.Minute,
			},
			expectedError: "",
		},
		{
			name: "Reconfigure with valid new spec and later timestamp",
			initialSpec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			initialAt: 5 * time.Minute,
			newAt:     7 * time.Minute,
			newConfig: map[string]any{
				"start":    0,
				"target":   200,
				"duration": 15 * time.Minute,
			},
			expectedSpec: MetricRampSpec{
				Start:    20, // Interpolated value at 7 minutes
				Target:   200,
				Duration: 15 * time.Minute,
			},
			expectedError: "",
		},
		{
			name: "Reconfigure with invalid duration",
			initialSpec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			initialAt: 5 * time.Minute,
			newAt:     7 * time.Minute,
			newConfig: map[string]any{
				"start":    0,
				"target":   200,
				"duration": 0, // Invalid duration
			},
			expectedSpec:  MetricRampSpec{},
			expectedError: "invalid duration",
		},
		{
			name: "Reconfigure with earlier timestamp and no interpolation",
			initialSpec: MetricRampSpec{
				Start:    50,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			initialAt: 5 * time.Minute,
			newAt:     3 * time.Minute,
			newConfig: map[string]any{
				"start":    50,
				"target":   150,
				"duration": 20 * time.Minute,
			},
			expectedSpec: MetricRampSpec{
				Start:    50,
				Target:   150,
				Duration: 20 * time.Minute,
			},
			expectedError: "",
		},
		{
			name: "Reconfigure part way through the current ramp",
			initialSpec: MetricRampSpec{
				Start:    0,
				Target:   100,
				Duration: 10 * time.Minute,
			},
			initialAt: 5 * time.Minute,
			newAt:     7 * time.Minute,
			newConfig: map[string]any{
				"start":    0,
				"target":   200,
				"duration": 15 * time.Minute,
			},
			expectedSpec: MetricRampSpec{
				Start:    20, // Interpolated value at 7 minutes
				Target:   200,
				Duration: 15 * time.Minute,
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetricRamp{
				spec: tt.initialSpec,
				at:   tt.initialAt,
			}

			err := m.Reconfigure(tt.newAt, tt.newConfig)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSpec, m.spec)
				assert.Equal(t, tt.newAt, m.at)
			}
		})
	}
}
