package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// SOAPService はSOAP記録関連のビジネスロジックを提供する
type SOAPService struct {
	repo *repository.SOAPRepository
}

// NewSOAPService は新しいSOAPServiceを生成する
func NewSOAPService(repo *repository.SOAPRepository) *SOAPService {
	return &SOAPService{repo: repo}
}

// GetByEncounterID は受診IDに紐づくSOAP記録を取得する
func (s *SOAPService) GetByEncounterID(ctx context.Context, encounterID int64) ([]model.SOAPNote, error) {
	return s.repo.GetByEncounterID(ctx, encounterID)
}

// Create は新しいSOAP記録を登録する
func (s *SOAPService) Create(ctx context.Context, n *model.SOAPNote) (int64, error) {
	return s.repo.Create(ctx, n)
}

// Update はSOAP記録を更新する
func (s *SOAPService) Update(ctx context.Context, id int64, n *model.SOAPNote) error {
	return s.repo.Update(ctx, id, n)
}
