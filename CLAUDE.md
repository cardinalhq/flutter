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

**Core Simulation Flow**: Config → Script → ScriptActions (sorted by `At,Type,ID`) → 1-second tick loop → Generators produce values → Producers format as OTel metrics/traces → Emitters fan out.

The tick loop in `script.run()` advances by 1 second per iteration: applies any actions whose `At <= tick`, calls `emitMetrics` then `emitTraces`, then sleeps 1s (unless dryrun). Actions are consumed sequentially from a pre-sorted slice.

**Factory Pattern**: No central registry. Both `generator.CreateMetricGenerator` and `metricproducer.CreateMetricExporter` use a `switch` on `spec["type"]` from `map[string]any`. To add a new generator or producer type, add a case to the relevant switch.

**Key Interfaces** (all in `pkg/`):
- **MetricGenerator** (`generator/`): Value generators — Constant, Ramp, RandomWalk, GaussianNoise, SpikyNoise. Generators are composable: each `Emit` takes the previous generator's output as `initial`, allowing chaining (e.g. ramp + noise).
- **MetricProducer** (`metricproducer/`): Formats metrics — Gauge, Sum. Has frequency throttling (`ShouldEmit`) and can be enabled/disabled mid-simulation.
- **TraceProducer** (`traceproducer/`): Uses an exemplar span tree + rate model (not generators). Rate interpolates linearly over `[At, To]` windows with ~10% jitter.
- **Emitter** (`emitter/`): Output destinations — OTLP (HTTP/proto, not gRPC), JSON, Debug, Ticker. All emitters receive every tick's data.

**Timeline** (`timeline/`): A higher-level declarative DSL that compiles down to ScriptActions via `Timeline.MergeIntoScript()`. Auto-generates IDs (xxhash), noise generators, and sequential ramp generators from human-readable segment definitions. Supports `--from` for historical backfill (skip emitting until a tick offset).

**Supporting Packages**:
- `config/` — YAML config loading, custom `Duration` type
- `state/` — `RunState` carries tick, wallclock, seed-based RNG (`math/rand/v2` PCG) per step
- `scriptaction/` — `ScriptAction` struct: the universal carrier with `{ID, At, To, Type, Spec}`
- `brokenwing/` — Custom error types

## Conventions

- All Go files must have Apache License 2.0 headers (enforced by `license-eye`)
- Struct tags: `mapstructure`, `yaml`, `json` for multi-format config support
- Seed-based RNG for reproducible simulations
- Table-driven tests with `testify/assert`
- Docker: Alpine-based, non-root (UID 2000), port 8080
- CI publishes to `public.ecr.aws/cardinalhq.io/flutter`
- Go tools (`license-eye`, `golangci-lint`, `goreleaser`) are managed via `//tool` directives in `go.mod`
