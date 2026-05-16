#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-/tmp/emr-exp-run/ehr-demo.db}"
EXPORT_DIR="${EXPORT_DIR:-/home/junkanki/naka/emr/exports}"

mkdir -p "$EXPORT_DIR"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REFERENCE_PATH="${REFERENCE_PATH:-$SCRIPT_DIR/../references/experiment_references.jsonl}"

if [[ ! -f "$REFERENCE_PATH" ]]; then
  "$SCRIPT_DIR/build_experiment_references.py" --output "$REFERENCE_PATH"
  "$SCRIPT_DIR/curate_experiment_references.py" --input "$REFERENCE_PATH" --output "$REFERENCE_PATH"
fi
"$SCRIPT_DIR/validate_experiment_references.py" "$REFERENCE_PATH"

"$SCRIPT_DIR/export_experiment_results.py" \
  --db "$DB_PATH" \
  --output "$EXPORT_DIR/experiment_results_current.xlsx" \
  --csv-dir "$EXPORT_DIR/latest_csv"

"$SCRIPT_DIR/score_experiment_results.py" \
  --db "$DB_PATH" \
  --references "$REFERENCE_PATH" \
  --output "$EXPORT_DIR/experiment_scores_current.xlsx" \
  --with-bertscore \
  --strict-text-deps \
  --environment-json "$EXPORT_DIR/scoring_environment_current.json"

"$SCRIPT_DIR/build_judge_prompts.py" \
  --db "$DB_PATH" \
  --references "$REFERENCE_PATH" \
  --output "$EXPORT_DIR/judge_prompts_current.jsonl"

if [[ -f "$EXPORT_DIR/judge_results_current.jsonl" ]]; then
  "$SCRIPT_DIR/merge_judge_results.py" \
    --scores-xlsx "$EXPORT_DIR/experiment_scores_current.xlsx" \
    --judge-results "$EXPORT_DIR/judge_results_current.jsonl" \
    --output "$EXPORT_DIR/experiment_scores_with_judge_current.xlsx"
fi

echo "$EXPORT_DIR/experiment_results_current.xlsx"
echo "$EXPORT_DIR/experiment_scores_current.xlsx"
echo "$EXPORT_DIR/scoring_environment_current.json"
echo "$EXPORT_DIR/judge_prompts_current.jsonl"
if [[ -f "$EXPORT_DIR/experiment_scores_with_judge_current.xlsx" ]]; then
  echo "$EXPORT_DIR/experiment_scores_with_judge_current.xlsx"
fi
