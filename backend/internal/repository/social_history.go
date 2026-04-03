package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// SocialHistoryRepository は社会歴データのDB操作を提供する
type SocialHistoryRepository struct {
	db *sql.DB
}

// NewSocialHistoryRepository は新しいSocialHistoryRepositoryを生成する
func NewSocialHistoryRepository(db *sql.DB) *SocialHistoryRepository {
	return &SocialHistoryRepository{db: db}
}

// ListByPatientID は患者IDに紐づく社会歴一覧を取得する
func (r *SocialHistoryRepository) ListByPatientID(ctx context.Context, patientID int64) ([]model.SocialHistory, error) {
	query := `SELECT id, patient_id, category, COALESCE(description,''), COALESCE(notes,''), is_slm_suggested, created_at, updated_at
		FROM social_history WHERE patient_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("list social history: %w", err)
	}
	defer rows.Close()

	var histories []model.SocialHistory
	for rows.Next() {
		var h model.SocialHistory
		if err := rows.Scan(&h.ID, &h.PatientID, &h.Category, &h.Description, &h.Notes, &h.IsSLMSuggested, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan social history: %w", err)
		}
		histories = append(histories, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return histories, nil
}

// Create は新しい社会歴を登録する
func (r *SocialHistoryRepository) Create(ctx context.Context, history *model.SocialHistory) error {
	query := `INSERT INTO social_history (patient_id, category, description, notes, is_slm_suggested) VALUES (?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, history.PatientID, history.Category, history.Description, history.Notes, history.IsSLMSuggested)
	if err != nil {
		return fmt.Errorf("create social history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	history.ID = id
	return nil
}

// Update は社会歴を更新する
func (r *SocialHistoryRepository) Update(ctx context.Context, id int64, history *model.SocialHistory) error {
	query := `UPDATE social_history SET category=?, description=?, notes=?, is_slm_suggested=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

	result, err := r.db.ExecContext(ctx, query, history.Category, history.Description, history.Notes, history.IsSLMSuggested, id)
	if err != nil {
		return fmt.Errorf("update social history: %w", err)
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

// Delete は社会歴を削除する
func (r *SocialHistoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM social_history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete social history: %w", err)
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
