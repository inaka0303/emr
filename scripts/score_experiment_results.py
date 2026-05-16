#!/usr/bin/env python3
"""Score saved human-experiment SOAP notes against the 8-case references.

This is a read-only batch runner for the EMR SQLite DB. It scores the latest
saved SOAP note for each experiment attempt and writes an XLSX workbook. The
reference JSONL should be built from the Google Doc with
`build_experiment_references.py`.
"""

from __future__ import annotations

import argparse
import importlib
import importlib.metadata as metadata
import json
import platform
import re
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

from metrics_drug import DRUG_SYNONYMS, extract_drugs  # noqa: E402
from metrics_text import compute_rouge_l, tokenize_ja  # noqa: E402
from metrics_vitals import compute_vitals_match  # noqa: E402


DEFAULT_BERTSCORE_MODEL = "cl-tohoku/bert-base-japanese-v3"
DEFAULT_BERTSCORE_NUM_LAYERS = 9

DRUG_CATEGORY_FIELDS = {
    "start": "medications_to_start",
    "continue": "medications_to_continue",
    "stop": "medications_to_stop",
    "consider": "medications_to_consider",
}

EXTRA_DRUG_SYNONYMS: dict[str, list[str]] = {
    "酸素投与": ["酸素投与", "酸素", "O2", "oxygen"],
    "生理食塩水": ["生理食塩水", "生食", "生食投与", "normal saline", "NS"],
    "モルヒネ": ["モルヒネ", "morphine"],
    "セフトリアキソン": ["セフトリアキソン", "ロセフィン", "CTRX", "ceftriaxone"],
    "ビタミンD製剤": ["ビタミンD", "ビタミン D", "活性型ビタミンD", "アルファカルシドール", "alfacalcidol"],
}

ACTION_CUES = {
    "start": [
        "開始", "投与", "内服", "静注", "経口", "舌下", "噛み砕", "ボーラス", "導入",
        "酸素", "ライン確保", "投与開始", "開始する", "準備",
    ],
    "continue": ["継続", "続行", "続け", "維持", "再開", "内服中", "服用中", "持参薬", "処方"],
    "stop": ["中止", "休薬", "禁忌", "使用しない", "避け", "保留", "使わない", "一旦中止"],
    "consider": ["検討", "考慮", "必要時", "場合", "確定時", "適応", "相談", "協議", "判断"],
}

DIAGNOSIS_SYNONYMS: dict[str, list[str]] = {
    "急性冠症候群疑い": ["急性冠症候群", "ACS", "acute coronary syndrome"],
    "前壁ST上昇型心筋梗塞疑い": ["前壁STEMI", "前壁 STEMI", "前壁心筋梗塞", "前壁 ST 上昇", "LAD", "STEMI"],
    "LAD病変疑い": ["LAD", "左前下行枝"],
    "典型的胸痛": ["典型的胸痛", "典型的", "胸痛"],
    "急性肺塞栓症疑い": ["急性肺塞栓症", "肺塞栓症", "肺血栓塞栓症", "PE", "pulmonary embolism"],
    "深部静脈血栓症疑い (右下肢)": ["深部静脈血栓症", "DVT", "右下肢", "下肢静脈血栓"],
    "若年女性の急性呼吸困難": ["若年女性", "急性呼吸困難", "呼吸困難"],
    "急性大動脈解離疑い": ["急性大動脈解離", "大動脈解離", "AD", "aortic dissection"],
    "Stanford B寄りだがStanford分類はCT待ち": ["Stanford B", "B型", "CT待ち", "Stanford分類"],
    "Stanford分類はCT待ち": ["Stanford分類", "CT待ち", "Stanford A", "Stanford B"],
    "重度高血圧": ["重度高血圧", "高血圧", "コントロール不良高血圧"],
    "重度コントロール不良高血圧": ["重度高血圧", "コントロール不良高血圧", "高血圧"],
    "慢性腎臓病 stage 3a": ["慢性腎臓病", "CKD", "stage 3a", "eGFR 50"],
    "急性呼吸不全": ["急性呼吸不全", "低酸素", "呼吸不全"],
    "急性心不全 (de novo) 疑い": ["急性心不全", "de novo", "心不全"],
    "たこつぼ症候群疑い": ["たこつぼ", "Takotsubo", "ストレス心筋症", "apical ballooning"],
    "市中肺炎疑い + 心不全合併": ["市中肺炎", "肺炎", "心不全合併"],
    "下壁ST上昇型心筋梗塞疑い": ["下壁STEMI", "下壁 STEMI", "下壁心筋梗塞", "下壁 ST 上昇", "inferior"],
    "右室梗塞疑い": ["右室梗塞", "右室", "RV infarction"],
    "高齢女性のatypical myocardial infarction": ["高齢女性", "atypical", "非典型", "心窩部", "心筋梗塞"],
    "provoked PE": ["provoked PE", "誘発性", "長距離フライト", "フライト"],
    "慢性心不全 (HFrEF) の急性増悪": ["慢性心不全", "HFrEF", "心不全急性増悪", "急性増悪"],
    "拡張型心筋症": ["拡張型心筋症", "DCM", "dilated cardiomyopathy"],
    "永続性心房細動": ["永続性心房細動", "心房細動", "AF"],
    "高血圧": ["高血圧", "HTN"],
    "Nohria-Stevenson wet-warm": ["Nohria", "wet-warm", "wet warm"],
    "Forrester II型": ["Forrester II", "Forrester 2", "II型"],
    "CS分類 CS1": ["CS1", "CS 分類", "高血圧型"],
    "塩分・水分管理乱れによる増悪": ["塩分", "水分管理", "管理が乱れ", "増悪契機"],
    "感染契機、ACS、PEのrule outが必要": ["感染", "ACS", "PE", "rule out", "除外"],
}

