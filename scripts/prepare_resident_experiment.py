#!/usr/bin/env python3
"""Prepare resident R01-R16 attempts in the current EMR experiment DB.

The resident cohort reuses the same 8 ACI-JP-Cardio cases as the student
experiment, but keeps independent patients/encounters/attempt IDs so student
data are not modified.
"""

from __future__ import annotations

import argparse
import json
import shutil
import sqlite3
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo


DEFAULT_DB = Path("/tmp/emr-exp-run/ehr-demo.db")

RESIDENT_ATTEMPTS = [
    # R1
    ("R01", "R1", "C1", "JC-AMI-A", "ai", 1),
    ("R02", "R1", "C6", "JC-PE-A", "control", 2),
    ("R03", "R1", "C7", "JC-AD-B", "ai", 3),
    ("R04", "R1", "C4", "JC-AHF-B", "control", 4),
    # R2
    ("R05", "R2", "C5", "JC-AMI-B", "control", 1),
    ("R06", "R2", "C2", "JC-PE-B", "ai", 2),
    ("R07", "R2", "C3", "JC-AD-A", "control", 3),
    ("R08", "R2", "C8", "JC-AHF-A", "ai", 4),
    # R3
    ("R09", "R3", "C6", "JC-PE-A", "ai", 1),
    ("R10", "R3", "C1", "JC-AMI-A", "control", 2),
    ("R11", "R3", "C4", "JC-AHF-B", "ai", 3),
    ("R12", "R3", "C7", "JC-AD-B", "control", 4),
    # R4
    ("R13", "R4", "C2", "JC-PE-B", "control", 1),
    ("R14", "R4", "C5", "JC-AMI-B", "ai", 2),
    ("R15", "R4", "C8", "JC-AHF-A", "control", 3),
    ("R16", "R4", "C3", "JC-AD-A", "ai", 4),
]

CANONICAL_ATTEMPT_BY_CASE = {
    "C1": "A01",
    "C2": "A02",
    "C3": "A03",
    "C4": "A04",
    "C5": "A17",
    "C6": "A21",
    "C7": "A19",
    "C8": "A23",
}


def dict_factory(cursor: sqlite3.Cursor, row: sqlite3.Row) -> dict[str, object]:
    return {col[0]: row[idx] for idx, col in enumerate(cursor.description)}


def now_jst() -> str:
    return datetime.now(ZoneInfo("Asia/Tokyo")).strftime("%Y%m%d_%H%M%S")


def backup_db(db_path: Path) -> Path:
    backup = db_path.with_name(f"{db_path.stem}.resident_backup_{now_jst()}{db_path.suffix}")
    shutil.copy2(db_path, backup)
    return backup


def existing_resident_usage(conn: sqlite3.Connection) -> list[dict[str, object]]:
    return conn.execute(
        """
        SELECT ea.attempt_id,
               COUNT(DISTINCT sn.id) AS soap_notes,
               COUNT(DISTINCT ev.id) AS events,
               COUNT(DISTINCT sd.encounter_id) AS drafts
        FROM experiment_attempts ea
        LEFT JOIN soap_notes sn ON sn.encounter_id = ea.encounter_id
        LEFT JOIN experiment_events ev ON ev.attempt_id = ea.attempt_id
        LEFT JOIN soap_drafts sd ON sd.encounter_id = ea.encounter_id
        WHERE ea.attempt_id LIKE 'R%'
        GROUP BY ea.attempt_id
        ORDER BY ea.attempt_id
        """
    ).fetchall()


