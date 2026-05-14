#!/usr/bin/env python3
"""Build reference JSONL for the 8-case human experiment from the Google Doc.

The Google Doc is the source of truth for the revised walk-in references. This
script extracts reference SOAP and rough key_facts used by the local scoring
runner. It intentionally writes a separate artifact and does not mutate the app
database or the upstream ACI-JP-Cardio benchmark files.
"""

from __future__ import annotations

import argparse
import json
import re
import urllib.request
from pathlib import Path
from typing import Any


DOC_EXPORT_URL = "https://docs.google.com/document/d/1zGrQEbcB4wxUiLyTpjpUCQd7WoQqxlfIXuk3Ogdm2SM/export?format=txt"

CASE_MAP = {
    "JC-AMI-A": {"case_id": "C1", "docs_no": 9},
    "JC-PE-B": {"case_id": "C2", "docs_no": 13},
    "JC-AD-A": {"case_id": "C3", "docs_no": 14},
    "JC-AHF-B": {"case_id": "C4", "docs_no": 5},
    "JC-AMI-B": {"case_id": "C5", "docs_no": 3},
    "JC-PE-A": {"case_id": "C6", "docs_no": 12},
    "JC-AD-B": {"case_id": "C7", "docs_no": 15},
    "JC-AHF-A": {"case_id": "C8", "docs_no": 20},
}


def read_doc_text(path_or_url: str) -> str:
    if path_or_url.startswith("http://") or path_or_url.startswith("https://"):
        with urllib.request.urlopen(path_or_url, timeout=30) as resp:
            return resp.read().decode("utf-8-sig")
    return Path(path_or_url).read_text(encoding="utf-8-sig")


def normalize_text(s: str) -> str:
    s = s.replace("\r\n", "\n").replace("\r", "\n")
    s = s.replace("\ufeff", "")
    return s


def split_case_sections(text: str) -> dict[str, str]:
    starts = []
    for match in re.finditer(r"(?m)^症例\s+\d+(?:\s*/\s*(?:\d+)?)?\s*[:：]?\s*(JC-[A-Z]+-[AB])\b", text):
        case_id = match.group(1)
        if case_id in CASE_MAP:
            starts.append((case_id, match.start()))
    sections: dict[str, str] = {}
    for idx, (case_id, start) in enumerate(starts):
        end = starts[idx + 1][1] if idx + 1 < len(starts) else len(text)
        sections[case_id] = text[start:end].strip()
    return sections


def extract_between(section: str, start_pattern: str, end_patterns: list[str]) -> str:
    start = re.search(start_pattern, section, flags=re.MULTILINE)
    if not start:
        return ""
    body = section[start.end():]
    end_positions = []
    for pat in end_patterns:
        m = re.search(pat, body, flags=re.MULTILINE)
        if m:
            end_positions.append(m.start())
    if end_positions:
        body = body[: min(end_positions)]
    return body.strip()


def extract_reference_soap(section: str) -> dict[str, str]:
    ref = extract_between(
        section,
        r"(?m)^■?\s*正解\s*\(reference\)\s*SOAP\s*$",
        [r"(?m)^■\s*key_facts", r"(?m)^key_facts"],
    )
    out: dict[str, str] = {}
    markers = {
        "S": r"\[S\s*\(Subjective\)\]",
        "O": r"\[O\s*\(Objective\)\]",
        "A": r"\[A\s*\(Assessment\)\]",
        "P": r"\[P\s*\(Plan\)\]",
    }
    spans = []
    for key, pat in markers.items():
        m = re.search(pat, ref)
        if m:
            spans.append((key, m.start(), m.end()))
    spans.sort(key=lambda x: x[1])
    for idx, (key, _start, content_start) in enumerate(spans):
        content_end = spans[idx + 1][1] if idx + 1 < len(spans) else len(ref)
        out[key] = ref[content_start:content_end].strip()
    return {k: out.get(k, "") for k in ("S", "O", "A", "P")}


