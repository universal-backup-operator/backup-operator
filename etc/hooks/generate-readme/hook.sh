#!/usr/bin/env bash
set -euo pipefail

SCRIPT_NAME="$0"
cd "$(dirname "$(readlink -f "${SCRIPT_NAME}")")"
go mod tidy
go run .
README="$(realpath "README.md")"
cd - 1>/dev/null
mv "${README}" README.md