SCORE_SYNONYMS = {
    "Wells_PE_provisional": ["Wells", "ウェルズ"],
    "Wells_PE_interpretation": ["高確率", "中等度確率", "低確率"],
    "CS_classification_provisional": ["CS1", "CS 1", "高血圧型"],
    "NYHA": ["NYHA"],
    "Nohria_Stevenson": ["Nohria", "wet-warm", "wet warm"],
    "Forrester": ["Forrester"],
    "CS_classification": ["CS1", "CS 1", "高血圧型"],
}


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


def norm_text(value: str) -> str:
    return re.sub(r"\s+", "", (value or "").lower())


def drug_reverse_index() -> dict[str, str]:
    index: dict[str, str] = {}
    for canonical, synonyms in {**DRUG_SYNONYMS, **EXTRA_DRUG_SYNONYMS}.items():
        index[canonical.lower()] = canonical
        for synonym in synonyms:
            index[synonym.lower()] = canonical
    return index


DRUG_REVERSE_INDEX = drug_reverse_index()


def normalize_drug_items(items: list[str]) -> set[str]:
    normalized: set[str] = set()
    for item in items or []:
        item_lower = item.lower()
        for synonym, canonical in DRUG_REVERSE_INDEX.items():
            if synonym in item_lower:
                normalized.add(canonical)
    return normalized


def extract_all_drugs(text: str) -> set[str]:
    found = set(extract_drugs(text))
    text_lower = (text or "").lower()
    for synonym, canonical in DRUG_REVERSE_INDEX.items():
        if synonym in text_lower:
            found.add(canonical)
    return found


def split_contexts(text: str) -> list[str]:
    return [
        chunk.strip()
        for chunk in re.split(r"(?<=[。．.!?！？])|\n|;|；", text or "")
        if chunk.strip()
    ]


def extract_drug_action_pairs(text: str) -> set[tuple[str, str]]:
    pairs: set[tuple[str, str]] = set()
    for context in split_contexts(text):
        drugs = extract_all_drugs(context)
        if not drugs:
            continue
        categories = [
            category
            for category, cues in ACTION_CUES.items()
            if any(cue.lower() in context.lower() for cue in cues)
        ]
        for category in categories:
            for drug in drugs:
                pairs.add((category, drug))
    return pairs


