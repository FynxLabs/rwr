#!/usr/bin/env bash
# Fail when total test coverage drops below the threshold.
#
# The threshold ratchets: it was set from the measured baseline (52.8% total,
# see openspec/changes/add-coverage-reporting/baseline.md) so the gate begins
# life green and only ever tightens deliberately.
set -euo pipefail

THRESHOLD="${COVERAGE_THRESHOLD:-50}"
profile="${1:-coverage.out}"

if [ ! -f "$profile" ]; then
    go test -coverprofile="$profile" ./... >/dev/null
fi

total="$(go tool cover -func="$profile" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"

echo "total coverage: ${total}% (threshold: ${THRESHOLD}%)"
if awk -v t="$total" -v th="$THRESHOLD" 'BEGIN { exit (t >= th) ? 0 : 1 }'; then
    exit 0
fi
echo "coverage ${total}% is below the ${THRESHOLD}% gate" >&2
exit 1
