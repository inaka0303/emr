package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

// RAGHandler は RAG検索サーバー (Python FastAPI, port 8082) への proxy.
// SOAPのP生成時などで「ガイドラインの根拠」を表示するために使う。
type RAGHandler struct {
	baseURL    string
	httpClient *http.Client
}

func NewRAGHandler() *RAGHandler {
	url := os.Getenv("RAG_API_URL")
	if url == "" {
		url = "http://localhost:8082"
	}
	return &RAGHandler{
		baseURL:    url,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type ragSearchRequest struct {
	Query string `json:"query"`
	N     int    `json:"n,omitempty"`
}

type ragSearchResult struct {
	ParentID  string  `json:"parent_id"`
	Text      string  `json:"text"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
	ChildHits int     `json:"child_hits"`
}

type ragSearchResponse struct {
	Query     string            `json:"query"`
	Results   []ragSearchResult `json:"results"`
	ElapsedMs int               `json:"elapsed_ms"`
}

// Search は RAG 検索をプロキシする
// POST /api/rag/search  body: {"query": "...", "n": 5}
func (h *RAGHandler) Search(c echo.Context) error {
	var req ragSearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "リクエスト形式が不正です"})
	}
	if req.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "queryは必須です"})
	}
	if req.N <= 0 {
		req.N = 5
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost,
		h.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "request build failed"})
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		slog.Warn("RAG server未接続", "error", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": fmt.Sprintf("RAG server未接続: %s （python3.11 rag_server.pyを起動してください）", h.baseURL),
		})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return c.JSON(resp.StatusCode, map[string]string{"error": string(respBody)})
	}

	var out ragSearchResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "RAGレスポンスの解析失敗"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": out})
}

// Health は RAG server のヘルスチェック
// GET /api/rag/health
func (h *RAGHandler) Health(c echo.Context) error {
	httpReq, _ := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, h.baseURL+"/health", nil)
	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data": map[string]string{"status": "unavailable", "reason": err.Error()},
		})
	}
	defer resp.Body.Close()
	var payload map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	return c.JSON(http.StatusOK, map[string]interface{}{"data": payload})
}
