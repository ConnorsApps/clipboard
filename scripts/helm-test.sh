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

# No password must fail to render.
err="$(helm template t "$CHART" 2>&1 >/dev/null)" && { echo "rendered without a password" >&2; exit 1; }
grep -q "config.passwords is required" <<<"$err" || { echo "unexpected error: $err" >&2; exit 1; }

# Helm adds a parent chart's global values to every subchart, so the closed
# schema must allow them.
helm template t "$CHART" -f "$V/00-base.yaml" --set global.example=1 >/dev/null

# values.schema.json is closed: an unknown key must fail.
err="$(helm template t "$CHART" -f "$V/00-base.yaml" --set bogus=1 2>&1 >/dev/null)" && { echo "rendered with an unknown value" >&2; exit 1; }
grep -q "values don't meet the specifications of the schema" <<<"$err" || { echo "unexpected error: $err" >&2; exit 1; }
echo "ok"
