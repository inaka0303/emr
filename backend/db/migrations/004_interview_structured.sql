-- 004_interview_structured.sql: 問診を4セクションに分離
-- 医学ワークフローに整合: 問診は患者から聞くこと、お薬は薬手帳、
-- 所見は医師が取るもの、検査はシステムから来るもの、と情報源で分離する。
-- 既存の raw_text は【問診記録】フィールドとしてそのまま扱う。

ALTER TABLE interview_notes ADD COLUMN medication_list TEXT NOT NULL DEFAULT '';
ALTER TABLE interview_notes ADD COLUMN exam_findings TEXT NOT NULL DEFAULT '';
ALTER TABLE interview_notes ADD COLUMN lab_results TEXT NOT NULL DEFAULT '';
