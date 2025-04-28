// This file is part of CardinalHQ, Inc.
//
// CardinalHQ, Inc. proprietary and confidential.
// Unauthorized copying, distribution, or modification of this file,
// via any medium, is strictly prohibited without prior written consent.
//
// Copyright 2025 CardinalHQ, Inc. All rights reserved.

package main

import "time"

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
