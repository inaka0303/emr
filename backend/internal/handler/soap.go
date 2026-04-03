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

// SOAPHandler はSOAP記録APIのHTTPハンドラ
type SOAPHandler struct {
	svc *service.SOAPService
}

// NewSOAPHandler は新しいSOAPHandlerを生成する
func NewSOAPHandler(svc *service.SOAPService) *SOAPHandler {
	return &SOAPHandler{svc: svc}
}

// GetByEncounter は受診に紐づくSOAP記録を返す
// GET /api/encounters/:id/soap
func (h *SOAPHandler) GetByEncounter(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な受診IDです"))
	}

	notes, err := h.svc.GetByEncounterID(c.Request().Context(), encounterID)
	if err != nil {
		slog.Error("SOAP記録取得エラー", "error", err, "encounter_id", encounterID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "SOAP記録の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(notes, nil))
}

// Create は新しいSOAP記録を登録する
// POST /api/encounters/:id/soap
func (h *SOAPHandler) Create(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な受診IDです"))
	}

	var n model.SOAPNote
	if err := c.Bind(&n); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	n.EncounterID = encounterID

	id, err := h.svc.Create(c.Request().Context(), &n)
	if err != nil {
		slog.Error("SOAP記録登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "SOAP記録の登録に失敗しました"))
	}

	n.ID = id
	return c.JSON(http.StatusCreated, model.NewSuccessResponse(n, nil))
}

// Update はSOAP記録を更新する
// PUT /api/soap/:id
func (h *SOAPHandler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効なSOAP記録IDです"))
	}

	var n model.SOAPNote
	if err := c.Bind(&n); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if err := h.svc.Update(c.Request().Context(), id, &n); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "SOAP記録が見つかりません"))
		}
		slog.Error("SOAP記録更新エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "SOAP記録の更新に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "updated"}, nil))
}
