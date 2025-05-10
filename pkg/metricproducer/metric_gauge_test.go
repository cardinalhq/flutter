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

package metricproducer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cardinalhq/flutter/pkg/brokenwing"
	"github.com/cardinalhq/flutter/pkg/generator"
	"github.com/cardinalhq/flutter/pkg/scriptaction"
)

func TestNewMetricGauge(t *testing.T) {
	t.Run("should return error for empty name", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{}
		_, err := NewMetricGauge(generators, "", scriptaction.ScriptAction{})
		assert.ErrorIs(t, err, brokenwing.ErrInvalidMetricName)
	})

	t.Run("should return error for failed decoder creation", func(t *testing.T) {
		// Simulate a scenario where the decoder creation fails
		// This might require mocking `config.NewMapstructureDecoder` if possible
		// Skipping implementation as it depends on mocking capabilities
	})

	t.Run("should return error for failed spec decoding", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{}
		mes := scriptaction.ScriptAction{
			Spec: map[string]any{"invalid": "spec"},
		}
		_, err := NewMetricGauge(generators, "test_metric", mes)
		assert.Error(t, err)
	})

	t.Run("should return error for no generators", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{}
		mes := scriptaction.ScriptAction{
			Spec: map[string]any{},
		}
		_, err := NewMetricGauge(generators, "test_metric", mes)
		assert.ErrorIs(t, err, brokenwing.ErrNoGenerators)
	})

	t.Run("should return error for unknown generator", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{}
		mes := scriptaction.ScriptAction{
			Spec: map[string]any{
				"Generators": []string{"unknown_generator"},
			},
		}
		_, err := NewMetricGauge(generators, "test_metric", mes)
		assert.ErrorIs(t, err, brokenwing.ErrUnknownGenerator)
	})

	t.Run("should create MetricGauge successfully", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{
			"valid_generator": nil,
		}
		mes := scriptaction.ScriptAction{
			Spec: map[string]any{
				"Generators": []string{"valid_generator"},
			},
		}
		gauge, err := NewMetricGauge(generators, "test_metric", mes)
		assert.NoError(t, err)
		assert.NotNil(t, gauge)
		assert.Equal(t, "test_metric", gauge.Name)
		assert.Equal(t, []string{"valid_generator"}, gauge.Generators)
	})
}

func TestMetricGauge_Reconfigure(t *testing.T) {
	t.Run("should return error for failed spec decoding", func(t *testing.T) {
		gauge := &MetricGauge{}
		err := gauge.Reconfigure(nil, map[string]any{"invalid": "spec"})
		assert.Error(t, err)
	})

	t.Run("should return error for unknown generator", func(t *testing.T) {
		gauge := &MetricGauge{}
		spec := map[string]any{
			"Generators": []string{"unknown_generator"},
		}
		err := gauge.Reconfigure(nil, spec)
		assert.ErrorIs(t, err, brokenwing.ErrUnknownGenerator)
	})

	t.Run("should reconfigure successfully", func(t *testing.T) {
		generators := map[string]generator.MetricGenerator{
			"valid_generator": nil,
		}
		gauge := &MetricGauge{}
		spec := map[string]any{
			"Generators": []string{"valid_generator"},
		}
		err := gauge.Reconfigure(generators, spec)
		assert.NoError(t, err)
		assert.Equal(t, []string{"valid_generator"}, gauge.Generators)
	})
}
