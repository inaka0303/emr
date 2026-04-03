package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ehr-demo/internal/model"
)

// ErrNotFound はリソースが見つからない場合のエラー
var ErrNotFound = errors.New("not found")

// PatientRepository は患者データのDB操作を提供する
type PatientRepository struct {
	db *sql.DB
}

// NewPatientRepository は新しいPatientRepositoryを生成する
func NewPatientRepository(db *sql.DB) *PatientRepository {
	return &PatientRepository{db: db}
}

// ListPatients は患者一覧を検索・ページネーション付きで取得する
func (r *PatientRepository) ListPatients(ctx context.Context, query string, page, limit int) ([]model.Patient, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 件数カウント
	var total int
	countQuery := "SELECT COUNT(*) FROM patients"
	args := []interface{}{}

	if query != "" {
		countQuery += " WHERE name LIKE ? OR name_kana LIKE ? OR mrn LIKE ?"
		like := "%" + query + "%"
		args = append(args, like, like, like)
	}

	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count patients: %w", err)
	}

	// データ取得
	dataQuery := "SELECT id, mrn, name, name_kana, birth_date, gender, COALESCE(blood_type,''), COALESCE(phone,''), COALESCE(address,''), COALESCE(emergency_contact_name,''), COALESCE(emergency_contact_phone,''), created_at, updated_at FROM patients"
	if query != "" {
		dataQuery += " WHERE name LIKE ? OR name_kana LIKE ? OR mrn LIKE ?"
	}
	dataQuery += " ORDER BY id DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list patients: %w", err)
	}
	defer rows.Close()

	var patients []model.Patient
	for rows.Next() {
		var p model.Patient
		if err := rows.Scan(&p.ID, &p.MRN, &p.Name, &p.NameKana, &p.BirthDate, &p.Gender, &p.BloodType, &p.Phone, &p.Address, &p.EmergencyContactName, &p.EmergencyContactPhone, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan patient: %w", err)
		}
		patients = append(patients, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	return patients, total, nil
}

// GetPatient はIDで患者を1件取得する
func (r *PatientRepository) GetPatient(ctx context.Context, id int64) (*model.Patient, error) {
	query := "SELECT id, mrn, name, name_kana, birth_date, gender, COALESCE(blood_type,''), COALESCE(phone,''), COALESCE(address,''), COALESCE(emergency_contact_name,''), COALESCE(emergency_contact_phone,''), created_at, updated_at FROM patients WHERE id = ?"

	var p model.Patient
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.MRN, &p.Name, &p.NameKana, &p.BirthDate, &p.Gender, &p.BloodType, &p.Phone, &p.Address, &p.EmergencyContactName, &p.EmergencyContactPhone, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get patient: %w", err)
	}

	return &p, nil
}

// CreatePatient は新しい患者を登録する
func (r *PatientRepository) CreatePatient(ctx context.Context, p *model.Patient) (int64, error) {
	query := `INSERT INTO patients (mrn, name, name_kana, birth_date, gender, blood_type, phone, address, emergency_contact_name, emergency_contact_phone)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, p.MRN, p.Name, p.NameKana, p.BirthDate, p.Gender, p.BloodType, p.Phone, p.Address, p.EmergencyContactName, p.EmergencyContactPhone)
	if err != nil {
		return 0, fmt.Errorf("create patient: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// UpdatePatient は患者情報を更新する
func (r *PatientRepository) UpdatePatient(ctx context.Context, id int64, p *model.Patient) error {
	query := `UPDATE patients SET name=?, name_kana=?, birth_date=?, gender=?, blood_type=?, phone=?, address=?, emergency_contact_name=?, emergency_contact_phone=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

	result, err := r.db.ExecContext(ctx, query, p.Name, p.NameKana, p.BirthDate, p.Gender, p.BloodType, p.Phone, p.Address, p.EmergencyContactName, p.EmergencyContactPhone, id)
	if err != nil {
		return fmt.Errorf("update patient: %w", err)
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
