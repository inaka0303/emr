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

// MedicalHistoryHandler は既往歴APIのHTTPハンドラ
type MedicalHistoryHandler struct {
	svc *service.MedicalHistoryService
}

// NewMedicalHistoryHandler は新しいMedicalHistoryHandlerを生成する
func NewMedicalHistoryHandler(svc *service.MedicalHistoryService) *MedicalHistoryHandler {
	return &MedicalHistoryHandler{svc: svc}
}

// ListByPatient は患者の既往歴一覧を返す
// GET /api/patients/:id/medical-history
func (h *MedicalHistoryHandler) ListByPatient(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	histories, err := h.svc.ListByPatientID(c.Request().Context(), patientID)
	if err != nil {
		slog.Error("既往歴一覧取得エラー", "error", err, "patient_id", patientID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "既往歴一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(histories, nil))
}

// Create は新しい既往歴を登録する
// POST /api/patients/:id/medical-history
func (h *MedicalHistoryHandler) Create(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	var history model.MedicalHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	history.PatientID = patientID

	if history.Condition == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（condition）が不足しています"))
	}

	if err := h.svc.Create(c.Request().Context(), &history); err != nil {
		slog.Error("既往歴登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "既往歴の登録に失敗しました"))
	}

	return c.JSON(http.StatusCreated, model.NewSuccessResponse(history, nil))
}

// Update は既往歴を更新する
// PUT /api/medical-history/:id
func (h *MedicalHistoryHandler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な既往歴IDです"))
	}

	var history model.MedicalHistory
	if err := c.Bind(&history); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if err := h.svc.Update(c.Request().Context(), id, &history); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "既往歴が見つかりません"))
		}
		slog.Error("既往歴更新エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "既往歴の更新に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "updated"}, nil))
}

// Delete は既往歴を削除する
// DELETE /api/medical-history/:id
func (h *MedicalHistoryHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な既往歴IDです"))
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "既往歴が見つかりません"))
		}
		slog.Error("既往歴削除エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "既往歴の削除に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "deleted"}, nil))
}
