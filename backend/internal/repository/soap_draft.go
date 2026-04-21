package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SOAPDraft はSLMで生成したSOAPドラフトのキャッシュレコード
type SOAPDraft struct {
	EncounterID  int64
	Subjective   string
	Objective    string
	Assessment   string
	Plan         string
	Model        string
	GenerationMS int64
}

// SOAPDraftRepository はSOAPドラフトキャッシュのDB操作を提供する
type SOAPDraftRepository struct {
	db *sql.DB
}

func NewSOAPDraftRepository(db *sql.DB) *SOAPDraftRepository {
	return &SOAPDraftRepository{db: db}
}

var ErrSOAPDraftNotFound = errors.New("soap draft not found")

// GetByEncounterID は encounter_id に紐づくSOAPドラフトを取得する
func (r *SOAPDraftRepository) GetByEncounterID(ctx context.Context, encounterID int64) (*SOAPDraft, error) {
	query := `SELECT encounter_id, subjective, objective, assessment, plan, model, generation_ms
		FROM soap_drafts WHERE encounter_id = ?`
	row := r.db.QueryRowContext(ctx, query, encounterID)

	var d SOAPDraft
	err := row.Scan(&d.EncounterID, &d.Subjective, &d.Objective, &d.Assessment, &d.Plan, &d.Model, &d.GenerationMS)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSOAPDraftNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get soap draft: %w", err)
	}
	return &d, nil
}

// Upsert はSOAPドラフトを保存する（既存があれば更新）
func (r *SOAPDraftRepository) Upsert(ctx context.Context, d *SOAPDraft) error {
	query := `
		INSERT INTO soap_drafts (encounter_id, subjective, objective, assessment, plan, model, generation_ms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(encounter_id) DO UPDATE SET
			subjective = excluded.subjective,
			objective = excluded.objective,
			assessment = excluded.assessment,
			plan = excluded.plan,
			model = excluded.model,
			generation_ms = excluded.generation_ms,
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, d.EncounterID, d.Subjective, d.Objective, d.Assessment, d.Plan, d.Model, d.GenerationMS)
	if err != nil {
		return fmt.Errorf("upsert soap draft: %w", err)
	}
	return nil
}

// Delete は encounter_id のSOAPドラフトキャッシュを削除する
func (r *SOAPDraftRepository) Delete(ctx context.Context, encounterID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM soap_drafts WHERE encounter_id = ?`, encounterID)
	if err != nil {
		return fmt.Errorf("delete soap draft: %w", err)
	}
	return nil
}
