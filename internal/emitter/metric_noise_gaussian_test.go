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
	"testing"
)

func TestCalcStdDev(t *testing.T) {
	tests := []struct {
		name      string
		desired   float64
		variation float64
		expected  float64
	}{
		{
			name:      "Positive desired value",
			desired:   2.0,
			variation: 6.0,
			expected:  2.0,
		},
		{
			name:      "Negative desired value, variation used",
			desired:   -1.0,
			variation: 6.0,
			expected:  2.0, // variation / 3
		},
		{
			name:      "Zero desired value, value used",
			desired:   0.0,
			variation: 9.0,
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcStdDev(tt.desired, tt.variation)
			if result != tt.expected {
				t.Errorf("calcStdDev(%f, %f) = %f; want %f", tt.desired, tt.variation, result, tt.expected)
			}
		})
	}
}
