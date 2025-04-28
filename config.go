// This file is part of CardinalHQ, Inc.
//
// CardinalHQ, Inc. proprietary and confidential.
// Unauthorized copying, distribution, or modification of this file,
// via any medium, is strictly prohibited without prior written consent.
//
// Copyright 2025 CardinalHQ, Inc. All rights reserved.

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
