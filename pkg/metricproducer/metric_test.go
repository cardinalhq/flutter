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
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cardinalhq/flutter/pkg/state"
)

func TestEmitDueToTo(t *testing.T) {
	tests := []struct {
		name         string
		spec         MetricProducerSpec
		runState     state.RunState
		expectedEmit bool
	}{
		{
			name: "Should emit when 'To' is zero",
			spec: MetricProducerSpec{
				To: 0,
			},
			runState: state.RunState{
				Tick: 10 * time.Second,
			},
			expectedEmit: true,
		},
		{
			name: "Should emit when 'Tick' is less than or equal to 'To'",
			spec: MetricProducerSpec{
				To: 10 * time.Second,
			},
			runState: state.RunState{
				Tick: 10 * time.Second,
			},
			expectedEmit: true,
		},
		{
			name: "Should not emit when 'Tick' is greater than 'To'",
			spec: MetricProducerSpec{
				To: 10 * time.Second,
			},
			runState: state.RunState{
				Tick: 15 * time.Second,
			},
			expectedEmit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.emitDueToTo(&tt.runState)
			assert.Equal(t, tt.expectedEmit, result)
		})
	}
}

func TestEmitDueToFrequency(t *testing.T) {
	tests := []struct {
		name         string
		spec         MetricProducerSpec
		runState     state.RunState
		expectedEmit bool
	}{
		{
			name: "Should emit when 'Tick' is greater than or equal to 'lastEmitted + Frequency'",
			spec: MetricProducerSpec{
				Frequency:   10 * time.Second,
				lastEmitted: 5 * time.Second,
			},
			runState: state.RunState{
				Tick: 15 * time.Second,
			},
			expectedEmit: true,
		},
		{
			name: "Should not emit when 'Tick' is less than 'lastEmitted + Frequency'",
			spec: MetricProducerSpec{
				Frequency:   10 * time.Second,
				lastEmitted: 5 * time.Second,
			},
			runState: state.RunState{
				Tick: 10 * time.Second,
			},
			expectedEmit: false,
		},
		{
			name: "Should emit when 'Tick' is exactly 'lastEmitted + Frequency'",
			spec: MetricProducerSpec{
				Frequency:   10 * time.Second,
				lastEmitted: 5 * time.Second,
			},
			runState: state.RunState{
				Tick: 15 * time.Second,
			},
			expectedEmit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spec.emitDueToFrequency(&tt.runState)
			assert.Equal(t, tt.expectedEmit, result)
		})
	}
}
