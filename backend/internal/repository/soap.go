package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// SOAPRepository はSOAP記録のDB操作を提供する
type SOAPRepository struct {
	db *sql.DB
}

// NewSOAPRepository は新しいSOAPRepositoryを生成する
func NewSOAPRepository(db *sql.DB) *SOAPRepository {
	return &SOAPRepository{db: db}
}

// GetByEncounterID は受診IDに紐づくSOAP記録を取得する
func (r *SOAPRepository) GetByEncounterID(ctx context.Context, encounterID int64) ([]model.SOAPNote, error) {
	query := `SELECT id, encounter_id, COALESCE(author,''), COALESCE(subjective,''), COALESCE(objective,''), COALESCE(assessment,''), COALESCE(plan,''), is_slm_suggested, created_at, updated_at
		FROM soap_notes WHERE encounter_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list soap notes: %w", err)
	}
	defer rows.Close()

	var notes []model.SOAPNote
	for rows.Next() {
		var n model.SOAPNote
		if err := rows.Scan(&n.ID, &n.EncounterID, &n.Author, &n.Subjective, &n.Objective, &n.Assessment, &n.Plan, &n.IsSLMSuggested, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan soap note: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return notes, nil
}

// Create は新しいSOAP記録を登録する
func (r *SOAPRepository) Create(ctx context.Context, n *model.SOAPNote) (int64, error) {
	query := `INSERT INTO soap_notes (encounter_id, author, subjective, objective, assessment, plan, is_slm_suggested)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, n.EncounterID, n.Author, n.Subjective, n.Objective, n.Assessment, n.Plan, n.IsSLMSuggested)
	if err != nil {
		return 0, fmt.Errorf("create soap note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// Update はSOAP記録を更新する
func (r *SOAPRepository) Update(ctx context.Context, id int64, n *model.SOAPNote) error {
	query := `UPDATE soap_notes SET author=?, subjective=?, objective=?, assessment=?, plan=?, is_slm_suggested=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

	result, err := r.db.ExecContext(ctx, query, n.Author, n.Subjective, n.Objective, n.Assessment, n.Plan, n.IsSLMSuggested, id)
	if err != nil {
		return fmt.Errorf("update soap note: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetByID はIDでSOAP記録を1件取得する
func (r *SOAPRepository) GetByID(ctx context.Context, id int64) (*model.SOAPNote, error) {
	query := `SELECT id, encounter_id, COALESCE(author,''), COALESCE(subjective,''), COALESCE(objective,''), COALESCE(assessment,''), COALESCE(plan,''), is_slm_suggested, created_at, updated_at
		FROM soap_notes WHERE id = ?`

	var n model.SOAPNote
	err := r.db.QueryRowContext(ctx, query, id).Scan(&n.ID, &n.EncounterID, &n.Author, &n.Subjective, &n.Objective, &n.Assessment, &n.Plan, &n.IsSLMSuggested, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get soap note: %w", err)
	}

	return &n, nil
}
