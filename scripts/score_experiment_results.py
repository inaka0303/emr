#!/usr/bin/env python3
"""Score saved human-experiment SOAP notes against the 8-case references.

This is a read-only batch runner for the EMR SQLite DB. It scores the latest
saved SOAP note for each experiment attempt and writes an XLSX workbook. The
reference JSONL should be built from the Google Doc with
`build_experiment_references.py`.
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

import export_experiment_results as export_results


SCRIPT_DIR = Path(__file__).resolve().parent
NAKA_ROOT = SCRIPT_DIR.parents[1]
EVAL_DIR = NAKA_ROOT / "data" / "aci_jp_cardio" / "admission" / "eval_runner"
DEFAULT_REFERENCES = SCRIPT_DIR.parent / "references" / "experiment_references.jsonl"

sys.path.insert(0, str(EVAL_DIR))

from metrics_diagnosis import compute_diagnosis_f1, normalize_keyfacts_diagnoses  # noqa: E402
from metrics_drug import compute_drug_f1, normalize_keyfacts_drugs  # noqa: E402
from metrics_text import compute_text_metrics, tokenize_ja  # noqa: E402
from metrics_vitals import compute_vitals_match  # noqa: E402


def load_references(path: Path) -> dict[str, dict[str, Any]]:
    refs: dict[str, dict[str, Any]] = {}
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            row = json.loads(line)
            refs[row["case_id"]] = row
    return refs


def reference_full_text(ref: dict[str, Any]) -> str:
    soap = ref.get("reference_soap") or {}
    return "\n\n".join(
        part
        for part in [
            f"S:\n{soap.get('S') or ''}".strip(),
            f"O:\n{soap.get('O') or ''}".strip(),
            f"A:\n{soap.get('A') or ''}".strip(),
            f"P:\n{soap.get('P') or ''}".strip(),
        ]
        if part
    )


def average(values: list[float | int | None]) -> float | None:
    numeric = [float(v) for v in values if isinstance(v, (int, float))]
    return round(sum(numeric) / len(numeric), 4) if numeric else None


def compact_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def rouge_l_fallback(generated: str, reference: str) -> float:
    g_tokens = tokenize_ja(generated).split()
    r_tokens = tokenize_ja(reference).split()
    if not g_tokens or not r_tokens:
        return 0.0

    prev = [0] * (len(r_tokens) + 1)
    for g in g_tokens:
        cur = [0]
        for j, r in enumerate(r_tokens, start=1):
            if g == r:
                cur.append(prev[j - 1] + 1)
            else:
                cur.append(max(prev[j], cur[-1]))
        prev = cur

    lcs = prev[-1]
    precision = lcs / len(g_tokens)
    recall = lcs / len(r_tokens)
    if precision + recall == 0:
        return 0.0
    return round(2 * precision * recall / (precision + recall), 4)


def compute_text_metrics_safe(generated: str, reference: str, with_bertscore: bool) -> dict[str, Any]:
    try:
        return compute_text_metrics(generated, reference, skip_bertscore=not with_bertscore)
    except ModuleNotFoundError as exc:
        if exc.name != "rouge_score":
            raise
        return {
            "rouge_l": rouge_l_fallback(generated, reference),
            "text_metric_note": "rouge_score package unavailable; used local ROUGE-L fallback",
        }


def score_attempt(row: dict[str, Any], ref: dict[str, Any] | None, with_bertscore: bool) -> dict[str, Any]:
    out: dict[str, Any] = {
        "attempt_id": row.get("attempt_id"),
        "subject_id": row.get("subject_id"),
        "sequence_order": row.get("sequence_order"),
        "case_id": row.get("case_id"),
        "source_case_id": row.get("source_case_id"),
        "docs_no": row.get("docs_no"),
        "intervention": row.get("intervention"),
        "status": row.get("status"),
        "duration_sec": row.get("duration_sec"),
        "ai_accept_count": row.get("ai_accept_count"),
        "ai_edit_count": row.get("ai_edit_count"),
        "ai_reject_count": row.get("ai_reject_count"),
        "soap_note_id": row.get("soap_note_id"),
        "soap_created_at": row.get("soap_created_at"),
        "soap_updated_at": row.get("soap_updated_at"),
        "has_saved_soap": int(row.get("has_saved_soap") or 0),
    }

    if not row.get("has_saved_soap"):
        out["score_status"] = "no_saved_soap"
        return out
    if ref is None:
        out["score_status"] = "missing_reference"
        return out

    gen_full = row.get("soap_full_text") or ""
    ref_full = reference_full_text(ref)
    key_facts = ref.get("key_facts") or {}

    text_metrics = compute_text_metrics_safe(gen_full, ref_full, with_bertscore=with_bertscore)
    gold_drugs = normalize_keyfacts_drugs(
        (key_facts.get("medications_to_start") or [])
        + (key_facts.get("medications_to_continue") or [])
        + (key_facts.get("medications_to_stop") or [])
        + (key_facts.get("medications_to_consider") or [])
    )
    gold_diagnoses = normalize_keyfacts_diagnoses(
        (key_facts.get("diagnoses") or []) + (key_facts.get("diagnoses_provisional") or [])
    )
    drug = compute_drug_f1(gen_full, gold_drugs)
    diagnosis = compute_diagnosis_f1(gen_full, gold_diagnoses)
    vitals = compute_vitals_match(gen_full, key_facts.get("vitals") or {}, key_facts.get("labs") or {})

    components = [
        text_metrics.get("rouge_l"),
        text_metrics.get("bertscore_f1"),
        drug.get("f1"),
        diagnosis.get("f1"),
        vitals.get("match_rate"),
    ]
    if not with_bertscore:
        components = [v for i, v in enumerate(components) if i != 1]

    out.update(
        {
            "score_status": "scored",
            "composite_score": average(components),
            "rouge_l": text_metrics.get("rouge_l"),
            "bertscore_f1": text_metrics.get("bertscore_f1"),
            "drug_f1": drug.get("f1"),
            "drug_precision": drug.get("precision"),
            "drug_recall": drug.get("recall"),
            "diagnosis_f1": diagnosis.get("f1"),
            "diagnosis_precision": diagnosis.get("precision"),
            "diagnosis_recall": diagnosis.get("recall"),
            "vitals_match_rate": vitals.get("match_rate"),
            "vitals_matched": vitals.get("matched"),
            "vitals_expected": vitals.get("expected"),
            "missing_drugs": ", ".join(drug.get("missing") or []),
            "false_positive_drugs": ", ".join(drug.get("false_positives") or []),
            "missing_diagnoses": ", ".join(diagnosis.get("missing") or []),
            "predicted_drugs": ", ".join(drug.get("predicted") or []),
            "predicted_diagnoses": ", ".join(diagnosis.get("predicted") or []),
            "vitals_details_json": compact_json(vitals.get("details") or {}),
            "text_metric_note": text_metrics.get("text_metric_note"),
            "generated_soap": gen_full,
            "reference_soap": ref_full,
        }
    )
    return out


def summarize(rows: list[dict[str, Any]], group_key: str) -> list[dict[str, Any]]:
    groups: dict[str, list[dict[str, Any]]] = {}
    for row in rows:
        groups.setdefault(str(row.get(group_key) or ""), []).append(row)
    summary = []
    for key, members in sorted(groups.items()):
        scored = [m for m in members if m.get("score_status") == "scored"]
        summary.append(
            {
                group_key: key,
                "attempts": len(members),
                "scored": len(scored),
                "composite_score": average([m.get("composite_score") for m in scored]),
                "rouge_l": average([m.get("rouge_l") for m in scored]),
                "drug_f1": average([m.get("drug_f1") for m in scored]),
                "diagnosis_f1": average([m.get("diagnosis_f1") for m in scored]),
                "vitals_match_rate": average([m.get("vitals_match_rate") for m in scored]),
            }
        )
    return summary


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", type=Path, default=export_results.pick_default_db(), help="SQLite DB path")
    parser.add_argument("--references", type=Path, default=DEFAULT_REFERENCES, help="Reference JSONL path")
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("/data2/junkanki/naka/exports")
        / f"experiment_scores_{datetime.now().strftime('%Y%m%d_%H%M%S')}.xlsx",
        help="Output .xlsx path",
    )
    parser.add_argument("--json-output", type=Path, help="Optional JSON copy")
    parser.add_argument("--with-bertscore", action="store_true", help="Also compute BERTScore F1; slower and model-dependent")
    args = parser.parse_args()

    if not args.references.exists():
        raise SystemExit(f"reference JSONL not found: {args.references}")

    refs = load_references(args.references)
    with export_results.connect_readonly(args.db) as conn:
        attempts = export_results.latest_results(conn)

    scores = [score_attempt(row, refs.get(row.get("case_id")), args.with_bertscore) for row in attempts]
    sheets = {
        "scores": scores,
        "summary_by_intervention": summarize(scores, "intervention"),
        "summary_by_case": summarize(scores, "case_id"),
    }
    export_results.write_xlsx(args.output, sheets)
    print(args.output)

    if args.json_output:
        args.json_output.parent.mkdir(parents=True, exist_ok=True)
        args.json_output.write_text(json.dumps({"scores": scores}, ensure_ascii=False, indent=2), encoding="utf-8")
        print(args.json_output)

    scored = sum(1 for row in scores if row.get("score_status") == "scored")
    print(f"scored {scored}/{len(scores)} saved SOAP notes")


if __name__ == "__main__":
    main()