def f1_from_sets(pred: set[Any], gold: set[Any]) -> dict[str, Any]:
    if not gold:
        return {
            "precision": None,
            "recall": None,
            "f1": None,
            "tp": 0,
            "fp": len(pred),
            "fn": 0,
            "predicted": sorted(pred),
            "gold": [],
            "missing": [],
            "false_positives": sorted(pred),
        }

    tp = pred & gold
    fp = pred - gold
    fn = gold - pred
    precision = len(tp) / len(pred) if pred else 0.0
    recall = len(tp) / len(gold)
    f1 = 2 * precision * recall / (precision + recall) if precision + recall else 0.0
    return {
        "precision": round(precision, 3),
        "recall": round(recall, 3),
        "f1": round(f1, 3),
        "tp": len(tp),
        "fp": len(fp),
        "fn": len(fn),
        "predicted": sorted(pred),
        "gold": sorted(gold),
        "missing": sorted(fn),
        "false_positives": sorted(fp),
    }


def compute_drug_metrics(text: str, key_facts: dict[str, Any]) -> dict[str, Any]:
    gold_by_category = {
        category: normalize_drug_items(key_facts.get(field) or [])
        for category, field in DRUG_CATEGORY_FIELDS.items()
    }
    gold_union = set().union(*gold_by_category.values()) if gold_by_category else set()
    pred_mentions = extract_all_drugs(text)
    mention = f1_from_sets(pred_mentions, gold_union)

    gold_pairs = {
        (category, drug)
        for category, drugs in gold_by_category.items()
        for drug in drugs
    }
    pred_pairs = extract_drug_action_pairs(text)
    action = f1_from_sets(pred_pairs, gold_pairs)

    by_category: dict[str, dict[str, Any]] = {}
    for category, gold in gold_by_category.items():
        pred = {drug for pred_category, drug in pred_pairs if pred_category == category}
        by_category[category] = f1_from_sets(pred, gold)

    return {
        "mention": mention,
        "action": action,
        "by_category": by_category,
        "gold_by_category": {k: sorted(v) for k, v in gold_by_category.items()},
        "predicted_pairs": sorted(f"{category}:{drug}" for category, drug in pred_pairs),
        "missing_pairs": sorted(f"{category}:{drug}" for category, drug in action.get("missing") or []),
        "false_positive_pairs": sorted(f"{category}:{drug}" for category, drug in action.get("false_positives") or []),
    }


def diagnosis_reverse_index() -> dict[str, str]:
    index: dict[str, str] = {}
    for canonical, synonyms in DIAGNOSIS_SYNONYMS.items():
        index[norm_text(canonical)] = canonical
        for synonym in synonyms:
            index[norm_text(synonym)] = canonical
    return index


DIAGNOSIS_REVERSE_INDEX = diagnosis_reverse_index()


def normalize_diagnosis_items(items: list[str]) -> set[str]:
    normalized: set[str] = set()
    for item in items or []:
        item_norm = norm_text(item)
        matched = False
        for synonym, canonical in DIAGNOSIS_REVERSE_INDEX.items():
            if synonym and synonym in item_norm:
                normalized.add(canonical)
                matched = True
        if not matched and item_norm:
            normalized.add(item.strip())
    return normalized


def extract_diagnosis_terms(text: str) -> set[str]:
    text_norm = norm_text(text)
    found: set[str] = set()
    for synonym, canonical in DIAGNOSIS_REVERSE_INDEX.items():
        if synonym and synonym in text_norm:
            found.add(canonical)
    return found


def normalize_score_value(value: Any) -> str:
    if isinstance(value, float):
        return f"{value:g}"
    return str(value)


def score_item_matched(text: str, key: str, value: Any) -> bool:
    text_norm = norm_text(text)
    value_norm = norm_text(normalize_score_value(value))
    if value_norm and value_norm in text_norm:
        if key == "Forrester" and "ii" not in value_norm and "2" not in value_norm:
            return any(token in text_norm for token in ("ii", "2型", "2"))
        return True

    for synonym in SCORE_SYNONYMS.get(key, []):
        if norm_text(synonym) in text_norm and (not value_norm or value_norm in text_norm):
            return True

    if key.startswith("Wells") and value_norm in text_norm:
        return True
    if key.startswith("CS") and ("cs1" in text_norm or "高血圧型" in text_norm):
        return normalize_score_value(value).lower() in {"cs1", "高血圧型"} or "CS1" in normalize_score_value(value)
    if key == "NYHA" and ("nyhaiv" in text_norm or "nyha4" in text_norm):
        return normalize_score_value(value).upper() in {"IV", "4"}
    return False


