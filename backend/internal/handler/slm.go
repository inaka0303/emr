package handler

import (
	"log/slog"
	"net/http"

	"github.com/example/ehr-demo/internal/slm"
	"github.com/labstack/echo/v4"
)

// SLMHandler はSLM連携APIのHTTPハンドラ
type SLMHandler struct {
	client *slm.Client
}

// NewSLMHandler は新しいSLMHandlerを生成する
func NewSLMHandler(client *slm.Client) *SLMHandler {
	return &SLMHandler{client: client}
}

// slmMeta はSLMレスポンスのメタ情報
type slmMeta struct {
	Model     string `json:"model"`
	IsMock    bool   `json:"is_mock"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// slmResponse はSLM APIの統一レスポンス
type slmResponse struct {
	Data interface{} `json:"data"`
	Meta slmMeta     `json:"meta"`
}

// suggestSOAPRequest はSOAP提案リクエスト
type suggestSOAPRequest struct {
	InterviewText string `json:"interview_text"`
}

// suggestSummaryRequest はサマリー提案リクエスト
type suggestSummaryRequest struct {
	InterviewText string `json:"interview_text"`
	Category      string `json:"category"`
}

// SuggestSOAP はSOAP形式の記録提案を生成する
// POST /api/slm/suggest/soap
func (h *SLMHandler) SuggestSOAP(c echo.Context) error {
	var req suggestSOAPRequest
	if err := c.Bind(&req); err != nil {
		slog.Error("リクエストバインドエラー", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "リクエストの形式が不正です",
		})
	}

	if req.InterviewText == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "interview_text は必須です",
		})
	}

	suggestion, isMock, latency, err := h.client.GenerateSOAP(c.Request().Context(), req.InterviewText)
	if err != nil {
		slog.Error("SOAP提案生成エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "SOAP提案の生成に失敗しました",
		})
	}

	return c.JSON(http.StatusOK, slmResponse{
		Data: suggestion,
		Meta: slmMeta{
			Model:     h.client.ModelName(),
			IsMock:    isMock,
			LatencyMs: latency.Milliseconds(),
		},
	})
}

// SuggestSummary は家族歴/社会歴の要約提案を生成する
// POST /api/slm/suggest/summary
func (h *SLMHandler) SuggestSummary(c echo.Context) error {
	var req suggestSummaryRequest
	if err := c.Bind(&req); err != nil {
		slog.Error("リクエストバインドエラー", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "リクエストの形式が不正です",
		})
	}

	if req.InterviewText == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "interview_text は必須です",
		})
	}

	if req.Category != "family_history" && req.Category != "social_history" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "category は 'family_history' または 'social_history' を指定してください",
		})
	}

	suggestion, isMock, latency, err := h.client.GenerateSummary(c.Request().Context(), req.InterviewText, req.Category)
	if err != nil {
		slog.Error("サマリー提案生成エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "サマリー提案の生成に失敗しました",
		})
	}

	return c.JSON(http.StatusOK, slmResponse{
		Data: suggestion,
		Meta: slmMeta{
			Model:     h.client.ModelName(),
			IsMock:    isMock,
			LatencyMs: latency.Milliseconds(),
		},
	})
}

// Health はSLM推論サーバーの接続状態を返す
// GET /api/slm/health
func (h *SLMHandler) Health(c echo.Context) error {
	status := "mock_mode"
	if h.client.IsHealthy() {
		status = "connected"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": map[string]string{
			"status":  status,
			"api_url": h.client.BaseURL(),
			"model":   h.client.ModelName(),
		},
	})
}
