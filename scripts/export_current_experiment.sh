#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-/tmp/emr-exp-run/ehr-demo.db}"
EXPORT_DIR="${EXPORT_DIR:-/home/junkanki/naka/emr/exports}"
REFERENCE_PATH="${REFERENCE_PATH:-/data2/junkanki/naka/exports/experiment_references.jsonl}"

mkdir -p "$EXPORT_DIR"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ ! -f "$REFERENCE_PATH" ]]; then
  "$SCRIPT_DIR/build_experiment_references.py" --output "$REFERENCE_PATH"
fi

"$SCRIPT_DIR/export_experiment_results.py" \
  --db "$DB_PATH" \
  --output "$EXPORT_DIR/experiment_results_current.xlsx" \
  --csv-dir "$EXPORT_DIR/latest_csv"

"$SCRIPT_DIR/score_experiment_results.py" \
  --db "$DB_PATH" \
  --references "$REFERENCE_PATH" \
  --output "$EXPORT_DIR/experiment_scores_current.xlsx"

echo "$EXPORT_DIR/experiment_results_current.xlsx"
echo "$EXPORT_DIR/experiment_scores_current.xlsx"
