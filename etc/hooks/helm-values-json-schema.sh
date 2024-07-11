#!/usr/bin/env bash
set -euo pipefail

if ! helm plugin list | grep -qE '^schema'; then
  echo "info: installing schema helm plugin"
  helm plugin install https://github.com/losisin/helm-values-schema-json.git
fi

find ./charts -type f -name values.yaml | \
while IFS= read -r FILE; do
  echo "processing: ${FILE}"
  cd "$(dirname "${FILE}")"
  helm schema -input values.yaml
  cd - 1>/dev/null
done
