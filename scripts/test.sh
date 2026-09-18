#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Validating development scripts"
bash -n "$ROOT_DIR/scripts/setup.sh" "$ROOT_DIR/scripts/test.sh" "$ROOT_DIR/scripts/dev.sh"

echo
echo "Checking the development environment"
"$ROOT_DIR/scripts/setup.sh" --check

echo
echo "Testing the Go API"
(cd "$ROOT_DIR" && go test ./...)

echo
echo "Building the Electron client"
(cd "$ROOT_DIR/client" && npm run build)

echo
echo "All checks passed."
