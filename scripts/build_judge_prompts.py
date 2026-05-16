#!/usr/bin/env python3
"""Build manual LLM-judge prompts for the human SOAP experiment.

This intentionally does not call any model API. It writes one JSONL row per
saved attempt so the rows can be reviewed in ChatGPT/GPT-5.5/Codex and the
returned JSON can later be merged by `merge_judge_results.py`.
"""

from __future__ import annotations

import argparse
import json
from datetime import datetime
from pathlib import Path
from typing import Any

import export_experiment_results as export_results
import score_experiment_results as scoring


DEFAULT_OUTPUT = Path("/home/junkanki/naka/emr/exports/judge_prompts_current.jsonl")
RUBRIC_VERSION = "gpt_judge_5axis_v1_2026-05-16"


def compact_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def pretty_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, indent=2)


def expected_schema(attempt_id: str) -> dict[str, Any]:
    return {
        "attempt_id": attempt_id,
        "medical": 1,
        "completeness": 1,
        "naturalness": 1,
        "hallucination_absence": 1,
        "format": 1,
        "major_omissions": ["短い日本語で列挙"],
        "unsafe_or_hallucinated_items": ["短い日本語で列挙"],
        "comment": "80字以内の日本語コメント",
    }


def build_prompt(row: dict[str, Any], ref: dict[str, Any]) -> str:
    attempt_id = str(row.get("attempt_id") or "")
    reference_soap = scoring.reference_full_text(ref)
    generated_soap = row.get("soap_full_text") or ""
    key_facts = ref.get("key_facts") or {}

    return f"""あなたは循環器診療と医学教育に詳しい日本語カルテ評価者です。
以下の「被験者SOAP」を「正解SOAP」と「採点用key_facts」に照らして、5軸を1-5点の整数で評価してください。

重要:
- JSONだけを返してください。Markdown、説明文、コードフェンスは禁止です。
- attempt_idは必ず "{attempt_id}" のまま返してください。
- 同義表現は許容します。正解SOAPと完全一致していなくても、医学的に同等なら減点しすぎないでください。
- ただし、危険な見落とし、禁忌薬、根拠のない検査所見・診断・数値の捏造は厳しく減点してください。
- この症例は問診直後/初診外来の記録です。まだ実施されていない検査結果を断定していればハルシネーションとして扱ってください。

5軸:
1. medical: 医学的妥当性。鑑別、緊急度、治療/搬送/検査方針が臨床的に妥当か。
2. completeness: 情報網羅性。主訴、重要既往、薬剤、バイタル、key_factsの診断・薬剤・処置を拾えているか。
3. naturalness: 日本語カルテとして自然で、医療者が読みやすいか。
4. hallucination_absence: ハルシネーションの少なさ。根拠のない所見、検査値、処方、断定診断がないか。
5. format: SOAP形式とセクション分けが守られているか。

点数の目安:
- 5: 臨床的に十分で、軽微な言い換えや省略のみ。
- 4: 実用上おおむね良いが、重要でない不足がある。
- 3: 部分的に有用だが、重要事項の漏れや曖昧さがある。
- 2: 主要診断/初期対応/安全性に大きな不足がある。
- 1: 危険または評価不能。

返答JSONスキーマ:
{pretty_json(expected_schema(attempt_id))}

メタ情報:
- attempt_id: {attempt_id}
- subject_id: {row.get("subject_id") or ""}
- case_id: {row.get("case_id") or ""}
- source_case_id: {row.get("source_case_id") or ref.get("source_case_id") or ""}
- intervention: {row.get("intervention") or ""}
- reference_version: {ref.get("reference_version") or ""}
- rubric_version: {RUBRIC_VERSION}

採点用key_facts:
{pretty_json(key_facts)}

正解SOAP:
{reference_soap}

被験者SOAP:
{generated_soap}
"""


def build_rows(db_path: Path, references_path: Path) -> list[dict[str, Any]]:
    refs = scoring.load_references(references_path)
    with export_results.connect_readonly(db_path) as conn:
        attempts = export_results.latest_results(conn)

    rows: list[dict[str, Any]] = []
    for row in attempts:
        if not row.get("has_saved_soap"):
            continue
        case_id = row.get("case_id")
        ref = refs.get(case_id)
        if not ref:
            continue

        rows.append(
            {
                "attempt_id": row.get("attempt_id"),
                "subject_id": row.get("subject_id"),
                "sequence_order": row.get("sequence_order"),
                "case_id": case_id,
                "source_case_id": row.get("source_case_id") or ref.get("source_case_id"),
                "intervention": row.get("intervention"),
                "reference_version": ref.get("reference_version"),
                "rubric_version": RUBRIC_VERSION,
                "generated_soap": row.get("soap_full_text") or "",
                "reference_soap": scoring.reference_full_text(ref),
                "reference_key_facts_json": compact_json(ref.get("key_facts") or {}),
                "prompt": build_prompt(row, ref),
            }
        )
    return rows


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", type=Path, default=export_results.pick_default_db(), help="SQLite DB path")
    parser.add_argument("--references", type=Path, default=scoring.DEFAULT_REFERENCES, help="Reference JSONL path")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT, help="Output JSONL path")
    args = parser.parse_args()

    if not args.references.exists():
        raise SystemExit(f"reference JSONL not found: {args.references}")

    rows = build_rows(args.db, args.references)
    write_jsonl(args.output, rows)
    print(args.output)
    print(f"built {len(rows)} judge prompts at {datetime.now().isoformat(timespec='seconds')}")


if __name__ == "__main__":
    main()