def extract_keyfacts_text(section: str) -> str:
    return extract_between(
        section,
        r"(?m)^■\s*key_facts.*$|^key_facts.*$",
        [r"(?m)^症例\s+\d+", r"(?m)^---+$"],
    )


def extract_bullet_value(text: str, labels: list[str]) -> str:
    label_pat = "|".join(re.escape(label) for label in labels)
    m = re.search(
        rf"(?:^|\n)\s*[・·]\s*(?:{label_pat})(?:\s*\([^)]*\))?\s*[:：]\s*(.+?)(?=\n\s*[・·]\s*[^→\n]+[:：]|\n\s*→|\Z)",
        text,
        flags=re.DOTALL,
    )
    return m.group(1).strip() if m else ""


def split_items(value: str) -> list[str]:
    value = re.sub(r"\n\s*→[\s\S]*$", "", value).strip()
    value = re.sub(r"\s+", " ", value)
    if not value:
        return []
    parts = re.split(r"[、，]| / |　/　", value)
    return [p.strip(" ・·;；") for p in parts if p.strip(" ・·;；")]


def parse_vitals(text: str) -> dict[str, Any]:
    vitals: dict[str, Any] = {}
    m = re.search(r"BP_sys\s*=\s*(\d+)", text)
    if m:
        vitals["BP_sys"] = int(m.group(1))
    m = re.search(r"BP_dia\s*=\s*(\d+)", text)
    if m:
        vitals["BP_dia"] = int(m.group(1))
    for key in ("HR", "SpO2", "RR"):
        m = re.search(rf"{key}\s*=\s*(\d+)", text)
        if m:
            vitals[key] = int(m.group(1))
    m = re.search(r"BT\s*=\s*(\d+(?:\.\d+)?)", text)
    if m:
        vitals["BT"] = float(m.group(1))
    return vitals


def parse_keyfacts(text: str) -> dict[str, Any]:
    return {
        "diagnoses": split_items(extract_bullet_value(text, ["確定診断"])),
        "diagnoses_provisional": split_items(extract_bullet_value(text, ["暫定診断", "本日初診"])),
        "medications_to_start": split_items(extract_bullet_value(text, ["開始薬"])),
        "medications_to_continue": split_items(extract_bullet_value(text, ["継続薬"])),
        "medications_to_stop": split_items(extract_bullet_value(text, ["中止薬"])),
        "medications_to_consider": split_items(extract_bullet_value(text, ["導入検討"])),
        "vitals": parse_vitals(text),
        "raw_key_facts_text": text,
    }


def build_references(text: str) -> list[dict[str, Any]]:
    text = normalize_text(text)
    sections = split_case_sections(text)
    missing = sorted(set(CASE_MAP) - set(sections))
    if missing:
        raise SystemExit(f"missing cases in document export: {', '.join(missing)}")

    rows = []
    for source_case_id, meta in CASE_MAP.items():
        section = sections[source_case_id]
        first_line = section.splitlines()[0] if section.splitlines() else source_case_id
        keyfacts_text = extract_keyfacts_text(section)
        rows.append(
            {
                "case_id": meta["case_id"],
                "source_case_id": source_case_id,
                "docs_no": meta["docs_no"],
                "title": first_line.strip(),
                "reference_soap": extract_reference_soap(section),
                "key_facts": parse_keyfacts(keyfacts_text),
            }
        )
    return sorted(rows, key=lambda r: r["case_id"])


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--doc", default=DOC_EXPORT_URL, help="Google Docs txt export URL or local txt path")
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("/data2/junkanki/naka/exports/experiment_references.jsonl"),
    )
    args = parser.parse_args()

    refs = build_references(read_doc_text(args.doc))
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        for row in refs:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
    print(args.output)
    for row in refs:
        vitals = row["key_facts"].get("vitals") or {}
        print(row["case_id"], row["source_case_id"], row["docs_no"], "SOAP", all(row["reference_soap"].get(k) for k in ("S", "O", "A", "P")), "vitals", vitals)


if __name__ == "__main__":
    main()