def compute_score_metrics(text: str, scores: dict[str, Any]) -> dict[str, Any]:
    gold = {f"{key}={normalize_score_value(value)}" for key, value in (scores or {}).items()}
    pred = {
        f"{key}={normalize_score_value(value)}"
        for key, value in (scores or {}).items()
        if score_item_matched(text, key, value)
    }
    return f1_from_sets(pred, gold)


def compute_diagnosis_metrics(text: str, key_facts: dict[str, Any]) -> dict[str, Any]:
    pred_terms = extract_diagnosis_terms(text)
    confirmed_gold = normalize_diagnosis_items(key_facts.get("diagnoses") or [])
    provisional_gold = normalize_diagnosis_items(key_facts.get("diagnoses_provisional") or [])
    score = compute_score_metrics(text, key_facts.get("scores") or {})

    confirmed = f1_from_sets(pred_terms & confirmed_gold if confirmed_gold else set(), confirmed_gold)
    provisional = f1_from_sets(pred_terms & provisional_gold if provisional_gold else set(), provisional_gold)
    union = f1_from_sets(pred_terms, confirmed_gold | provisional_gold)

    components = [confirmed.get("f1"), provisional.get("f1"), score.get("f1")]
    return {
        "confirmed": confirmed,
        "provisional": provisional,
        "score": score,
        "union": union,
        "combined_f1": average(components),
        "predicted_terms": sorted(pred_terms),
    }


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


def module_version(module_name: str, package_name: str | None = None) -> str:
    try:
        importlib.import_module(module_name)
    except Exception as exc:
        return f"missing ({type(exc).__name__}: {exc})"
    try:
        return metadata.version(package_name or module_name)
    except metadata.PackageNotFoundError:
        return "importable/version unknown"


def require_text_dependencies(with_bertscore: bool) -> None:
    required = [
        ("rouge_score", "rouge-score"),
        ("fugashi", "fugashi"),
        ("unidic_lite", "unidic-lite"),
    ]
    if with_bertscore:
        required.extend(
            [
                ("bert_score", "bert-score"),
                ("torch", "torch"),
                ("transformers", "transformers"),
            ]
        )

    missing: list[str] = []
    for module_name, package_name in required:
        try:
            importlib.import_module(module_name)
        except Exception as exc:
            missing.append(f"{package_name} ({module_name}): {exc}")

    try:
        import fugashi

        _ = fugashi.Tagger()
    except Exception as exc:
        missing.append(f"fugashi tokenizer initialization: {exc}")

    if missing:
        raise SystemExit(
            "Required text metric dependencies are missing:\n"
            + "\n".join(f"- {item}" for item in missing)
            + "\nInstall with: python3 -m pip install --user -r requirements-scoring.txt"
        )


def bertscore_device() -> str:
    try:
        import torch

        return "cuda" if torch.cuda.is_available() else "cpu"
    except Exception:
        return "cpu"


_BERTSCORER = None
_BERTSCORER_CONFIG: tuple[str, int, str] | None = None


def compute_bertscore_cached(generated: str, reference: str, model_type: str, num_layers: int) -> float | None:
    if not generated or not reference:
        return 0.0

    global _BERTSCORER, _BERTSCORER_CONFIG
    device = bertscore_device()
    config = (model_type, num_layers, device)
    if _BERTSCORER is None or _BERTSCORER_CONFIG != config:
        from bert_score import BERTScorer

        _BERTSCORER = BERTScorer(
            model_type=model_type,
            num_layers=num_layers,
            device=device,
            batch_size=1,
            rescale_with_baseline=False,
        )
        _BERTSCORER_CONFIG = config

    _, _, f1 = _BERTSCORER.score([generated], [reference])
    return round(f1.item(), 4)


