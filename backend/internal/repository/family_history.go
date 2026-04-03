package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// FamilyHistoryRepository は家族歴データのDB操作を提供する
type FamilyHistoryRepository struct {
	db *sql.DB
}

// NewFamilyHistoryRepository は新しいFamilyHistoryRepositoryを生成する
func NewFamilyHistoryRepository(db *sql.DB) *FamilyHistoryRepository {
	return &FamilyHistoryRepository{db: db}
}

// ListByPatientID は患者IDに紐づく家族歴一覧を取得する
func (r *FamilyHistoryRepository) ListByPatientID(ctx context.Context, patientID int64) ([]model.FamilyHistory, error) {
	query := `SELECT id, patient_id, relation, condition, COALESCE(notes,''), is_slm_suggested, created_at, updated_at
		FROM family_history WHERE patient_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("list family history: %w", err)
	}
	defer rows.Close()

	var histories []model.FamilyHistory
	for rows.Next() {
		var h model.FamilyHistory
		if err := rows.Scan(&h.ID, &h.PatientID, &h.Relation, &h.Condition, &h.Notes, &h.IsSLMSuggested, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan family history: %w", err)
		}
		histories = append(histories, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return histories, nil
}

// Create は新しい家族歴を登録する
func (r *FamilyHistoryRepository) Create(ctx context.Context, history *model.FamilyHistory) error {
	query := `INSERT INTO family_history (patient_id, relation, condition, notes, is_slm_suggested) VALUES (?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, history.PatientID, history.Relation, history.Condition, history.Notes, history.IsSLMSuggested)
	if err != nil {
		return fmt.Errorf("create family history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	history.ID = id
	return nil
}

// Update は家族歴を更新する
func (r *FamilyHistoryRepository) Update(ctx context.Context, id int64, history *model.FamilyHistory) error {
	query := `UPDATE family_history SET relation=?, condition=?, notes=?, is_slm_suggested=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

	result, err := r.db.ExecContext(ctx, query, history.Relation, history.Condition, history.Notes, history.IsSLMSuggested, id)
	if err != nil {
		return fmt.Errorf("update family history: %w", err)
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

// Delete は家族歴を削除する
func (r *FamilyHistoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM family_history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete family history: %w", err)
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
