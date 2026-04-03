package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// FamilyHistoryService は家族歴関連のビジネスロジックを提供する
type FamilyHistoryService struct {
	repo *repository.FamilyHistoryRepository
}

// NewFamilyHistoryService は新しいFamilyHistoryServiceを生成する
func NewFamilyHistoryService(repo *repository.FamilyHistoryRepository) *FamilyHistoryService {
	return &FamilyHistoryService{repo: repo}
}

// ListByPatientID は患者IDに紐づく家族歴一覧を取得する
func (s *FamilyHistoryService) ListByPatientID(ctx context.Context, patientID int64) ([]model.FamilyHistory, error) {
	return s.repo.ListByPatientID(ctx, patientID)
}

// Create は新しい家族歴を登録する
func (s *FamilyHistoryService) Create(ctx context.Context, history *model.FamilyHistory) error {
	return s.repo.Create(ctx, history)
}

// Update は家族歴を更新する
func (s *FamilyHistoryService) Update(ctx context.Context, id int64, history *model.FamilyHistory) error {
	return s.repo.Update(ctx, id, history)
}

// Delete は家族歴を削除する
func (s *FamilyHistoryService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
