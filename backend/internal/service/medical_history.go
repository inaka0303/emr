package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// MedicalHistoryService は既往歴関連のビジネスロジックを提供する
type MedicalHistoryService struct {
	repo *repository.MedicalHistoryRepository
}

// NewMedicalHistoryService は新しいMedicalHistoryServiceを生成する
func NewMedicalHistoryService(repo *repository.MedicalHistoryRepository) *MedicalHistoryService {
	return &MedicalHistoryService{repo: repo}
}

// ListByPatientID は患者IDに紐づく既往歴一覧を取得する
func (s *MedicalHistoryService) ListByPatientID(ctx context.Context, patientID int64) ([]model.MedicalHistory, error) {
	return s.repo.ListByPatientID(ctx, patientID)
}

// Create は新しい既往歴を登録する
func (s *MedicalHistoryService) Create(ctx context.Context, history *model.MedicalHistory) error {
	return s.repo.Create(ctx, history)
}

// Update は既往歴を更新する
func (s *MedicalHistoryService) Update(ctx context.Context, id int64, history *model.MedicalHistory) error {
	return s.repo.Update(ctx, id, history)
}

// Delete は既往歴を削除する
func (s *MedicalHistoryService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
