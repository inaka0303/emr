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

// EncounterHandler は受診APIのHTTPハンドラ
type EncounterHandler struct {
	svc *service.EncounterService
}

// NewEncounterHandler は新しいEncounterHandlerを生成する
func NewEncounterHandler(svc *service.EncounterService) *EncounterHandler {
	return &EncounterHandler{svc: svc}
}

// ListByPatient は患者の受診履歴一覧を返す
// GET /api/patients/:id/encounters
func (h *EncounterHandler) ListByPatient(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	encounters, err := h.svc.ListByPatientID(c.Request().Context(), patientID)
	if err != nil {
		slog.Error("受診一覧取得エラー", "error", err, "patient_id", patientID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "受診一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(encounters, nil))
}

// Create は新しい受診を登録する
// POST /api/patients/:id/encounters
func (h *EncounterHandler) Create(c echo.Context) error {
	patientID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	var e model.Encounter
	if err := c.Bind(&e); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	e.PatientID = patientID

	if e.EncounterDate == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（encounter_date）が不足しています"))
	}

	id, err := h.svc.Create(c.Request().Context(), &e)
	if err != nil {
		slog.Error("受診登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "受診の登録に失敗しました"))
	}

	e.ID = id
	return c.JSON(http.StatusCreated, model.NewSuccessResponse(e, nil))
}

// Get は指定IDの受診詳細を返す
// GET /api/encounters/:id
func (h *EncounterHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な受診IDです"))
	}

	encounter, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "受診が見つかりません"))
		}
		slog.Error("受診取得エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "受診情報の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(encounter, nil))
}
