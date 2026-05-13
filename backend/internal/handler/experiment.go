package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/labstack/echo/v4"
)

// ExperimentHandler は A01-A32 の実験 attempt を扱う。
type ExperimentHandler struct {
	repo *repository.ExperimentRepository
}

func NewExperimentHandler(repo *repository.ExperimentRepository) *ExperimentHandler {
	return &ExperimentHandler{repo: repo}
}

func (h *ExperimentHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/attempts", h.ListAttempts)
	g.GET("/attempts/:id", h.GetAttempt)
	g.POST("/attempts/:id/start", h.StartAttempt)
	g.POST("/attempts/:id/finish", h.FinishAttempt)
	g.POST("/attempts/:id/events", h.RecordEvent)
}

func (h *ExperimentHandler) ListAttempts(c echo.Context) error {
	attempts, err := h.repo.ListAttempts(c.Request().Context())
	if err != nil {
		slog.Error("実験attempt一覧取得エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "実験attempt一覧の取得に失敗しました"))
	}
	return c.JSON(http.StatusOK, model.NewSuccessResponse(attempts, nil))
}

func (h *ExperimentHandler) GetAttempt(c echo.Context) error {
	attempt, err := h.repo.GetAttempt(c.Request().Context(), strings.ToUpper(c.Param("id")))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "実験attemptが見つかりません"))
		}
		slog.Error("実験attempt取得エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "実験attemptの取得に失敗しました"))
	}
	return c.JSON(http.StatusOK, model.NewSuccessResponse(attempt, nil))
}

func (h *ExperimentHandler) StartAttempt(c echo.Context) error {
	attemptID := strings.ToUpper(c.Param("id"))
	attempt, err := h.repo.StartAttempt(c.Request().Context(), attemptID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "実験attemptが見つかりません"))
		}
		slog.Error("実験attempt開始エラー", "error", err, "attempt_id", attemptID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "実験attemptの開始に失敗しました"))
	}
	_ = h.repo.RecordEvent(c.Request().Context(), attemptID, "attempt_started", map[string]string{"source": "frontend"})
	return c.JSON(http.StatusOK, model.NewSuccessResponse(attempt, nil))
}

func (h *ExperimentHandler) FinishAttempt(c echo.Context) error {
	attemptID := strings.ToUpper(c.Param("id"))
	var input model.ExperimentFinishInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}
	attempt, err := h.repo.FinishAttempt(c.Request().Context(), attemptID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "実験attemptが見つかりません"))
		}
		slog.Error("実験attempt終了エラー", "error", err, "attempt_id", attemptID)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "実験attemptの終了に失敗しました"))
	}
	_ = h.repo.RecordEvent(c.Request().Context(), attemptID, "attempt_finished", input)
	return c.JSON(http.StatusOK, model.NewSuccessResponse(attempt, nil))
}

type experimentEventRequest struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func (h *ExperimentHandler) RecordEvent(c echo.Context) error {
	attemptID := strings.ToUpper(c.Param("id"))
	var req experimentEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}
	req.EventType = strings.TrimSpace(req.EventType)
	if req.EventType == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "event_type は必須です"))
	}

	var payload interface{} = map[string]interface{}{}
	if len(req.Payload) > 0 {
		var decoded interface{}
		if err := json.Unmarshal(req.Payload, &decoded); err != nil {
			return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "payload はJSONオブジェクトまたは配列で指定してください"))
		}
		payload = decoded
	}
	if err := h.repo.RecordEvent(c.Request().Context(), attemptID, req.EventType, payload); err != nil {
		slog.Error("実験イベント記録エラー", "error", err, "attempt_id", attemptID, "event_type", req.EventType)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "実験イベントの記録に失敗しました"))
	}
	return c.JSON(http.StatusCreated, model.NewSuccessResponse(map[string]string{"status": "recorded"}, nil))
}
