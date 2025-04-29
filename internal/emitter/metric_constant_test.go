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

	"github.com/stretchr/testify/assert"
)

func TestNewMetricConstant(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		input := map[string]any{
			"value": 10.5,
		}

		metricConstant, err := NewMetricConstant(input)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)
		assert.Equal(t, 10.5, metricConstant.spec.Value)
	})

	t.Run("missing value", func(t *testing.T) {
		input := map[string]any{}

		metricConstant, err := NewMetricConstant(input)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)
		assert.Equal(t, 0.0, metricConstant.spec.Value) // Default value
	})

	t.Run("invalid value type", func(t *testing.T) {
		input := map[string]any{
			"value": "invalid",
		}

		metricConstant, err := NewMetricConstant(input)
		assert.Error(t, err)
		assert.Nil(t, metricConstant)
	})
}

func TestMetricConstant_Reconfigure(t *testing.T) {
	t.Run("valid reconfiguration", func(t *testing.T) {
		initialInput := map[string]any{
			"value": 5.0,
		}
		metricConstant, err := NewMetricConstant(initialInput)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)
		assert.Equal(t, 5.0, metricConstant.spec.Value)

		reconfigureInput := map[string]any{
			"value": 15.0,
		}
		err = metricConstant.Reconfigure(reconfigureInput)
		assert.NoError(t, err)
		assert.Equal(t, 15.0, metricConstant.spec.Value)
	})

	t.Run("missing value in reconfiguration", func(t *testing.T) {
		initialInput := map[string]any{
			"value": 5.0,
		}
		metricConstant, err := NewMetricConstant(initialInput)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)
		assert.Equal(t, 5.0, metricConstant.spec.Value)

		reconfigureInput := map[string]any{}
		err = metricConstant.Reconfigure(reconfigureInput)
		assert.NoError(t, err)
		assert.Equal(t, 5.0, metricConstant.spec.Value) // Default value
	})

	t.Run("invalid value type in reconfiguration", func(t *testing.T) {
		initialInput := map[string]any{
			"value": 5.0,
		}
		metricConstant, err := NewMetricConstant(initialInput)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)
		assert.Equal(t, 5.0, metricConstant.spec.Value)

		reconfigureInput := map[string]any{
			"value": "invalid",
		}
		err = metricConstant.Reconfigure(reconfigureInput)
		assert.Error(t, err)
		assert.Equal(t, 5.0, metricConstant.spec.Value) // Value should remain unchanged
	})
}

func TestMetricConstant_Emit(t *testing.T) {
	t.Run("emit with positive value", func(t *testing.T) {
		input := map[string]any{
			"value": 10.5,
		}
		metricConstant, err := NewMetricConstant(input)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)

		result := metricConstant.Emit(nil, 5.0)
		assert.Equal(t, 15.5, result)
	})

	t.Run("emit with zero value", func(t *testing.T) {
		input := map[string]any{
			"value": 0.0,
		}
		metricConstant, err := NewMetricConstant(input)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)

		result := metricConstant.Emit(nil, 5.0)
		assert.Equal(t, 5.0, result)
	})

	t.Run("emit with negative value", func(t *testing.T) {
		input := map[string]any{
			"value": -3.5,
		}
		metricConstant, err := NewMetricConstant(input)
		assert.NoError(t, err)
		assert.NotNil(t, metricConstant)

		result := metricConstant.Emit(nil, 5.0)
		assert.Equal(t, 1.5, result)
	})
}
