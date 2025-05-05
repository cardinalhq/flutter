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

package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/script"
)

var SimulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate a load test",
	Long:  `Simulate a load test using the provided configuration files.`,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("no config files provided")
		}
		return Simulate(args)
	},
}

func Simulate(args []string) error {
	// load the config files in order, merging as we go
	cfg, err := config.LoadConfigs(args)
	if err != nil {
		return fmt.Errorf("error loading config files: %w", err)
	}

	return script.Simulate(cfg)

}
