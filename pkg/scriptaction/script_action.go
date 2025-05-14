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

package scriptaction

import "time"

type ScriptAction struct {
	ID   string         `mapstructure:"id" yaml:"id" json:"id"`
	At   time.Duration  `mapstructure:"at" yaml:"at" json:"at"`
	To   time.Duration  `mapstructure:"to,omitempty" yaml:"to,omitempty" json:"to,omitempty"`
	Type string         `mapstructure:"type" yaml:"type" json:"type"`
	Spec map[string]any `mapstructure:"spec" yaml:"spec" json:"spec"`
}
