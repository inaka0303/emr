-- 005_experiment_attempts.sql: ACI-JP-Cardio human experiment tracking
-- attempt_id (A01-A32) を実験の主キーとして扱い、被験者・症例・介入条件・対象 encounter を固定する。

CREATE TABLE IF NOT EXISTS experiment_attempts (
    attempt_id TEXT PRIMARY KEY,
    subject_id TEXT NOT NULL,
    case_id TEXT NOT NULL,
    source_case_id TEXT NOT NULL,
    intervention TEXT NOT NULL CHECK (intervention IN ('ai', 'control')),
    sequence_order INTEGER NOT NULL,
    patient_id INTEGER NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    encounter_id INTEGER NOT NULL UNIQUE REFERENCES encounters(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'ready' CHECK (status IN ('ready', 'in_progress', 'finished', 'abandoned')),
    started_at DATETIME,
    ended_at DATETIME,
    duration_sec INTEGER NOT NULL DEFAULT 0,
    interruption_sec INTEGER NOT NULL DEFAULT 0,
    ai_wait_ms INTEGER NOT NULL DEFAULT 0,
    ai_candidate_count INTEGER NOT NULL DEFAULT 0,
    ai_accept_count INTEGER NOT NULL DEFAULT 0,
    ai_edit_count INTEGER NOT NULL DEFAULT 0,
    ai_reject_count INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS experiment_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    attempt_id TEXT NOT NULL REFERENCES experiment_attempts(attempt_id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_experiment_attempts_subject ON experiment_attempts(subject_id, sequence_order);
CREATE INDEX IF NOT EXISTS idx_experiment_attempts_case ON experiment_attempts(case_id, intervention);
CREATE INDEX IF NOT EXISTS idx_experiment_attempts_encounter ON experiment_attempts(encounter_id);
CREATE INDEX IF NOT EXISTS idx_experiment_events_attempt ON experiment_events(attempt_id, created_at);
