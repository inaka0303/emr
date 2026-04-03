package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// MedicalHistoryRepository は既往歴データのDB操作を提供する
type MedicalHistoryRepository struct {
	db *sql.DB
}

// NewMedicalHistoryRepository は新しいMedicalHistoryRepositoryを生成する
func NewMedicalHistoryRepository(db *sql.DB) *MedicalHistoryRepository {
	return &MedicalHistoryRepository{db: db}
}

// ListByPatientID は患者IDに紐づく既往歴一覧を取得する
func (r *MedicalHistoryRepository) ListByPatientID(ctx context.Context, patientID int64) ([]model.MedicalHistory, error) {
	query := `SELECT id, patient_id, condition, COALESCE(onset_date,''), COALESCE(status,''), COALESCE(notes,''), created_at, updated_at
		FROM medical_history WHERE patient_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("list medical history: %w", err)
	}
	defer rows.Close()

	var histories []model.MedicalHistory
	for rows.Next() {
		var h model.MedicalHistory
		if err := rows.Scan(&h.ID, &h.PatientID, &h.Condition, &h.OnsetDate, &h.Status, &h.Notes, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan medical history: %w", err)
		}
		histories = append(histories, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return histories, nil
}

// Create は新しい既往歴を登録する
func (r *MedicalHistoryRepository) Create(ctx context.Context, history *model.MedicalHistory) error {
	query := `INSERT INTO medical_history (patient_id, condition, onset_date, status, notes) VALUES (?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, history.PatientID, history.Condition, history.OnsetDate, history.Status, history.Notes)
	if err != nil {
		return fmt.Errorf("create medical history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	history.ID = id
	return nil
}

// Update は既往歴を更新する
func (r *MedicalHistoryRepository) Update(ctx context.Context, id int64, history *model.MedicalHistory) error {
	query := `UPDATE medical_history SET condition=?, onset_date=?, status=?, notes=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

	result, err := r.db.ExecContext(ctx, query, history.Condition, history.OnsetDate, history.Status, history.Notes, id)
	if err != nil {
		return fmt.Errorf("update medical history: %w", err)
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

// Delete は既往歴を削除する
func (r *MedicalHistoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM medical_history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete medical history: %w", err)
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
