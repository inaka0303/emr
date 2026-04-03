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

// FamilyHistoryHandler は家族歴APIのHTTPハンドラ
type FamilyHistoryHandler struct {
	svc *service.FamilyHistoryService
}

// NewFamilyHistoryHandler は新しいFamilyHistoryHandlerを生成する
func NewFamilyHistoryHandler(svc *service.FamilyHistoryService) *FamilyHistoryHandler {
	return &FamilyHistoryHandler{svc: svc}
}

// ListByPatient は患者の家族歴一覧を返す
// GET /api/patients/:id/family-history
func (h *FamilyHistoryHandler) ListByPatient(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	histories, err := h.svc.ListByPatientID(c.Request().Context(), patientID)
	if err != nil {
		slog.Error("家族歴一覧取得エラー", "error", err, "patient_id", patientID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "家族歴一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(histories, nil))
}

// Create は新しい家族歴を登録する
// POST /api/patients/:id/family-history
func (h *FamilyHistoryHandler) Create(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	var history model.FamilyHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	history.PatientID = patientID

	if history.Relation == "" || history.Condition == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（relation, condition）が不足しています"))
	}

	if err := h.svc.Create(c.Request().Context(), &history); err != nil {
		slog.Error("家族歴登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "家族歴の登録に失敗しました"))
	}

	return c.JSON(http.StatusCreated, model.NewSuccessResponse(history, nil))
}

// Update は家族歴を更新する
// PUT /api/family-history/:id
func (h *FamilyHistoryHandler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な家族歴IDです"))
	}

	var history model.FamilyHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if err := h.svc.Update(c.Request().Context(), id, &history); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "家族歴が見つかりません"))
		}
		slog.Error("家族歴更新エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "家族歴の更新に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "updated"}, nil))
}

// Delete は家族歴を削除する
// DELETE /api/family-history/:id
func (h *FamilyHistoryHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な家族歴IDです"))
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "家族歴が見つかりません"))
		}
		slog.Error("家族歴削除エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "家族歴の削除に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "deleted"}, nil))
}
