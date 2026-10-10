#!/usr/bin/env bash
# Lint and render the chart for each tests/values scenario; check the guards fail.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART="$ROOT/charts/clipboard"
V="$ROOT/tests/values"

for f in "$V"/*.yaml; do
  echo "== $(basename "$f")"
  helm lint --quiet "$CHART" -f "$V/00-base.yaml" -f "$f"
  helm template t "$CHART" -f "$V/00-base.yaml" -f "$f" >/dev/null
done

# Must fail to render, with the given message.
must_fail() {
  local msg="$1" err
  shift
  err="$(helm template t "$CHART" "$@" 2>&1 >/dev/null)" && { echo "rendered, expected: $msg" >&2; exit 1; }
  grep -q -- "$msg" <<<"$err" || { echo "wrong error for '$msg': $err" >&2; exit 1; }
}
must_fail "config.passwords is required"
must_fail "no longer uses HULL" -f "$V/00-base.yaml" --set hull.objects.secret=x
echo "ok"