def delete_existing_resident_rows(conn: sqlite3.Connection) -> None:
    attempts = conn.execute(
        """
        SELECT attempt_id, patient_id, encounter_id
        FROM experiment_attempts
        WHERE attempt_id LIKE 'R%'
        ORDER BY attempt_id
        """
    ).fetchall()
    if not attempts:
        return

    patient_ids = [row["patient_id"] for row in attempts]
    encounter_ids = [row["encounter_id"] for row in attempts]
    attempt_ids = [row["attempt_id"] for row in attempts]

    for attempt_id in attempt_ids:
        conn.execute("DELETE FROM experiment_events WHERE attempt_id = ?", (attempt_id,))
        conn.execute("DELETE FROM experiment_attempts WHERE attempt_id = ?", (attempt_id,))
    for encounter_id in encounter_ids:
        conn.execute("DELETE FROM admission_summaries WHERE encounter_id = ?", (encounter_id,))
        conn.execute("DELETE FROM soap_drafts WHERE encounter_id = ?", (encounter_id,))
        conn.execute("DELETE FROM soap_notes WHERE encounter_id = ?", (encounter_id,))
        conn.execute("DELETE FROM interview_notes WHERE encounter_id = ?", (encounter_id,))
        conn.execute("DELETE FROM encounters WHERE id = ?", (encounter_id,))
    for patient_id in patient_ids:
        conn.execute("DELETE FROM medical_history WHERE patient_id = ?", (patient_id,))
        conn.execute("DELETE FROM family_history WHERE patient_id = ?", (patient_id,))
        conn.execute("DELETE FROM social_history WHERE patient_id = ?", (patient_id,))
        conn.execute("DELETE FROM patients WHERE id = ?", (patient_id,))


def clone_table_rows(
    conn: sqlite3.Connection,
    table: str,
    source_patient_id: int,
    target_patient_id: int,
    columns: list[str],
) -> None:
    rows = conn.execute(
        f"SELECT {', '.join(columns)} FROM {table} WHERE patient_id = ? ORDER BY id",
        (source_patient_id,),
    ).fetchall()
    placeholders = ",".join(["?"] * len(columns))
    for row in rows:
        values = [target_patient_id if col == "patient_id" else row[col] for col in columns]
        conn.execute(
            f"INSERT INTO {table} ({', '.join(columns)}) VALUES ({placeholders})",
            values,
        )


def build_resident_structured_data(raw: str | None, attempt: tuple[str, str, str, str, str, int]) -> str:
    attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order = attempt
    try:
        data = json.loads(raw or "{}")
    except json.JSONDecodeError:
        data = {}
    experiment = data.setdefault("experiment", {})
    experiment.update(
        {
            "attempt_id": attempt_id,
            "subject_id": subject_id,
            "case_id": case_id,
            "source_case_id": source_case_id,
            "intervention": intervention,
            "sequence_order": sequence_order,
            "cohort": "resident",
        }
    )
    return json.dumps(data, ensure_ascii=False, separators=(",", ":"))


def insert_resident_attempt(conn: sqlite3.Connection, attempt: tuple[str, str, str, str, str, int]) -> None:
    attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order = attempt
    source_attempt_id = CANONICAL_ATTEMPT_BY_CASE[case_id]
    source = conn.execute(
        """
        SELECT ea.patient_id, ea.encounter_id, p.*, e.id AS source_encounter_id,
               e.encounter_date, e.encounter_type, e.department, e.chief_complaint
        FROM experiment_attempts ea
        JOIN patients p ON p.id = ea.patient_id
        JOIN encounters e ON e.id = ea.encounter_id
        WHERE ea.attempt_id = ?
        """,
        (source_attempt_id,),
    ).fetchone()
    if source is None:
        raise RuntimeError(f"source attempt not found: {source_attempt_id}")

    patient_cur = conn.execute(
        """
        INSERT INTO patients
          (mrn, name, name_kana, birth_date, gender, blood_type, phone, address,
           emergency_contact_name, emergency_contact_phone)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            f"RES-{attempt_id}",
            f"研修医実験患者 {attempt_id}",
            f"ケンシュウイジッケンカンジャ {attempt_id}",
            source["birth_date"],
            source["gender"],
            source["blood_type"],
            "",
            "研修医実験症例",
            "",
            "",
        ),
    )
    patient_id = int(patient_cur.lastrowid)

    encounter_cur = conn.execute(
        """
        INSERT INTO encounters
          (patient_id, encounter_date, encounter_type, department, attending_doctor, status, chief_complaint)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """,
        (
            patient_id,
            source["encounter_date"],
            source["encounter_type"],
            source["department"],
            "研修医実験担当",
            "進行中",
            source["chief_complaint"],
        ),
    )
    encounter_id = int(encounter_cur.lastrowid)

    interview = conn.execute(
        """
        SELECT raw_text, structured_data, medication_list, exam_findings, lab_results
        FROM interview_notes
        WHERE encounter_id = ?
        ORDER BY id
        LIMIT 1
        """,
        (source["source_encounter_id"],),
    ).fetchone()
    if interview is None:
        raise RuntimeError(f"source interview not found: {source_attempt_id}")
    conn.execute(
        """
        INSERT INTO interview_notes
          (encounter_id, raw_text, medication_list, exam_findings, lab_results, structured_data)
        VALUES (?, ?, ?, ?, ?, ?)
        """,
        (
            encounter_id,
            interview["raw_text"],
            interview["medication_list"],
            interview["exam_findings"],
            interview["lab_results"],
            build_resident_structured_data(interview["structured_data"], attempt),
        ),
    )

    clone_table_rows(
        conn,
        "medical_history",
        int(source["patient_id"]),
        patient_id,
        ["patient_id", "condition", "onset_date", "status", "notes"],
    )
    clone_table_rows(
        conn,
        "family_history",
        int(source["patient_id"]),
        patient_id,
        ["patient_id", "relation", "condition", "notes", "is_slm_suggested"],
    )
    clone_table_rows(
        conn,
        "social_history",
        int(source["patient_id"]),
        patient_id,
        ["patient_id", "category", "description", "notes", "is_slm_suggested"],
    )

    conn.execute(
        """
        INSERT INTO experiment_attempts
          (attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order, patient_id, encounter_id, notes)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            attempt_id,
            subject_id,
            case_id,
            source_case_id,
            intervention,
            sequence_order,
            patient_id,
            encounter_id,
            "resident_cohort=2026-05-21",
        ),
    )


