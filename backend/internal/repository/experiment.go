package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// ExperimentRepository は実験 attempt とイベントログを扱う。
type ExperimentRepository struct {
	db *sql.DB
}

func NewExperimentRepository(db *sql.DB) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

func (r *ExperimentRepository) ListAttempts(ctx context.Context) ([]model.ExperimentAttempt, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order,
		       patient_id, encounter_id, status,
		       COALESCE(started_at,''), COALESCE(ended_at,''),
		       duration_sec, interruption_sec, ai_wait_ms,
		       ai_candidate_count, ai_accept_count, ai_edit_count, ai_reject_count,
		       COALESCE(notes,''), created_at, updated_at
		FROM experiment_attempts
		ORDER BY attempt_id`)
	if err != nil {
		return nil, fmt.Errorf("list experiment attempts: %w", err)
	}
	defer rows.Close()

	var attempts []model.ExperimentAttempt
	for rows.Next() {
		a, err := scanExperimentAttempt(rows)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate experiment attempts: %w", err)
	}
	return attempts, nil
}

func (r *ExperimentRepository) GetAttempt(ctx context.Context, attemptID string) (*model.ExperimentAttempt, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order,
		       patient_id, encounter_id, status,
		       COALESCE(started_at,''), COALESCE(ended_at,''),
		       duration_sec, interruption_sec, ai_wait_ms,
		       ai_candidate_count, ai_accept_count, ai_edit_count, ai_reject_count,
		       COALESCE(notes,''), created_at, updated_at
		FROM experiment_attempts
		WHERE attempt_id = ?`, attemptID)
	a, err := scanExperimentAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ExperimentRepository) GetAttemptByEncounter(ctx context.Context, encounterID int64) (*model.ExperimentAttempt, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order,
		       patient_id, encounter_id, status,
		       COALESCE(started_at,''), COALESCE(ended_at,''),
		       duration_sec, interruption_sec, ai_wait_ms,
		       ai_candidate_count, ai_accept_count, ai_edit_count, ai_reject_count,
		       COALESCE(notes,''), created_at, updated_at
		FROM experiment_attempts
		WHERE encounter_id = ?`, encounterID)
	a, err := scanExperimentAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ExperimentRepository) StartAttempt(ctx context.Context, attemptID string) (*model.ExperimentAttempt, error) {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE experiment_attempts
		SET status = 'in_progress',
		    started_at = COALESCE(started_at, CURRENT_TIMESTAMP),
		    updated_at = CURRENT_TIMESTAMP
		WHERE attempt_id = ?`, attemptID); err != nil {
		return nil, fmt.Errorf("start experiment attempt: %w", err)
	}
	return r.GetAttempt(ctx, attemptID)
}

func (r *ExperimentRepository) FinishAttempt(ctx context.Context, attemptID string, input model.ExperimentFinishInput) (*model.ExperimentAttempt, error) {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE experiment_attempts
		SET status = 'finished',
		    ended_at = CURRENT_TIMESTAMP,
		    duration_sec = ?,
		    interruption_sec = ?,
		    ai_wait_ms = ?,
		    ai_candidate_count = ?,
		    ai_accept_count = ?,
		    ai_edit_count = ?,
		    ai_reject_count = ?,
		    notes = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE attempt_id = ?`,
		input.DurationSec,
		input.InterruptionSec,
		input.AIWaitMS,
		input.AICandidateCount,
		input.AIAcceptCount,
		input.AIEditCount,
		input.AIRejectCount,
		input.Notes,
		attemptID,
	); err != nil {
		return nil, fmt.Errorf("finish experiment attempt: %w", err)
	}
	return r.GetAttempt(ctx, attemptID)
}

func (r *ExperimentRepository) AddAIUsage(ctx context.Context, attemptID string, waitMS int64, candidateDelta int) error {
	if attemptID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE experiment_attempts
		SET ai_wait_ms = ai_wait_ms + ?,
		    ai_candidate_count = ai_candidate_count + ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE attempt_id = ?`, waitMS, candidateDelta, attemptID)
	if err != nil {
		return fmt.Errorf("add ai usage: %w", err)
	}
	return nil
}

func (r *ExperimentRepository) RecordEvent(ctx context.Context, attemptID, eventType string, payload interface{}) error {
	if attemptID == "" || eventType == "" {
		return nil
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal experiment event payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO experiment_events (attempt_id, event_type, payload_json)
		VALUES (?, ?, ?)`, attemptID, eventType, string(payloadBytes))
	if err != nil {
		return fmt.Errorf("record experiment event: %w", err)
	}
	return nil
}

type experimentAttemptScanner interface {
	Scan(dest ...interface{}) error
}

func scanExperimentAttempt(row experimentAttemptScanner) (*model.ExperimentAttempt, error) {
	var a model.ExperimentAttempt
	err := row.Scan(
		&a.AttemptID,
		&a.SubjectID,
		&a.CaseID,
		&a.SourceCaseID,
		&a.Intervention,
		&a.SequenceOrder,
		&a.PatientID,
		&a.EncounterID,
		&a.Status,
		&a.StartedAt,
		&a.EndedAt,
		&a.DurationSec,
		&a.InterruptionSec,
		&a.AIWaitMS,
		&a.AICandidateCount,
		&a.AIAcceptCount,
		&a.AIEditCount,
		&a.AIRejectCount,
		&a.Notes,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan experiment attempt: %w", err)
	}
	return &a, nil
}
