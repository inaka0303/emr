package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// EncounterRepository は受診データのDB操作を提供する
type EncounterRepository struct {
	db *sql.DB
}

// NewEncounterRepository は新しいEncounterRepositoryを生成する
func NewEncounterRepository(db *sql.DB) *EncounterRepository {
	return &EncounterRepository{db: db}
}

// ListByPatientID は患者IDに紐づく受診一覧を取得する
func (r *EncounterRepository) ListByPatientID(ctx context.Context, patientID int64) ([]model.Encounter, error) {
	query := `SELECT id, patient_id, encounter_date, COALESCE(encounter_type,''), COALESCE(department,''), COALESCE(attending_doctor,''), COALESCE(status,''), COALESCE(chief_complaint,''), created_at, updated_at
		FROM encounters WHERE patient_id = ? ORDER BY encounter_date DESC`

	rows, err := r.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("list encounters: %w", err)
	}
	defer rows.Close()

	var encounters []model.Encounter
	for rows.Next() {
		var e model.Encounter
		if err := rows.Scan(&e.ID, &e.PatientID, &e.EncounterDate, &e.EncounterType, &e.Department, &e.AttendingDoctor, &e.Status, &e.ChiefComplaint, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan encounter: %w", err)
		}
		encounters = append(encounters, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return encounters, nil
}

// GetByID はIDで受診を1件取得する
func (r *EncounterRepository) GetByID(ctx context.Context, id int64) (*model.Encounter, error) {
	query := `SELECT id, patient_id, encounter_date, COALESCE(encounter_type,''), COALESCE(department,''), COALESCE(attending_doctor,''), COALESCE(status,''), COALESCE(chief_complaint,''), created_at, updated_at
		FROM encounters WHERE id = ?`

	var e model.Encounter
	err := r.db.QueryRowContext(ctx, query, id).Scan(&e.ID, &e.PatientID, &e.EncounterDate, &e.EncounterType, &e.Department, &e.AttendingDoctor, &e.Status, &e.ChiefComplaint, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get encounter: %w", err)
	}

	return &e, nil
}

// Create は新しい受診を登録する
func (r *EncounterRepository) Create(ctx context.Context, e *model.Encounter) (int64, error) {
	query := `INSERT INTO encounters (patient_id, encounter_date, encounter_type, department, attending_doctor, status, chief_complaint)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, e.PatientID, e.EncounterDate, e.EncounterType, e.Department, e.AttendingDoctor, e.Status, e.ChiefComplaint)
	if err != nil {
		return 0, fmt.Errorf("create encounter: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}