def compute_text_metrics_safe(
    generated: str,
    reference: str,
    with_bertscore: bool,
    bertscore_model: str,
    bertscore_num_layers: int,
    strict_text_deps: bool,
) -> dict[str, Any]:
    out: dict[str, Any]
    try:
        out = {"rouge_l": compute_rouge_l(generated, reference)}
    except ModuleNotFoundError as exc:
        if exc.name != "rouge_score":
            raise
        out = {
            "rouge_l": rouge_l_fallback(generated, reference),
            "text_metric_note": "rouge_score package unavailable; used local ROUGE-L fallback",
        }

    if with_bertscore:
        try:
            bs = compute_bertscore_cached(generated, reference, bertscore_model, bertscore_num_layers)
            if bs is not None:
                out["bertscore_f1"] = bs
        except Exception as exc:
            if strict_text_deps:
                raise
            out["bertscore_note"] = f"BERTScore failed: {type(exc).__name__}: {exc}"
    return out


def score_attempt(
    row: dict[str, Any],
    ref: dict[str, Any] | None,
    with_bertscore: bool,
    bertscore_model: str,
    bertscore_num_layers: int,
    strict_text_deps: bool,
) -> dict[str, Any]:
    out: dict[str, Any] = {
        "attempt_id": row.get("attempt_id"),
        "subject_id": row.get("subject_id"),
        "sequence_order": row.get("sequence_order"),
        "case_id": row.get("case_id"),
        "source_case_id": row.get("source_case_id"),
        "docs_no": row.get("docs_no"),
        "reference_version": ref.get("reference_version") if ref else None,
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

    text_metrics = compute_text_metrics_safe(
        gen_full,
        ref_full,
        with_bertscore=with_bertscore,
        bertscore_model=bertscore_model,
        bertscore_num_layers=bertscore_num_layers,
        strict_text_deps=strict_text_deps,
    )
    drug_metrics = compute_drug_metrics(gen_full, key_facts)
    drug = drug_metrics["action"]
    drug_mention = drug_metrics["mention"]
    diagnosis_metrics = compute_diagnosis_metrics(gen_full, key_facts)
    diagnosis = diagnosis_metrics["union"]
    vitals = compute_vitals_match(gen_full, key_facts.get("vitals") or {}, key_facts.get("labs") or {})

    components = [
        text_metrics.get("rouge_l"),
        text_metrics.get("bertscore_f1"),
        drug.get("f1"),
        diagnosis_metrics.get("combined_f1"),
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
            "drug_mention_f1": drug_mention.get("f1"),
            "drug_mention_precision": drug_mention.get("precision"),
            "drug_mention_recall": drug_mention.get("recall"),
            "drug_start_f1": drug_metrics["by_category"]["start"].get("f1"),
            "drug_continue_f1": drug_metrics["by_category"]["continue"].get("f1"),
            "drug_stop_f1": drug_metrics["by_category"]["stop"].get("f1"),
            "drug_consider_f1": drug_metrics["by_category"]["consider"].get("f1"),
            "diagnosis_f1": diagnosis_metrics.get("combined_f1"),
            "diagnosis_union_f1": diagnosis.get("f1"),
            "diagnosis_confirmed_f1": diagnosis_metrics["confirmed"].get("f1"),
            "diagnosis_provisional_f1": diagnosis_metrics["provisional"].get("f1"),
            "diagnosis_score_f1": diagnosis_metrics["score"].get("f1"),
            "diagnosis_precision": diagnosis.get("precision"),
            "diagnosis_recall": diagnosis.get("recall"),
            "vitals_match_rate": vitals.get("match_rate"),
            "vitals_matched": vitals.get("matched"),
            "vitals_expected": vitals.get("expected"),
            "missing_drugs": ", ".join(drug_mention.get("missing") or []),
            "false_positive_drugs": ", ".join(drug_mention.get("false_positives") or []),
            "missing_drug_pairs": ", ".join(drug_metrics.get("missing_pairs") or []),
            "false_positive_drug_pairs": ", ".join(drug_metrics.get("false_positive_pairs") or []),
            "predicted_drug_pairs": ", ".join(drug_metrics.get("predicted_pairs") or []),
            "gold_drugs_by_category_json": compact_json(drug_metrics.get("gold_by_category") or {}),
            "missing_diagnoses": ", ".join(diagnosis.get("missing") or []),
            "missing_confirmed_diagnoses": ", ".join(diagnosis_metrics["confirmed"].get("missing") or []),
            "missing_provisional_diagnoses": ", ".join(diagnosis_metrics["provisional"].get("missing") or []),
            "missing_score_items": ", ".join(diagnosis_metrics["score"].get("missing") or []),
            "predicted_drugs": ", ".join(drug_mention.get("predicted") or []),
            "predicted_diagnoses": ", ".join(diagnosis_metrics.get("predicted_terms") or []),
            "vitals_details_json": compact_json(vitals.get("details") or {}),
            "text_metric_note": text_metrics.get("text_metric_note"),
            "bertscore_note": text_metrics.get("bertscore_note"),
            "reference_key_facts_json": compact_json(key_facts),
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
                "duration_sec": average([m.get("duration_sec") for m in scored]),
                "composite_score": average([m.get("composite_score") for m in scored]),
                "rouge_l": average([m.get("rouge_l") for m in scored]),
                "bertscore_f1": average([m.get("bertscore_f1") for m in scored]),
                "drug_f1": average([m.get("drug_f1") for m in scored]),
                "drug_mention_f1": average([m.get("drug_mention_f1") for m in scored]),
                "drug_start_f1": average([m.get("drug_start_f1") for m in scored]),
                "drug_continue_f1": average([m.get("drug_continue_f1") for m in scored]),
                "drug_stop_f1": average([m.get("drug_stop_f1") for m in scored]),
                "drug_consider_f1": average([m.get("drug_consider_f1") for m in scored]),
                "diagnosis_f1": average([m.get("diagnosis_f1") for m in scored]),
                "diagnosis_union_f1": average([m.get("diagnosis_union_f1") for m in scored]),
                "diagnosis_confirmed_f1": average([m.get("diagnosis_confirmed_f1") for m in scored]),
                "diagnosis_provisional_f1": average([m.get("diagnosis_provisional_f1") for m in scored]),
                "diagnosis_score_f1": average([m.get("diagnosis_score_f1") for m in scored]),
                "vitals_match_rate": average([m.get("vitals_match_rate") for m in scored]),
            }
        )
    return summary


