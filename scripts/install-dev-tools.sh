#!/bin/bash
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

set -euo pipefail

# Dev tool versions - update these to change versions across the project
GOLANGCI_LINT_VERSION="v1.64.8"
LICENSE_EYE_VERSION="latest"

# Project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"

echo "Installing development tools to $BIN_DIR..."

# Create bin directory if it doesn't exist
mkdir -p "$BIN_DIR"

# Install tools with pinned versions to project-local bin directory
echo "Installing golangci-lint $GOLANGCI_LINT_VERSION..."
GOBIN="$BIN_DIR" go install "github.com/golangci/golangci-lint/cmd/golangci-lint@$GOLANGCI_LINT_VERSION" || echo "Failed to install golangci-lint"

echo "Installing license-eye $LICENSE_EYE_VERSION..."
GOBIN="$BIN_DIR" go install "github.com/apache/skywalking-eyes/cmd/license-eye@$LICENSE_EYE_VERSION" || echo "Failed to install license-eye"

echo ""
echo "Installation complete. Installed tools:"
ls -la "$BIN_DIR" | grep -E "golangci-lint|license-eye" || echo "No tools found"
