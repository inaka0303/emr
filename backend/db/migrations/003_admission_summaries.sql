-- 003_admission_summaries.sql: 入院時サマリの永続化
-- encounter_id 単位で1つのサマリを保持する（upsert想定）

CREATE TABLE IF NOT EXISTS admission_summaries (
    encounter_id INTEGER PRIMARY KEY REFERENCES encounters(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    is_slm_draft BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
