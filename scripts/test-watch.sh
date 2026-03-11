#!/usr/bin/env bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "go toolchain is required for test-watch"
  exit 1
fi

last_hash=""

while true; do
  current_hash="$(
    find . -type f \
      \( -name '*.go' -o -name '*.md' -o -name '*.yaml' -o -name '*.yml' \) \
      ! -path './.git/*' \
      ! -path './bin/*' \
      -print0 | xargs -0 shasum | shasum | awk '{print $1}'
  )"

  if [[ "$current_hash" != "$last_hash" ]]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] changes detected; running make test"
    make test
    last_hash="$current_hash"
  fi

  sleep 1
done
