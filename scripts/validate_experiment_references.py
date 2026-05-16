#!/usr/bin/env python3
"""Validate the fixed 8-case experiment reference JSONL."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_REFERENCE_PATH = SCRIPT_DIR.parent / "references" / "experiment_references.jsonl"
EXPECTED_CASES = [f"C{i}" for i in range(1, 9)]
REQUIRED_VITAL_KEYS = ("BP_sys", "BP_dia", "HR", "SpO2", "RR", "BT")
KEYFACT_LIST_FIELDS = (
    "diagnoses",
    "diagnoses_provisional",
    "medications_to_start",
    "medications_to_continue",
    "medications_to_stop",
    "medications_to_consider",
    "procedures",
)
FORBIDDEN_KEYFACT_PATTERNS = (
    "→",
    "薬剤 F1",
    "診断 F1",
    "バイタル一致率",
    "Opus judge",
)


def load_rows(path: Path) -> list[dict[str, Any]]:
    rows = []
    with path.open(encoding="utf-8") as f:
        for lineno, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as exc:
                raise SystemExit(f"{path}:{lineno}: invalid JSON: {exc}") from exc
    return rows


def assert_true(ok: bool, message: str) -> None:
    if not ok:
        raise SystemExit(message)


def validate(path: Path) -> None:
    rows = load_rows(path)
    case_ids = [str(row.get("case_id")) for row in rows]
    assert_true(case_ids == EXPECTED_CASES, f"case order mismatch: {case_ids}")

    for row in rows:
        case_id = row["case_id"]
        soap = row.get("reference_soap") or {}
        for section in ("S", "O", "A", "P"):
            assert_true(bool((soap.get(section) or "").strip()), f"{case_id}: empty SOAP {section}")

        key_facts = row.get("key_facts") or {}
        assert_true(key_facts.get("vitals"), f"{case_id}: empty vitals")
        for key in REQUIRED_VITAL_KEYS:
            assert_true(key in key_facts["vitals"], f"{case_id}: missing vital {key}")

        for field in KEYFACT_LIST_FIELDS:
            value = key_facts.get(field)
            assert_true(isinstance(value, list), f"{case_id}: {field} must be a list")
            for item in value:
                assert_true(isinstance(item, str) and item.strip(), f"{case_id}: empty item in {field}")
                for forbidden in FORBIDDEN_KEYFACT_PATTERNS:
                    assert_true(forbidden not in item, f"{case_id}: forbidden marker {forbidden!r} in {field}: {item}")
                assert_true(not re.search(r"\[[a-z]+\]", item), f"{case_id}: Google comment marker in {field}: {item}")

        if case_id == "C1":
            cont = "\n".join(key_facts["medications_to_continue"])
            stop = "\n".join(key_facts["medications_to_stop"])
            assert_true("テルミサルタン" in cont, "C1: telmisartan should be in continue")
            assert_true("アムロジピン" not in cont, "C1: amlodipine must not be in continue")
            assert_true("アムロジピン" in stop, "C1: amlodipine should be in stop")

    print(f"validated {len(rows)} references: {path}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("path", nargs="?", type=Path, default=DEFAULT_REFERENCE_PATH)
    args = parser.parse_args()
    validate(args.path)


if __name__ == "__main__":
    main()
