package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// InterviewRepository は問診記録データのDB操作を提供する
type InterviewRepository struct {
	db *sql.DB
}

// NewInterviewRepository は新しいInterviewRepositoryを生成する
func NewInterviewRepository(db *sql.DB) *InterviewRepository {
	return &InterviewRepository{db: db}
}

// ListByEncounterID は受診IDに紐づく問診記録一覧を取得する（4セクション対応）
func (r *InterviewRepository) ListByEncounterID(ctx context.Context, encounterID int64) ([]model.InterviewNote, error) {
	query := `SELECT id, encounter_id,
		COALESCE(raw_text,''),
		COALESCE(medication_list,''),
		COALESCE(exam_findings,''),
		COALESCE(lab_results,''),
		structured_data, created_at
		FROM interview_notes WHERE encounter_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list interview notes: %w", err)
	}
	defer rows.Close()

	var notes []model.InterviewNote
	for rows.Next() {
		var n model.InterviewNote
		var structuredData sql.NullString
		if err := rows.Scan(&n.ID, &n.EncounterID, &n.RawText, &n.MedicationList, &n.ExamFindings, &n.LabResults, &structuredData, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan interview note: %w", err)
		}
		if structuredData.Valid {
			n.StructuredData = []byte(structuredData.String)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return notes, nil
}

// Create は新しい問診記録を登録する（純粋なINSERT、4セクション対応）
func (r *InterviewRepository) Create(ctx context.Context, note *model.InterviewNote) error {
	query := `INSERT INTO interview_notes
		(encounter_id, raw_text, medication_list, exam_findings, lab_results, structured_data)
		VALUES (?, ?, ?, ?, ?, ?)`

	var structuredData interface{}
	if note.StructuredData != nil {
		structuredData = string(note.StructuredData)
	}

	result, err := r.db.ExecContext(ctx, query,
		note.EncounterID, note.RawText, note.MedicationList, note.ExamFindings, note.LabResults,
		structuredData)
	if err != nil {
		return fmt.Errorf("create interview note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	note.ID = id
	return nil
}

// UpsertByEncounterID は encounter_id に紐づく問診を1件にまとめる（upsert）。
// 既存があれば最古の1件を更新し、他の重複は削除する。
// 医師がアプリで問診を打ち込みながら編集する場合、1秒 debounce 毎にこれが呼ばれても
// 1 encounter = 1 note の不変条件を維持する。
func (r *InterviewRepository) UpsertByEncounterID(ctx context.Context, note *model.InterviewNote) error {
	var existingID int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id FROM interview_notes WHERE encounter_id = ? ORDER BY created_at ASC LIMIT 1`,
		note.EncounterID,
	).Scan(&existingID)

	if errors.Is(err, sql.ErrNoRows) {
		return r.Create(ctx, note)
	}
	if err != nil {
		return fmt.Errorf("check existing interview: %w", err)
	}

	var structuredData interface{}
	if note.StructuredData != nil {
		structuredData = string(note.StructuredData)
	}
	if _, err := r.db.ExecContext(
		ctx,
		`UPDATE interview_notes SET
			raw_text = ?,
			medication_list = ?,
			exam_findings = ?,
			lab_results = ?,
			structured_data = ?
		WHERE id = ?`,
		note.RawText, note.MedicationList, note.ExamFindings, note.LabResults, structuredData, existingID,
	); err != nil {
		return fmt.Errorf("update interview: %w", err)
	}

	// 重複した古いレコードを掃除
	if _, err := r.db.ExecContext(
		ctx,
		`DELETE FROM interview_notes WHERE encounter_id = ? AND id != ?`,
		note.EncounterID, existingID,
	); err != nil {
		return fmt.Errorf("cleanup duplicate interviews: %w", err)
	}

	note.ID = existingID
	return nil
}

// DeleteByEncounterID は encounter_id に紐づくすべての問診記録を削除する
func (r *InterviewRepository) DeleteByEncounterID(ctx context.Context, encounterID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM interview_notes WHERE encounter_id = ?`, encounterID)
	if err != nil {
		return fmt.Errorf("delete interviews: %w", err)
	}
	return nil
}
