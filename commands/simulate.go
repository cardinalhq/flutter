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
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cardinalhq/flutter/internal/config"
	"github.com/cardinalhq/flutter/internal/script"
)

var (
	// these will hold all --config and --timeline values
	configPaths   []string
	timelineFiles []string
	dumpConfig    bool
	dryrun        bool
	from          time.Duration
	emitJson      bool
)

func init() {
	// --config / -c can be specified multiple times
	SimulateCmd.Flags().
		StringArrayVarP(&configPaths, "config", "c", nil, "Configuration file(s) to load (repeatable)")

	// --timeline / -t can be specified multiple times
	SimulateCmd.Flags().
		StringArrayVarP(&timelineFiles, "timeline", "t", nil, "Timeline file(s) to parse (repeatable)")

	// --dump-config will dump the merged config to stdout
	SimulateCmd.Flags().
		BoolVar(&dumpConfig, "dump-config", false, "Dump the merged config to stdout and exit")

	// --dryrun will not actually run the simulation
	SimulateCmd.Flags().
		BoolVar(&dryrun, "dryrun", false, "Do not actually run the simulation")

	// --from will set the start time for the simulation
	SimulateCmd.Flags().
		DurationVar(&from, "from", 0, "Start time for the simulation (default: now)")

		// --json will show the output timeline in JSON format
	SimulateCmd.Flags().
		BoolVar(&emitJson, "json", false, "Dump the timeline in JSON format")

	// require at least one config
	_ = SimulateCmd.MarkFlagRequired("config")
}

var SimulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate a load test",
	Long:  `Simulate a load test using the provided configuration and optional timeline files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSimulate(configPaths, timelineFiles)
	},
}

func runSimulate(configs, timelines []string) error {
	if len(configs) == 0 {
		return errors.New("no --config files provided")
	}

	// load and merge all config files in order
	cfg, err := config.LoadConfigs(configs)
	if err != nil {
		return fmt.Errorf("error loading config files: %w", err)
	}

	for _, timeline := range timelines {
		slog.Info("Loading timeline file", "file", timeline)
		b, err := os.ReadFile(timeline)
		if err != nil {
			return fmt.Errorf("error reading timeline file %q: %w", timeline, err)
		}
		tl, err := script.ParseTimeline(b)
		if err != nil {
			return fmt.Errorf("error parsing timeline file %q: %w", timeline, err)
		}
		if err := tl.MergeIntoConfig(cfg); err != nil {
			return fmt.Errorf("error merging timeline into config: %w", err)
		}
	}

	if dumpConfig {
		b, err := config.MarshalYAML(cfg)
		if err != nil {
			return fmt.Errorf("error marshalling config to YAML: %w", err)
		}
		fmt.Println(string(b))
		return nil
	}

	cfg.Dryrun = cfg.Dryrun || dryrun
	cfg.DumpJSON = emitJson

	return script.Simulate(cfg, from)
}
