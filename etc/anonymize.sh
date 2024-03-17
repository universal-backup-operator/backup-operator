#!/usr/bin/env bash
set -eo pipefail

# We could save SVG as an Optimized SVG, but that would cause loss of some project data...
# ...that is why we keep Inkscape SVG format with small crutches :)
sed -ri 's/(inkscape:export-filename=).+/\1"logo.png"/g' "$1"
