# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Flutter is a Go CLI tool for load testing and telemetry simulation against OpenTelemetry collectors. It reads YAML configs defining metric generators, producers, and trace producers, then emits telemetry via OTLP/HTTP.

## Build & Development Commands

```bash
make all          # Build and test
make local        # Build binary to bin/flutter (fast, no tests)
make test         # Generate + race-detected tests
make test-only    # Just tests with race detector: go test -race ./...
make lint         # golangci-lint (config: .golangci.yaml)
make check        # Pre-commit: test + license-check + lint
make license-check # Verify Apache 2.0 headers on all Go files
make images       # Multi-arch Docker images via goreleaser
```

Run a single test:
```bash
go test -race -run TestName ./pkg/generator/
```

## Architecture

**CLI Layer** (`cmd/flutter/`, `commands/`): Cobra-based CLI. Main command is `flutter simulate` which takes `-c` config files and `-t` timeline files.

**Core Simulation Flow**: Config → Script → ScriptActions (timed events) → Generators produce values → Producers format as OTel metrics/traces → Emitters send output.

**Key Interfaces** (all in `pkg/`):
- **MetricGenerator** (`generator/`): Value generators — Constant, Ramp, RandomWalk, GaussianNoise, SpikyNoise
- **MetricProducer** (`metricproducer/`): Formats metrics — Gauge, Sum
- **TraceProducer** (`traceproducer/`): Generates spans/traces
- **Emitter** (`emitter/`): Output destinations — OTLP, JSON, Debug, Ticker

**Supporting Packages**:
- `config/` — YAML config loading, custom `Duration` type
- `script/` — Orchestration, manages generators/producers/emitters, runs simulation loop
- `state/` — `RunState` carries tick, wallclock, seed-based RNG (`math/rand/v2` PCG) per step
- `scriptaction/` — Timed action definitions (at, to, type, spec)
- `brokenwing/` — Custom error types
- `timeline/` — Timeline file parsing

## Conventions

- All Go files must have Apache License 2.0 headers (enforced by `license-eye`)
- Struct tags: `mapstructure`, `yaml`, `json` for multi-format config support
- Seed-based RNG for reproducible simulations
- Table-driven tests with `testify/assert`
- Docker: Alpine-based, non-root (UID 2000), port 8080
- CI publishes to `public.ecr.aws/cardinalhq.io/flutter`
