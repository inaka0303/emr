package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type AdmissionSummary struct {
	EncounterID int64
	Content     string
	Author      string
	IsSLMDraft  bool
}

type AdmissionSummaryRepository struct {
	db *sql.DB
}

func NewAdmissionSummaryRepository(db *sql.DB) *AdmissionSummaryRepository {
	return &AdmissionSummaryRepository{db: db}
}

var ErrAdmissionSummaryNotFound = errors.New("admission summary not found")

func (r *AdmissionSummaryRepository) GetByEncounterID(ctx context.Context, encounterID int64) (*AdmissionSummary, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT encounter_id, content, author, is_slm_draft FROM admission_summaries WHERE encounter_id = ?`,
		encounterID,
	)
	var s AdmissionSummary
	err := row.Scan(&s.EncounterID, &s.Content, &s.Author, &s.IsSLMDraft)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAdmissionSummaryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get admission summary: %w", err)
	}
	return &s, nil
}

func (r *AdmissionSummaryRepository) Upsert(ctx context.Context, s *AdmissionSummary) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admission_summaries (encounter_id, content, author, is_slm_draft, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(encounter_id) DO UPDATE SET
			content = excluded.content,
			author = excluded.author,
			is_slm_draft = excluded.is_slm_draft,
			updated_at = CURRENT_TIMESTAMP`,
		s.EncounterID, s.Content, s.Author, s.IsSLMDraft,
	)
	if err != nil {
		return fmt.Errorf("upsert admission summary: %w", err)
	}
	return nil
}
