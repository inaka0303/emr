package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// EncounterService は受診関連のビジネスロジックを提供する
type EncounterService struct {
	repo *repository.EncounterRepository
}

// NewEncounterService は新しいEncounterServiceを生成する
func NewEncounterService(repo *repository.EncounterRepository) *EncounterService {
	return &EncounterService{repo: repo}
}

// ListByPatientID は患者IDに紐づく受診一覧を取得する
func (s *EncounterService) ListByPatientID(ctx context.Context, patientID int64) ([]model.Encounter, error) {
	return s.repo.ListByPatientID(ctx, patientID)
}

// GetByID はIDで受診を取得する
func (s *EncounterService) GetByID(ctx context.Context, id int64) (*model.Encounter, error) {
	return s.repo.GetByID(ctx, id)
}

// Create は新しい受診を登録する
func (s *EncounterService) Create(ctx context.Context, e *model.Encounter) (int64, error) {
	return s.repo.Create(ctx, e)
}
