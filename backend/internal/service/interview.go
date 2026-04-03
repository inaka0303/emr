package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// InterviewService は問診記録関連のビジネスロジックを提供する
type InterviewService struct {
	repo *repository.InterviewRepository
}

// NewInterviewService は新しいInterviewServiceを生成する
func NewInterviewService(repo *repository.InterviewRepository) *InterviewService {
	return &InterviewService{repo: repo}
}

// ListByEncounterID は受診IDに紐づく問診記録一覧を取得する
func (s *InterviewService) ListByEncounterID(ctx context.Context, encounterID int64) ([]model.InterviewNote, error) {
	return s.repo.ListByEncounterID(ctx, encounterID)
}

// Create は新しい問診記録を登録する
func (s *InterviewService) Create(ctx context.Context, note *model.InterviewNote) error {
	return s.repo.Create(ctx, note)
}
