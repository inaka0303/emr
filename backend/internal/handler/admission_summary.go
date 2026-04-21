package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/example/ehr-demo/internal/repository"
	"github.com/labstack/echo/v4"
)

type AdmissionSummaryHandler struct {
	repo *repository.AdmissionSummaryRepository
}

func NewAdmissionSummaryHandler(repo *repository.AdmissionSummaryRepository) *AdmissionSummaryHandler {
	return &AdmissionSummaryHandler{repo: repo}
}

type admissionSummaryRequest struct {
	Content    string `json:"content"`
	Author     string `json:"author"`
	IsSLMDraft bool   `json:"is_slm_draft"`
}

type admissionSummaryResponse struct {
	EncounterID int64  `json:"encounter_id"`
	Content     string `json:"content"`
	Author      string `json:"author"`
	IsSLMDraft  bool   `json:"is_slm_draft"`
}

// GET /api/encounters/:id/admission-summary
func (h *AdmissionSummaryHandler) Get(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "無効な受診IDです"})
	}
	s, err := h.repo.GetByEncounterID(c.Request().Context(), encounterID)
	if errors.Is(err, repository.ErrAdmissionSummaryNotFound) {
		return c.JSON(http.StatusOK, map[string]interface{}{"data": nil})
	}
	if err != nil {
		slog.Error("入院時サマリ取得エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "取得に失敗しました"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": admissionSummaryResponse{
			EncounterID: s.EncounterID,
			Content:     s.Content,
			Author:      s.Author,
			IsSLMDraft:  s.IsSLMDraft,
		},
	})
}

// POST /api/encounters/:id/admission-summary
func (h *AdmissionSummaryHandler) Save(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "無効な受診IDです"})
	}
	var req admissionSummaryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "リクエスト形式が不正です"})
	}
	s := &repository.AdmissionSummary{
		EncounterID: encounterID,
		Content:     req.Content,
		Author:      req.Author,
		IsSLMDraft:  req.IsSLMDraft,
	}
	if err := h.repo.Upsert(c.Request().Context(), s); err != nil {
		slog.Error("入院時サマリ保存エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存に失敗しました"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": admissionSummaryResponse{
			EncounterID: s.EncounterID,
			Content:     s.Content,
			Author:      s.Author,
			IsSLMDraft:  s.IsSLMDraft,
		},
	})
}
