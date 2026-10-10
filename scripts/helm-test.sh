#!/usr/bin/env bash
# Lint the chart and render it for every scenario in tests/values/ (each merged
# over 00-base.yaml), then check that misconfiguration fails loudly.
#
# Usage: scripts/helm-test.sh [out-dir] [chart-dir]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$(mktemp -d)}"
CHART="${2:-$ROOT/charts/clipboard}"
VALUES="$ROOT/tests/values"

mkdir -p "$OUT"
status=0

helm lint "$CHART" -f "$VALUES/00-base.yaml" || status=1

for f in "$VALUES"/[0-9][0-9]-*.yaml; do
  name="$(basename "$f" .yaml)"
  [ "$name" = "00-base" ] && continue
  if ! helm template clipboard "$CHART" -n clipboard -f "$VALUES/00-base.yaml" -f "$f" > "$OUT/$name.yaml" 2> "$OUT/$name.err"; then
    echo "FAIL render $name: $(cat "$OUT/$name.err")" >&2
    status=1
  fi
  [ -s "$OUT/$name.err" ] || rm -f "$OUT/$name.err"
done

# Each of these must fail to render, with the message in the second field.
expect_failure() {
  local label="$1" message="$2"
  shift 2
  if err="$(helm template clipboard "$CHART" -n clipboard "$@" 2>&1 >/dev/null)"; then
    echo "FAIL $label: rendered, but should have failed" >&2
    status=1
  elif ! grep -q -- "$message" <<<"$err"; then
    echo "FAIL $label: failed with an unexpected error: $err" >&2
    status=1
  fi
}

expect_failure "no password" "config.passwords is required"
expect_failure "leftover hull values" "no longer uses HULL" -f "$VALUES/00-base.yaml" --set hull.objects.secret=x
expect_failure "route without gateway" "httpRoute.parentRef.name is required" -f "$VALUES/00-base.yaml" --set httpRoute.enabled=true

[ "$status" -eq 0 ] && echo "helm tests passed (renders in $OUT)"
exit "$status"