def summarize_by_keys(rows: list[dict[str, Any]], group_keys: list[str]) -> list[dict[str, Any]]:
    groups: dict[tuple[str, ...], list[dict[str, Any]]] = {}
    for row in rows:
        key = tuple(str(row.get(group_key) or "") for group_key in group_keys)
        groups.setdefault(key, []).append(row)

    summary = []
    for key, members in sorted(groups.items()):
        scored = [m for m in members if m.get("score_status") == "scored"]
        out = {group_key: key[idx] for idx, group_key in enumerate(group_keys)}
        out.update(
            {
                "attempts": len(members),
                "scored": len(scored),
                "composite_score": average([m.get("composite_score") for m in scored]),
                "rouge_l": average([m.get("rouge_l") for m in scored]),
                "bertscore_f1": average([m.get("bertscore_f1") for m in scored]),
                "drug_f1": average([m.get("drug_f1") for m in scored]),
                "drug_mention_f1": average([m.get("drug_mention_f1") for m in scored]),
                "diagnosis_f1": average([m.get("diagnosis_f1") for m in scored]),
                "diagnosis_score_f1": average([m.get("diagnosis_score_f1") for m in scored]),
                "vitals_match_rate": average([m.get("vitals_match_rate") for m in scored]),
                "duration_sec": average([m.get("duration_sec") for m in scored]),
            }
        )
        summary.append(out)
    return summary


def tokenizer_status() -> str:
    try:
        import fugashi

        tagger = fugashi.Tagger()
        sample = "急性心筋梗塞を疑う"
        tokens = " / ".join(w.surface for w in tagger(sample))
        return f"fugashi+unidic-lite ({tokens})"
    except Exception as exc:
        return f"char fallback ({type(exc).__name__}: {exc})"


