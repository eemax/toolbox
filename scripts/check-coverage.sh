#!/usr/bin/env bash
set -euo pipefail

FLOOR_ENTRIES='
./internal/add:65
./internal/cli:60
./internal/manifest:70
./internal/runner:65
./internal/doctor:70
./internal/output:60
'

failures=0
while IFS=':' read -r pkg floor; do
  if [[ -z "${pkg}" ]]; then
    continue
  fi
  profile="$(mktemp)"
  go test -coverprofile "$profile" "$pkg" >/dev/null
  pct="$(go tool cover -func="$profile" | awk '/^total:/{gsub("%", "", $3); print $3}')"
  rm -f "$profile"

  printf '%s coverage: %s%% (floor: %s%%)\n' "$pkg" "$pct" "$floor"
  if awk "BEGIN { exit !($pct + 0 < $floor + 0) }"; then
    printf 'coverage floor failed for %s: got %s%%, need >= %s%%\n' "$pkg" "$pct" "$floor" >&2
    failures=1
  fi
done <<EOF
$FLOOR_ENTRIES
EOF

if [[ "$failures" -ne 0 ]]; then
  exit 1
fi
