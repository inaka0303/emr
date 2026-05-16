#!/usr/bin/env python3
"""Apply hand-curated key_facts to the 8-case experiment reference JSONL.

The reference SOAP text still comes from the reviewed Google Doc export, but
the scoring key_facts are intentionally hand-maintained here. The Google Doc
format is convenient for clinical review, not for exact metric inputs.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_REFERENCE_PATH = SCRIPT_DIR.parent / "references" / "experiment_references.jsonl"

REQUIRED_VITAL_KEYS = ("BP_sys", "BP_dia", "HR", "SpO2", "RR", "BT")


def kf(
    *,
    diagnoses: list[str] | None = None,
    diagnoses_provisional: list[str] | None = None,
    scores: dict[str, Any] | None = None,
    medications_to_start: list[str] | None = None,
    medications_to_continue: list[str] | None = None,
    medications_to_stop: list[str] | None = None,
    medications_to_consider: list[str] | None = None,
    vitals: dict[str, int | float],
    procedures: list[str] | None = None,
    notes: list[str] | None = None,
) -> dict[str, Any]:
    return {
        "diagnoses": diagnoses or [],
        "diagnoses_provisional": diagnoses_provisional or [],
        "scores": scores or {},
        "medications_to_start": medications_to_start or [],
        "medications_to_continue": medications_to_continue or [],
        "medications_to_stop": medications_to_stop or [],
        "medications_to_consider": medications_to_consider or [],
        "vitals": vitals,
        "labs": {},
        "procedures": procedures or [],
        "curation_notes": notes or [],
    }


MANUAL_KEY_FACTS: dict[str, dict[str, Any]] = {
    "C1": kf(
        diagnoses_provisional=[
            "急性冠症候群疑い",
            "前壁ST上昇型心筋梗塞疑い",
            "LAD病変疑い",
            "典型的胸痛",
        ],
        medications_to_start=[
            "アスピリン 200mg 噛み砕き内服",
            "クロピドグレル 300mg (PCI直前、搬送先と協議)",
            "ニトログリセリン舌下 0.3mg (BP維持下、右室梗塞示唆なしの場合)",
            "酸素投与 (SpO2 <94%時)",
            "フェンタニルまたはモルヒネ静注",
            "生理食塩水 (静脈ライン確保)",
        ],
        medications_to_continue=[
            "テルミサルタン 40mg/日 (血圧・腎機能・Kをみて継続/再開判断)",
        ],
        medications_to_stop=[
            "アムロジピン 5mg/日 (AMI急性期は一旦中止し、血圧・心機能をみて再開判断)",
        ],
        vitals={"BP_sys": 145, "BP_dia": 92, "HR": 92, "SpO2": 96, "RR": 20, "BT": 36.5},
        procedures=[
            "12誘導ECGを即時記録",
            "両上肢血圧差を確認",
            "高感度トロポニンI、CK-MB、BNP、D-dimer、電解質、腎機能、肝機能、血糖、CBC、凝固系を採血",
            "静脈ライン2本確保、心電図モニター、AED準備",
            "ST上昇があれば採血を待たずPCI可能施設へ救急搬送",
            "胸部X線で肺うっ血、気胸、大動脈陰影を評価",
        ],
        notes=[
            "Specialist review 2026-05-14: チカグレロルではなくクロピドグレルを採用。",
            "Specialist review 2026-05-14: アムロジピンはAMI急性期に一旦中止し経過判断。",
        ],
    ),
    "C2": kf(
        diagnoses_provisional=[
            "急性肺塞栓症疑い",
            "深部静脈血栓症疑い (右下肢)",
            "若年女性の急性呼吸困難",
        ],
        scores={"Wells_PE_provisional": 4.5, "Wells_PE_interpretation": "中等度確率"},
        medications_to_consider=[
            "未分画ヘパリン静注 (PE確定時)",
            "アピキサバン 10mg 1日2回 7日間、その後5mg 1日2回 (low risk PE確定時)",
            "アルテプラーゼ (massive PEで血行動態不安定時)",
            "酸素投与 (SpO2 >=94%維持目的、必要時)",
        ],
        vitals={"BP_sys": 118, "BP_dia": 76, "HR": 102, "SpO2": 94, "RR": 24, "BT": 36.6},
        procedures=[
            "12誘導ECGで右心負荷所見とACSを確認",
            "D-dimer、トロポニンI、BNP、血液ガス、電解質、腎機能を採血",
            "胸部X線で自然気胸などを除外",
            "WellsスコアとD-dimerでPE事前確率を評価",
            "中等度以上かつD-dimer陽性なら造影CTPA",
            "両側下肢静脈エコーでDVT検索",
        ],
    ),
    "C3": kf(
        diagnoses_provisional=[
            "急性大動脈解離疑い",
            "Stanford B寄りだがStanford分類はCT待ち",
            "重度高血圧",
            "慢性腎臓病 stage 3a",
        ],
        medications_to_start=[
            "ニカルジピン静注",
            "ランジオロール静注",
            "フェンタニルまたはモルヒネ静注",
            "生理食塩水 (ライン確保、造影CT前負荷)",
        ],
        medications_to_continue=[
            "テルミサルタン 40mg/日",
            "アムロジピン 5mg/日",
            "アトルバスタチン 10mg/日",
        ],
        vitals={"BP_sys": 168, "BP_dia": 95, "HR": 78, "SpO2": 97, "RR": 18, "BT": 36.4},
        procedures=[
            "緊急造影CT (大動脈全長、心臓、腹部)",
            "12誘導ECGでType A解離による冠動脈巻き込みを評価",
            "両下肢動脈触知、聴診、腹部診察、神経学的所見を評価",
            "降圧目標SBP 100-120 mmHg、HR 60-80/minでimpulse control",
            "心臓血管外科へ緊急コンサルト",
            "Stanford Aまたはcomplicated Bなら高度医療機関へ緊急搬送",
        ],
    ),
    "C4": kf(
        diagnoses_provisional=[
            "急性呼吸不全",
            "急性心不全 (de novo) 疑い",
            "たこつぼ症候群疑い",
            "市中肺炎疑い + 心不全合併",
        ],
        scores={"CS_classification_provisional": "CS1"},
        medications_to_start=[
            "酸素投与 (経鼻3-5L、改善不十分ならNPPV検討)",
            "フロセミド静注 20mg (肺うっ血像確認後)",
            "ニトログリセリン静注 (BP維持下で肺うっ血対策に考慮)",
            "経験的抗菌薬 CTRX (細菌性肺炎所見時)",
        ],
        medications_to_continue=[
            "ビタミンD製剤 (整形外科処方、医師管理下で継続)",
        ],
        medications_to_stop=[
            "市販総合感冒薬",
        ],
        vitals={"BP_sys": 142, "BP_dia": 82, "HR": 108, "SpO2": 89, "RR": 28, "BT": 37.4},
        procedures=[
            "酸素投与、NPPV準備",
            "12誘導ECG",
            "トロポニンI、BNP/NT-proBNP、炎症反応、D-dimer、血液ガスを採血",
            "胸部X線",
            "経胸壁心エコーでapical ballooningとLVEFを評価",
            "CCU/HCU入院判定",
        ],
    ),
    "C5": kf(
        diagnoses_provisional=[
            "急性冠症候群疑い",
            "下壁ST上昇型心筋梗塞疑い",
            "右室梗塞疑い",
            "高齢女性のatypical myocardial infarction",
        ],
        medications_to_start=[
            "アスピリン 200mg 噛み砕き内服",
            "酸素投与 (SpO2 <94%時)",
            "生理食塩水 (右室梗塞合併時のvolume loading)",
        ],
        medications_to_consider=[
            "アムロジピン 5mg/日 (血圧経過次第で一時休薬検討)",
            "アレンドロン酸 (急性期は休薬)",
            "アルファカルシドール 0.5μg/日",
        ],
        medications_to_stop=[
            "ニトログリセリン舌下 (右室梗塞合併時禁忌)",
        ],
        vitals={"BP_sys": 102, "BP_dia": 68, "HR": 56, "SpO2": 95, "RR": 18, "BT": 36.2},
        procedures=[
            "12誘導ECG、必要時右側胸部誘導V3R/V4R",
            "ST変化があれば採血を待たずPCI可能施設へ救急搬送",
            "高感度トロポニンI、BNP、D-dimer、電解質、腎機能、肝機能、血糖、膵酵素、CBC、凝固系を採血",
            "静脈ライン確保、低血圧時のvolume loading準備",
            "胸部X線",
            "家族連絡と緊急PCI説明準備",
        ],
    ),
    "C6": kf(
        diagnoses_provisional=[
            "急性肺塞栓症疑い",
            "provoked PE",
            "深部静脈血栓症疑い (右下肢)",
        ],
        scores={"Wells_PE_provisional": 7.5, "Wells_PE_interpretation": "高確率"},
        medications_to_start=[
            "未分画ヘパリン静注",
            "アピキサバン 10mg 1日2回 7日間、その後5mg 1日2回",
            "酸素投与 (SpO2 >=94%維持目的)",
            "アルテプラーゼ tPA (massive PEで血行動態不安定時)",
            "生理食塩水 (ライン確保)",
        ],
        medications_to_continue=[
            "アトルバスタチン 10mg/日",
        ],
        vitals={"BP_sys": 128, "BP_dia": 82, "HR": 110, "SpO2": 92, "RR": 24, "BT": 36.7},
        procedures=[
            "Wells高確率のためD-dimer結果を待たずCTPAへ進める",
            "12誘導ECG",
            "D-dimer、トロポニンI、BNP、血液ガス、腎機能、凝固系を採血",
            "胸部X線",
            "両側下肢静脈エコー",
            "CTPA確定後にsPESI、RV/LV比、トロポニンで重症度評価",
        ],
    ),
    "C7": kf(
        diagnoses_provisional=[
            "急性大動脈解離疑い",
            "Stanford分類はCT待ち",
            "重度コントロール不良高血圧",
        ],
        medications_to_start=[
            "ニカルジピン静注",
            "ランジオロール静注",
            "フェンタニルまたはモルヒネ静注",
        ],
        vitals={"BP_sys": 168, "BP_dia": 98, "HR": 102, "SpO2": 96, "RR": 22, "BT": 36.6},
        procedures=[
            "緊急造影CT (大動脈全長、心臓、腹部)",
            "12誘導ECGでType A解離によるRCA巻き込みを評価",
            "両下肢動脈触知、聴診、腹部診察、神経学的所見を即評価",
            "降圧目標SBP 100-120 mmHg、HR 60-80/minでimpulse control",
            "心臓血管外科へ緊急コンサルト",
            "Stanford Aなら心血管外科対応可能施設へ緊急搬送",
        ],
    ),
    "C8": kf(
        diagnoses=[
            "慢性心不全 (HFrEF) の急性増悪",
            "拡張型心筋症",
            "慢性腎臓病 stage 3a",
            "永続性心房細動",
            "高血圧",
        ],
        diagnoses_provisional=[
            "Nohria-Stevenson wet-warm",
            "Forrester II型",
            "CS分類 CS1",
            "塩分・水分管理乱れによる増悪",
            "感染契機、ACS、PEのrule outが必要",
        ],
        scores={
            "NYHA": "IV",
            "Nohria_Stevenson": "wet-warm",
            "Forrester": "II",
            "CS_classification": "CS1",
        },
        medications_to_start=[
            "酸素投与 (経鼻3-5L、改善不十分ならNPPV検討)",
            "フロセミド静注 40mg ボーラス",
            "ニトログリセリン静注 (BP維持下)",
            "経験的抗菌薬 CTRX (細菌性肺炎所見時)",
            "未分画ヘパリン (PE確定時)",
        ],
        medications_to_continue=[
            "エナラプリル 5mg/日 (急性期は循環動態次第)",
            "ビソプロロール 2.5mg/日 (中断回避、循環動態次第)",
            "エドキサバン 30mg/日",
        ],
        medications_to_consider=[
            "サクビトリルバルサルタンへのswitch",
            "スピロノラクトン 25mg/日",
            "ダパグリフロジン 10mg/日",
        ],
        vitals={"BP_sys": 138, "BP_dia": 82, "HR": 96, "SpO2": 91, "RR": 26, "BT": 36.6},
        procedures=[
            "酸素投与、NPPV準備",
            "12誘導ECG",
            "BNP/NT-proBNP、トロポニンI、炎症反応、D-dimer、腎肝機能、血液ガスを採血",
            "胸部X線",
            "経胸壁心エコー",
            "CCU/HCU入院",
            "訪問看護、地域連携、介護保険申請、心臓リハビリを含む退院後支援",
        ],
    ),
}


def load_rows(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def write_rows(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")


def curate(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    missing = sorted(set(MANUAL_KEY_FACTS) - {str(row.get("case_id")) for row in rows})
    if missing:
        raise SystemExit(f"missing cases in base reference: {', '.join(missing)}")

    curated: list[dict[str, Any]] = []
    for row in rows:
        case_id = str(row.get("case_id"))
        if case_id not in MANUAL_KEY_FACTS:
            continue
        key_facts = dict(MANUAL_KEY_FACTS[case_id])
        for vital_key in REQUIRED_VITAL_KEYS:
            if vital_key not in key_facts["vitals"]:
                raise SystemExit(f"{case_id}: missing vital {vital_key}")
        out = dict(row)
        out["reference_version"] = "experiment_manual_v1_2026-05-16"
        out["key_facts"] = key_facts
        curated.append(out)
    return sorted(curated, key=lambda row: row["case_id"])


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, default=DEFAULT_REFERENCE_PATH)
    parser.add_argument("--output", type=Path, default=DEFAULT_REFERENCE_PATH)
    args = parser.parse_args()

    rows = curate(load_rows(args.input))
    write_rows(args.output, rows)
    print(args.output)
    for row in rows:
        kf = row["key_facts"]
        print(
            row["case_id"],
            row["source_case_id"],
            "diagnoses",
            len(kf["diagnoses"]) + len(kf["diagnoses_provisional"]),
            "start",
            len(kf["medications_to_start"]),
            "continue",
            len(kf["medications_to_continue"]),
            "stop",
            len(kf["medications_to_stop"]),
            "consider",
            len(kf["medications_to_consider"]),
        )


if __name__ == "__main__":
    main()
