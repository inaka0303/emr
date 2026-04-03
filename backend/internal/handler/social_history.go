package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/example/ehr-demo/internal/service"
	"github.com/labstack/echo/v4"
)

// SocialHistoryHandler は社会歴APIのHTTPハンドラ
type SocialHistoryHandler struct {
	svc *service.SocialHistoryService
}

// NewSocialHistoryHandler は新しいSocialHistoryHandlerを生成する
func NewSocialHistoryHandler(svc *service.SocialHistoryService) *SocialHistoryHandler {
	return &SocialHistoryHandler{svc: svc}
}

// ListByPatient は患者の社会歴一覧を返す
// GET /api/patients/:id/social-history
func (h *SocialHistoryHandler) ListByPatient(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	histories, err := h.svc.ListByPatientID(c.Request().Context(), patientID)
	if err != nil {
		slog.Error("社会歴一覧取得エラー", "error", err, "patient_id", patientID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "社会歴一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(histories, nil))
}

// Create は新しい社会歴を登録する
// POST /api/patients/:id/social-history
func (h *SocialHistoryHandler) Create(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	var history model.SocialHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	history.PatientID = patientID

	if history.Category == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（category）が不足しています"))
	}

	if err := h.svc.Create(c.Request().Context(), &history); err != nil {
		slog.Error("社会歴登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "社会歴の登録に失敗しました"))
	}

	return c.JSON(http.StatusCreated, model.NewSuccessResponse(history, nil))
}

// Update は社会歴を更新する
// PUT /api/social-history/:id
func (h *SocialHistoryHandler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な社会歴IDです"))
	}

	var history model.SocialHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if err := h.svc.Update(c.Request().Context(), id, &history); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "社会歴が見つかりません"))
		}
		slog.Error("社会歴更新エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "社会歴の更新に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "updated"}, nil))
}

// Delete は社会歴を削除する
// DELETE /api/social-history/:id
func (h *SocialHistoryHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な社会歴IDです"))
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "社会歴が見つかりません"))
		}
		slog.Error("社会歴削除エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "社会歴の削除に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "deleted"}, nil))
}
