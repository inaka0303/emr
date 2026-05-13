#!/usr/bin/env bash
# Warm up ACI-JP-Cardio experiment AI attempts without recording metrics.
#
# Default target is the normal backend on :8080. Override EMR_URL for a test
# instance, e.g. EMR_URL=http://localhost:18080 ./scripts/warmup_experiment.sh

set -euo pipefail

EMR_URL="${EMR_URL:-http://localhost:8080}"
WARMUP_SOAP_DRAFT="${WARMUP_SOAP_DRAFT:-1}"

# One AI attempt per case C1-C8.
ATTEMPTS=(A01 A05 A03 A07 A17 A21 A19 A23)

if ! curl -fsS "$EMR_URL/health" >/dev/null; then
  echo "ERROR: EMR backend is not reachable: $EMR_URL" >&2
  exit 1
fi

for ATTEMPT_ID in "${ATTEMPTS[@]}"; do
  ATTEMPT_JSON="$(curl -fsS "$EMR_URL/api/experiment/attempts/$ATTEMPT_ID")"
  ENCOUNTER_ID="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["encounter_id"])' <<<"$ATTEMPT_JSON")"
  CASE_ID="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["case_id"])' <<<"$ATTEMPT_JSON")"
  INTERVENTION="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["intervention"])' <<<"$ATTEMPT_JSON")"

  if [ "$INTERVENTION" != "ai" ]; then
    echo "  [skip] $ATTEMPT_ID ($CASE_ID): intervention=$INTERVENTION"
    continue
  fi

  INTERVIEW="$(curl -fsS "$EMR_URL/api/encounters/$ENCOUNTER_ID/interviews" | python3 -c '
import json, sys
arr = json.load(sys.stdin).get("data") or []
d = arr[0] if arr else {}
parts = []
if d.get("raw_text"): parts.append("【問診記録】\n" + d["raw_text"])
if d.get("medication_list"): parts.append("【お薬手帳より】\n" + d["medication_list"])
if d.get("exam_findings"): parts.append("【診察所見メモ】\n" + d["exam_findings"])
if d.get("lab_results"): parts.append("【検査結果】\n" + d["lab_results"])
print("\n\n".join(parts))
')"

  PAYLOAD="$(ENCOUNTER_ID="$ENCOUNTER_ID" python3 -c '
import json, os, sys
print(json.dumps({
  "text": "症状",
  "context": "soap_subjective",
  "encounter_id": int(os.environ["ENCOUNTER_ID"]),
  "interview_text": sys.stdin.read(),
}, ensure_ascii=False))
' <<<"$INTERVIEW")"

  START="$(date +%s%3N)"
  curl -fsS -X POST "$EMR_URL/api/slm/autocomplete" \
    -H "Content-Type: application/json" \
    -H "X-Experiment-Attempt: $ATTEMPT_ID" \
    -H "X-Experiment-Warmup: true" \
    -d "$PAYLOAD" >/dev/null || true

  if [ "$WARMUP_SOAP_DRAFT" = "1" ]; then
    curl -fsS -N --max-time 180 -X POST "$EMR_URL/api/encounters/$ENCOUNTER_ID/soap-draft/stream" \
      -H "Content-Type: application/json" \
      -H "X-Experiment-Attempt: $ATTEMPT_ID" \
      -H "X-Experiment-Warmup: true" \
      -d '{"force":false}' >/dev/null || true
  fi

  ELAPSED="$(( $(date +%s%3N) - START ))"
  echo "  [ok] $ATTEMPT_ID ($CASE_ID, encounter=$ENCOUNTER_ID) warmed in ${ELAPSED}ms"
done

echo "experiment warmup complete"
