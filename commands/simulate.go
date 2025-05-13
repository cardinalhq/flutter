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
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cardinalhq/flutter/pkg/config"
	"github.com/cardinalhq/flutter/pkg/emitter"
	"github.com/cardinalhq/flutter/pkg/script"
	"github.com/cardinalhq/flutter/pkg/timeline"
)

var (
	// these will hold all --config and --timeline values
	configPaths   []string
	timelineFiles []string
	dryrun        bool
	from          time.Duration
	emitJson      bool
	emitDebug     bool
	dumpActions   bool
)

func init() {
	// --config / -c can be specified multiple times
	SimulateCmd.Flags().
		StringArrayVarP(&configPaths, "config", "c", nil, "Configuration file(s) to load (repeatable)")

	// --timeline / -t can be specified multiple times
	SimulateCmd.Flags().
		StringArrayVarP(&timelineFiles, "timeline", "t", nil, "Timeline file(s) to parse (repeatable)")

	// --dryrun will not actually run the simulation
	SimulateCmd.Flags().
		BoolVar(&dryrun, "dryrun", false, "Do not actually run the simulation")

	// --from will set the start time for the simulation
	SimulateCmd.Flags().
		DurationVar(&from, "from", 0, "Start time for the simulation (default: now)")

		// --json will show the output timeline in JSON format
	SimulateCmd.Flags().
		BoolVar(&emitJson, "json", false, "Dump the timeline in JSON format")

	// --debug will show the output timeline in JSON format
	SimulateCmd.Flags().
		BoolVar(&emitDebug, "debug", false, "Dump the OpenTelemetry payloads in JSON format")

	// --dump-actions will show the actions in JSON format
	SimulateCmd.Flags().
		BoolVar(&dumpActions, "dump-actions", false, "Dump the actions in JSON format and exit")
	// --dump-metrics will show the metrics in JSON format
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
	// load and merge all config files in order
	cfg, err := config.LoadConfigs(configs)
	if err != nil {
		return fmt.Errorf("error loading config files: %w", err)
	}

	rscript := script.NewScript()
	for _, tl := range timelines {
		slog.Info("Loading timeline file", "file", tl)
		b, err := os.ReadFile(tl)
		if err != nil {
			return fmt.Errorf("error reading timeline file %q: %w", tl, err)
		}
		ptl, err := timeline.ParseTimeline(b)
		if err != nil {
			return fmt.Errorf("error parsing timeline file %q: %w", tl, err)
		}
		if err := ptl.MergeIntoScript(rscript); err != nil {
			return fmt.Errorf("error merging timeline into config: %w", err)
		}
	}

	if dumpActions {
		if err := rscript.Dump(os.Stdout); err != nil {
			return fmt.Errorf("error dumping actions: %w", err)
		}
		return nil
	}

	cfg.Dryrun = cfg.Dryrun || dryrun

	if !cfg.Dryrun {
		rscript.AddEmitter(emitter.NewTickerEmitter(os.Stdout))
	}

	if emitJson {
		rscript.AddEmitter(emitter.NewJSONMetricEmitter(os.Stdout))
	}

	if emitDebug {
		rscript.AddEmitter(emitter.NewDebugMetricEmitter(os.Stdout))
	}

	if cfg.OTLPDestination.Endpoint != "" && !cfg.Dryrun {
		slog.Info("Using OTLP destination", "endpoint", cfg.OTLPDestination.Endpoint)
		client := &http.Client{
			Timeout: cfg.OTLPDestination.Timeout,
		}
		otlp, err := emitter.NewOTLPMetricEmitter(client, cfg.OTLPDestination.Endpoint, cfg.OTLPDestination.Headers)
		if err != nil {
			return fmt.Errorf("error creating OTLP emitter: %w", err)
		}
		rscript.AddEmitter(otlp)
	}

	return script.Simulate(context.Background(), cfg, rscript, from)
}