def collect_scoring_environment(args: argparse.Namespace, scores: list[dict[str, Any]]) -> list[dict[str, Any]]:
    env = {
        "recorded_at": datetime.now().isoformat(timespec="seconds"),
        "python": sys.version.replace("\n", " "),
        "python_executable": sys.executable,
        "platform": platform.platform(),
        "db_path": str(args.db),
        "reference_path": str(args.references),
        "with_bertscore": str(bool(args.with_bertscore)),
        "strict_text_deps": str(bool(args.strict_text_deps)),
        "scored_rows": str(sum(1 for row in scores if row.get("score_status") == "scored")),
        "total_attempt_rows": str(len(scores)),
        "rouge_implementation": "rouge-score rougeL on fugashi-tokenized Japanese text",
        "tokenizer": tokenizer_status(),
        "bertscore_model": args.bertscore_model if args.with_bertscore else "",
        "bertscore_num_layers": str(args.bertscore_num_layers) if args.with_bertscore else "",
        "bertscore_device": bertscore_device() if args.with_bertscore else "",
        "rouge-score": module_version("rouge_score", "rouge-score"),
        "fugashi": module_version("fugashi", "fugashi"),
        "unidic-lite": module_version("unidic_lite", "unidic-lite"),
        "bert-score": module_version("bert_score", "bert-score"),
        "torch": module_version("torch", "torch"),
        "transformers": module_version("transformers", "transformers"),
    }
    return [{"key": key, "value": value} for key, value in env.items()]


def write_environment_json(path: Path, environment_rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps({row["key"]: row["value"] for row in environment_rows}, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


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
    parser.add_argument(
        "--strict-text-deps",
        action="store_true",
        help="Fail if rouge_score/fugashi/unidic-lite or requested BERTScore dependencies are unavailable",
    )
    parser.add_argument("--bertscore-model", default=DEFAULT_BERTSCORE_MODEL, help="Hugging Face model for BERTScore")
    parser.add_argument("--bertscore-num-layers", type=int, default=DEFAULT_BERTSCORE_NUM_LAYERS)
    parser.add_argument("--environment-json", type=Path, help="Optional JSON copy of scoring environment")
    args = parser.parse_args()

    if not args.references.exists():
        raise SystemExit(f"reference JSONL not found: {args.references}")
    if args.strict_text_deps:
        require_text_dependencies(args.with_bertscore)

    refs = load_references(args.references)
    with export_results.connect_readonly(args.db) as conn:
        attempts = export_results.latest_results(conn)

    scores = [
        score_attempt(
            row,
            refs.get(row.get("case_id")),
            args.with_bertscore,
            args.bertscore_model,
            args.bertscore_num_layers,
            args.strict_text_deps,
        )
        for row in attempts
    ]
    if args.with_bertscore and args.strict_text_deps:
        missing_bertscore = [
            str(row.get("attempt_id"))
            for row in scores
            if row.get("score_status") == "scored" and row.get("bertscore_f1") is None
        ]
        if missing_bertscore:
            raise SystemExit(f"BERTScore was requested but missing for attempts: {', '.join(missing_bertscore)}")

    environment_rows = collect_scoring_environment(args, scores)
    sheets = {
        "scores": scores,
        "summary_by_intervention": summarize(scores, "intervention"),
        "summary_by_case": summarize(scores, "case_id"),
        "summary_by_subject": summarize(scores, "subject_id"),
        "summary_subject_x_intervention": summarize_by_keys(scores, ["subject_id", "intervention"]),
        "scoring_environment": environment_rows,
    }
    export_results.write_xlsx(args.output, sheets)
    print(args.output)

    if args.json_output:
        args.json_output.parent.mkdir(parents=True, exist_ok=True)
        args.json_output.write_text(
            json.dumps({"scores": scores, "scoring_environment": environment_rows}, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        print(args.json_output)

    if args.environment_json:
        write_environment_json(args.environment_json, environment_rows)
        print(args.environment_json)

    scored = sum(1 for row in scores if row.get("score_status") == "scored")
    print(f"scored {scored}/{len(scores)} saved SOAP notes")


if __name__ == "__main__":
    main()
