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

package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Events []Event `json:"events" yaml:"events"`
}

type Event struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	At          time.Time      `json:"at" yaml:"at"`
	Type        string         `json:"type" yaml:"type"`
	Spec        map[string]any `json:"spec" yaml:"spec"`
}

func ReadConfig(file string) (*Config, error) {
	cfg := Config{}
	y, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(y, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
