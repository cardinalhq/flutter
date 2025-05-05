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

package script

import "encoding/json"

type Timeline struct {
	Metrics []Metric `json:"metrics"`
}

type Metric struct {
	Name               string            `json:"name"`
	Type               string            `json:"type"`
	ResourceAttributes map[string]string `json:"resourceAttributes"`
	Variants           []Variant         `json:"variants"`
}

type Variant struct {
	Attributes map[string]string `json:"attributes"`
	Timeline   []DataPoint       `json:"timeline"`
}

type DataPoint struct {
	StartTs int64 `json:"start_ts"`
	EndTs   int64 `json:"end_ts"`
	Median  int   `json:"median"`
}

func ParseTimeline(b []byte) (*Timeline, error) {
	var timeline Timeline
	if err := json.Unmarshal(b, &timeline); err != nil {
		return nil, err
	}
	return &timeline, nil
}
