# This file is part of CardinalHQ, Inc.
#
# CardinalHQ, Inc. proprietary and confidential.
# Unauthorized copying, distribution, or modification of this file,
# via any medium, is strictly prohibited without prior written consent.
#
# Copyright 2025 CardinalHQ, Inc. All rights reserved.

TARGETS=test local
PLATFORM=linux/amd64,linux/arm64
BUILDX=docker buildx build --pull --platform ${PLATFORM}
IMAGE_PREFIX=033263751764.dkr.ecr.us-east-2.amazonaws.com/cardinalhq/
IMAGE_TAG=latest-dev

#
# Build targets.  Adding to these will cause magic to occur.
#

# These are targets for "make local"
BINARIES = flutter

# These are the targets for Docker images, used both for the multi-arch and
# single (local) Docker builds.
# Dockerfiles should have a target that ends in -image, e.g. agent-image.
IMAGE_TARGETS = flutter

#
# Below here lies magic...
#

all_deps := $(shell find main.go cmd internal -name '*.go' | grep -v _test) Makefile

#
# Default target.
#

.PHONY: all
all: ${TARGETS}

#
# Generate all the things.
#
generate: ${all_deps}

#
# Run pre-commit checks
#
check: test license-check lint

license-check:
	go tool license-eye header check

lint:
	go tool golangci-lint run --timeout 15m --config .golangci.yaml

#
# Build locally, mostly for development speed.
#

.PHONY: local
local: $(addprefix bin/,$(BINARIES))

bin/flutter: ${all_deps}
	@[ -d bin ] || mkdir bin
	go build -o $@

#
# Multi-architecture image builds
#
.PHONY: images
images: test-only goreleaser-dev

.PHONY: goreleaser-dev
goreleaser-dev:
	go tool goreleaser release --clean

#
# Test targets
#

.PHONY: test
test: generate test-only

.PHONY: test-only
test-only:
	go test -race ./...

#
# Clean the world.
#

.PHONY: clean
clean:
	rm -f bin/*

.PHONY: really-clean
really-clean: clean
	rm -f ${rl_deps}
