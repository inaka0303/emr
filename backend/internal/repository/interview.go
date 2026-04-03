package repository

import (
	"context"
	"database/sql"
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

// ListByEncounterID は受診IDに紐づく問診記録一覧を取得する
func (r *InterviewRepository) ListByEncounterID(ctx context.Context, encounterID int64) ([]model.InterviewNote, error) {
	query := `SELECT id, encounter_id, COALESCE(raw_text,''), structured_data, created_at
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
		if err := rows.Scan(&n.ID, &n.EncounterID, &n.RawText, &structuredData, &n.CreatedAt); err != nil {
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

// Create は新しい問診記録を登録する
func (r *InterviewRepository) Create(ctx context.Context, note *model.InterviewNote) error {
	query := `INSERT INTO interview_notes (encounter_id, raw_text, structured_data) VALUES (?, ?, ?)`

	var structuredData interface{}
	if note.StructuredData != nil {
		structuredData = string(note.StructuredData)
	}

	result, err := r.db.ExecContext(ctx, query, note.EncounterID, note.RawText, structuredData)
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
