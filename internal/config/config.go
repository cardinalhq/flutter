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

package config

import (
	"log/slog"
	"os"
	"time"

	"maps"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Seed            uint64          `mapstructure:"seed" yaml:"seed" json:"seed"`
	WallclockStart  time.Time       `mapstructure:"wallclockStart" yaml:"wallclockStart" json:"wallclockStart"`
	Duration        time.Duration   `mapstructure:"duration" yaml:"duration" json:"duration"`
	Script          []ScriptAction  `mapstructure:"script" yaml:"script" json:"script"`
	Dryrun          bool            `mapstructure:"dryrun" yaml:"dryrun" json:"dryrun"`
	OTLPDestination OTLPDestination `mapstructure:"otlpDestination" yaml:"otlpDestination" json:"otlpDestination"`
}

type OTLPDestination struct {
	Endpoint string            `mapstructure:"endpoint" yaml:"endpoint" json:"endpoint"`
	Headers  map[string]string `mapstructure:"headers" yaml:"headers" json:"headers"`
	Timeout  time.Duration     `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
}

type ScriptAction struct {
	At   time.Duration  `mapstructure:"at" yaml:"at" json:"at"`
	Name string         `mapstructure:"name" yaml:"name" json:"name"`
	Type string         `mapstructure:"type" yaml:"type" json:"type"`
	Spec map[string]any `mapstructure:"spec" yaml:"spec" json:"spec"`
}

func LoadConfigs(fnames []string) (*Config, error) {
	merged := &Config{
		OTLPDestination: OTLPDestination{
			Timeout: 5 * time.Second,
		},
	}
	for _, fname := range fnames {
		slog.Info("Loading config", "file", fname)
		config, err := loadConfig(fname)
		if err != nil {
			return nil, err
		}
		if !config.WallclockStart.IsZero() {
			merged.WallclockStart = config.WallclockStart
		}
		if config.Dryrun {
			merged.Dryrun = true
		}
		if config.Seed != 0 {
			merged.Seed = config.Seed
		}
		if config.Duration != 0 {
			merged.Duration = config.Duration
		}
		if config.OTLPDestination.Timeout != 0 {
			merged.OTLPDestination.Timeout = config.OTLPDestination.Timeout
		}
		if config.OTLPDestination.Endpoint != "" {
			merged.OTLPDestination.Endpoint = config.OTLPDestination.Endpoint
		}
		if config.OTLPDestination.Headers != nil {
			if merged.OTLPDestination.Headers == nil {
				merged.OTLPDestination.Headers = make(map[string]string)
			}
			maps.Copy(merged.OTLPDestination.Headers, config.OTLPDestination.Headers)
		}
		merged.Script = append(merged.Script, config.Script...)
	}
	return merged, nil
}

func loadConfig(fname string) (*Config, error) {
	var config Config
	if err := LoadYAML(fname, &config); err != nil {
		return nil, err
	}
	if config.OTLPDestination.Timeout == 0 {
		config.OTLPDestination.Timeout = 5 * time.Second
	}
	return &config, nil
}

func LoadYAML(fname string, config *Config) error {
	b, err := os.ReadFile(fname)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, config)
}

func MarshalYAML(config *Config) ([]byte, error) {
	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	return b, nil
}
