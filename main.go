// This file is part of CardinalHQ, Inc.
//
// CardinalHQ, Inc. proprietary and confidential.
// Unauthorized copying, distribution, or modification of this file,
// via any medium, is strictly prohibited without prior written consent.
//
// Copyright 2025 CardinalHQ, Inc. All rights reserved.

package main

import (
	"log/slog"
	"os"
)

func main() {
	// the first command line argument is the config file
	cfg, err := ReadConfig(os.Args[1])
	if err != nil {
		panic(err)
	}

	slog.Info("Config loaded", slog.Any("config", cfg))
}
