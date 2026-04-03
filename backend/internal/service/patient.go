package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// PatientService は患者関連のビジネスロジックを提供する
type PatientService struct {
	repo *repository.PatientRepository
}

// NewPatientService は新しいPatientServiceを生成する
func NewPatientService(repo *repository.PatientRepository) *PatientService {
	return &PatientService{repo: repo}
}

// ListPatients は患者一覧を取得する
func (s *PatientService) ListPatients(ctx context.Context, query string, page, limit int) ([]model.Patient, int, error) {
	return s.repo.ListPatients(ctx, query, page, limit)
}

// GetPatient はIDで患者を取得する
func (s *PatientService) GetPatient(ctx context.Context, id int64) (*model.Patient, error) {
	return s.repo.GetPatient(ctx, id)
}

// CreatePatient は新しい患者を登録する
func (s *PatientService) CreatePatient(ctx context.Context, p *model.Patient) (int64, error) {
	return s.repo.CreatePatient(ctx, p)
}

// UpdatePatient は患者情報を更新する
func (s *PatientService) UpdatePatient(ctx context.Context, id int64, p *model.Patient) error {
	return s.repo.UpdatePatient(ctx, id, p)
}
