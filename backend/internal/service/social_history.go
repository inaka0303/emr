package service

import (
	"context"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
)

// SocialHistoryService は社会歴関連のビジネスロジックを提供する
type SocialHistoryService struct {
	repo *repository.SocialHistoryRepository
}

// NewSocialHistoryService は新しいSocialHistoryServiceを生成する
func NewSocialHistoryService(repo *repository.SocialHistoryRepository) *SocialHistoryService {
	return &SocialHistoryService{repo: repo}
}

// ListByPatientID は患者IDに紐づく社会歴一覧を取得する
func (s *SocialHistoryService) ListByPatientID(ctx context.Context, patientID int64) ([]model.SocialHistory, error) {
	return s.repo.ListByPatientID(ctx, patientID)
}

// Create は新しい社会歴を登録する
func (s *SocialHistoryService) Create(ctx context.Context, history *model.SocialHistory) error {
	return s.repo.Create(ctx, history)
}

// Update は社会歴を更新する
func (s *SocialHistoryService) Update(ctx context.Context, id int64, history *model.SocialHistory) error {
	return s.repo.Update(ctx, id, history)
}

// Delete は社会歴を削除する
func (s *SocialHistoryService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
