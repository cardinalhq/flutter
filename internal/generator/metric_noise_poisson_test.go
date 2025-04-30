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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cardinalhq/flutter/internal/state"
)

func TestSamplePoisson(t *testing.T) {
	r := state.MakeRNG(123)

	tests := []struct {
		name     string
		lambda   float64
		expected func(float64) bool
	}{
		{
			name:   "ZeroLambda",
			lambda: 0,
			expected: func(result float64) bool {
				return result == 0
			},
		},
		{
			name:   "SmallLambda",
			lambda: 5,
			expected: func(result float64) bool {
				return result >= 0 // Poisson values are non-negative
			},
		},
		{
			name:   "LargeLambda",
			lambda: 50,
			expected: func(result float64) bool {
				return result >= 0 // Poisson values are non-negative
			},
		},
		{
			name:   "NegativeLambda",
			lambda: -10,
			expected: func(result float64) bool {
				return result == 0 // Negative λ should return 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := samplePoisson(tt.lambda, r)
			assert.Equal(t, result, math.Round(result), "samplePoisson should return an integer")
		})
	}
}

func TestSamplePoissonDistribution(t *testing.T) {
	r := state.MakeRNG(123)
	lambda := 10.0
	samples := 100000
	var sum float64

	for range samples {
		sum += samplePoisson(lambda, r)
	}

	mean := sum / float64(samples)
	if math.Abs(mean-lambda) > 0.5 {
		t.Errorf("samplePoisson mean = %v, expected approximately %v", mean, lambda)
	}
}
