-- 002_soap_drafts.sql: SOAPドラフト永続キャッシュ
-- SLMで生成したSOAPドラフトを encounter_id 単位で保存し、
-- 次回以降の参照は即座にキャッシュから返す（8〜15秒の生成遅延を回避）。

CREATE TABLE IF NOT EXISTS soap_drafts (
    encounter_id INTEGER PRIMARY KEY REFERENCES encounters(id) ON DELETE CASCADE,
    subjective TEXT NOT NULL DEFAULT '',
    objective TEXT NOT NULL DEFAULT '',
    assessment TEXT NOT NULL DEFAULT '',
    plan TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    generation_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
