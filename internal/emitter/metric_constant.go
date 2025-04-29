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

import "github.com/mitchellh/mapstructure"

type MetricConstantSpec struct {
	Value float64 `mapstructure:"value" yaml:"value" json:"value"`
}

type MetricConstant struct {
	spec MetricConstantSpec
}

var _ MetricEmitter = (*MetricConstant)(nil)

func NewMetricConstant(is map[string]any) (*MetricConstant, error) {
	spec := MetricConstantSpec{
		Value: 0,
	}
	if err := mapstructure.Decode(is, &spec); err != nil {
		return nil, err
	}
	return &MetricConstant{
		spec: spec,
	}, nil
}

func (m *MetricConstant) Reconfigure(is map[string]any) error {
	return mapstructure.Decode(is, &m.spec)
}

func (m *MetricConstant) Emit(incoming float64) float64 {
	return incoming + m.spec.Value
}
