# Copyright 2025 CardinalHQ, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
# Below here lies magic...x
#

all_deps := $(shell find cmd commands internal -name '*.go' | grep -v _test) Makefile

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
	go build -o $@ cmd/flutter/main.go

#
# Multi-architecture image builds
#
.PHONY: images
images: test-only goreleaser-dev

.PHONY: goreleaser-dev
goreleaser-dev:
	goreleaser release --clean

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