def verify(conn: sqlite3.Connection) -> None:
    rows = conn.execute(
        """
        SELECT attempt_id, subject_id, sequence_order, case_id, source_case_id, intervention,
               patient_id, encounter_id, status
        FROM experiment_attempts
        WHERE attempt_id LIKE 'R%'
        ORDER BY attempt_id
        """
    ).fetchall()
    if len(rows) != len(RESIDENT_ATTEMPTS):
        raise RuntimeError(f"expected {len(RESIDENT_ATTEMPTS)} resident attempts, got {len(rows)}")
    for row in rows:
        soap_row = conn.execute(
            """
            SELECT COUNT(*)
            FROM soap_notes
            WHERE encounter_id = ?
            """,
            (row["encounter_id"],),
        ).fetchone()
        event_row = conn.execute(
            "SELECT COUNT(*) FROM experiment_events WHERE attempt_id = ?",
            (row["attempt_id"],),
        ).fetchone()
        n_soap = int(next(iter(soap_row.values())))
        n_events = int(next(iter(event_row.values())))
        if n_soap or n_events:
            raise RuntimeError(f"{row['attempt_id']} is not clean: soap={n_soap}, events={n_events}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", type=Path, default=DEFAULT_DB)
    parser.add_argument("--force", action="store_true", help="Delete existing clean R attempts and recreate them.")
    parser.add_argument("--no-backup", action="store_true")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    if not args.db.exists():
        raise SystemExit(f"DB not found: {args.db}")

    backup = None
    if not args.no_backup:
        backup = backup_db(args.db)

    conn = sqlite3.connect(args.db)
    conn.row_factory = dict_factory
    conn.execute("PRAGMA foreign_keys = ON")

    existing = existing_resident_usage(conn)
    if existing and not args.force:
        raise SystemExit("Resident attempts already exist. Use --force only before data collection.")
    if existing:
        used = [r for r in existing if int(r["soap_notes"]) > 0]
        if used:
            raise SystemExit(f"Refusing to overwrite resident attempts with saved SOAP notes: {used}")

    with conn:
        delete_existing_resident_rows(conn)
        for attempt in RESIDENT_ATTEMPTS:
            insert_resident_attempt(conn, attempt)
        verify(conn)

    print(f"prepared {len(RESIDENT_ATTEMPTS)} resident attempts in {args.db}")
    if backup:
        print(f"backup: {backup}")
    for row in conn.execute(
        """
        SELECT attempt_id, subject_id, sequence_order, case_id, source_case_id, intervention
        FROM experiment_attempts
        WHERE attempt_id LIKE 'R%'
        ORDER BY attempt_id
        """
    ):
        print(
            f"{row['attempt_id']} {row['subject_id']} order={row['sequence_order']} "
            f"{row['case_id']} {row['source_case_id']} {row['intervention']}"
        )


if __name__ == "__main__":
    main()
