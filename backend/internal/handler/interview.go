package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/service"
	"github.com/labstack/echo/v4"
)

// InterviewHandler は問診記録APIのHTTPハンドラ
type InterviewHandler struct {
	svc *service.InterviewService
}

// NewInterviewHandler は新しいInterviewHandlerを生成する
func NewInterviewHandler(svc *service.InterviewService) *InterviewHandler {
	return &InterviewHandler{svc: svc}
}

// ListByEncounter は受診に紐づく問診記録一覧を返す
// GET /api/encounters/:id/interviews
func (h *InterviewHandler) ListByEncounter(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な受診IDです"))
	}

	notes, err := h.svc.ListByEncounterID(c.Request().Context(), encounterID)
	if err != nil {
		slog.Error("問診記録一覧取得エラー", "error", err, "encounter_id", encounterID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "問診記録一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(notes, nil))
}

// Create は新しい問診記録を登録する
// POST /api/encounters/:id/interviews
func (h *InterviewHandler) Create(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な受診IDです"))
	}

	var note model.InterviewNote
	if err := c.Bind(&note); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	note.EncounterID = encounterID

	if note.RawText == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（raw_text）が不足しています"))
	}

	if err := h.svc.Create(c.Request().Context(), &note); err != nil {
		slog.Error("問診記録登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "問診記録の登録に失敗しました"))
	}

	return c.JSON(http.StatusCreated, model.NewSuccessResponse(note, nil))
}
