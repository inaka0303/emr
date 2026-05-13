#!/usr/bin/env bash
# Verify all experiment attempts are in ready state before a pilot session.

set -euo pipefail

EMR_URL="${EMR_URL:-http://localhost:8080}"

if ! curl -fsS "$EMR_URL/health" >/dev/null; then
  echo "ERROR: EMR backend is not reachable: $EMR_URL" >&2
  exit 1
fi

curl -fsS "$EMR_URL/api/experiment/attempts" | python3 -c '
import json, sys
attempts = json.load(sys.stdin).get("data") or []
bad = [a for a in attempts if a.get("status") != "ready"]
if bad:
    print("non-ready attempts:")
    for a in bad:
        print(f"  {a.get('attempt_id')} status={a.get('status')} subject={a.get('subject_id')} case={a.get('case_id')}")
    raise SystemExit(1)
print(f"all {len(attempts)} attempts are ready")
'
