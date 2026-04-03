package slm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const modelName = "qwen3.5-0.8b-medical"

// Client はSLM推論サーバーとの通信を担当する
type Client struct {
	baseURL    string
	httpClient *http.Client
	useMock    bool
	mu         sync.RWMutex
}

// ChatCompletionRequest はOpenAI互換リクエスト
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

// Message はチャットメッセージ
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse はOpenAI互換レスポンス
type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice はレスポンスの選択肢
type Choice struct {
	Message Message `json:"message"`
}

// SOAPSuggestion はSOAP提案レスポンス
type SOAPSuggestion struct {
	Subjective string `json:"subjective"`
	Objective  string `json:"objective"`
	Assessment string `json:"assessment"`
	Plan       string `json:"plan"`
}

// SummarySuggestionItem はサマリー提案の個別項目
type SummarySuggestionItem struct {
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// SummarySuggestion はサマリー提案レスポンス
type SummarySuggestion struct {
	Suggestions []SummarySuggestionItem `json:"suggestions"`
}

// NewClient は新しいSLMクライアントを生成する
// 起動時にbaseURLへの接続テストを行い、失敗時はモックモードに設定する
// バックグラウンドで定期的にヘルスチェックを行い、接続状態を更新する
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		useMock: true, // 初期はモック
	}

	// 初回接続テスト
	if c.checkHealth() {
		c.useMock = false
		slog.Info("SLM推論サーバー接続成功", "url", baseURL)
	} else {
		slog.Warn("SLM推論サーバー未接続、モックモードで起動", "url", baseURL)
	}

	// バックグラウンドヘルスチェック（60秒ごと）
	go c.backgroundHealthCheck()

	return c
}

// checkHealth は推論サーバーへの接続状態を確認する
func (c *Client) checkHealth() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// backgroundHealthCheck はバックグラウンドで定期的にヘルスチェックを行い、接続状態を更新する
func (c *Client) backgroundHealthCheck() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		healthy := c.checkHealth()
		c.mu.Lock()
		if c.useMock && healthy {
			slog.Info("SLM推論サーバー復帰検出、実SLMモードに切替")
		} else if !c.useMock && !healthy {
			slog.Warn("SLM推論サーバー接続断検出、モックモードに切替")
		}
		c.useMock = !healthy
		c.mu.Unlock()
	}
}

// IsHealthy は推論サーバーの接続状態を返す
func (c *Client) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.useMock
}

// BaseURL はベースURLを返す
func (c *Client) BaseURL() string {
	return c.baseURL
}

// ModelName はモデル名を返す
func (c *Client) ModelName() string {
	return modelName
}

// GenerateSOAP はSOAP提案を生成する
// 返り値: 提案、isMock、レイテンシ、エラー
func (c *Client) GenerateSOAP(ctx context.Context, interviewText string) (*SOAPSuggestion, bool, time.Duration, error) {
	start := time.Now()
	truncated := truncateText(interviewText, 50)
	slog.Info("SOAP提案リクエスト", "text_preview", truncated)

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		result := generateMockSOAP(interviewText)
		latency := time.Since(start)
		slog.Info("SOAP提案完了(モック)", "latency_ms", latency.Milliseconds())
		return result, true, latency, nil
	}

	// OpenAI互換APIにリクエスト送信
	systemPrompt := "あなたは医療記録の作成を支援するAIアシスタントです。入力された問診内容から、SOAP形式（Subjective, Objective, Assessment, Plan）の記録を提案してください。以下のJSON形式で返してください:\n{\"subjective\": \"...\", \"objective\": \"...\", \"assessment\": \"...\", \"plan\": \"...\"}"

	result, err := c.callChatCompletion(ctx, systemPrompt, interviewText)
	if err != nil {
		slog.Warn("SLM API呼び出し失敗。モックにフォールバック", "error", err)

		mockResult := generateMockSOAP(interviewText)
		latency := time.Since(start)
		slog.Info("SOAP提案完了(フォールバックモック)", "latency_ms", latency.Milliseconds())
		return mockResult, true, latency, nil
	}

	var suggestion SOAPSuggestion
	if err := json.Unmarshal([]byte(result), &suggestion); err != nil {
		slog.Warn("SLMレスポンスのパースに失敗。モックにフォールバック", "error", err, "raw", result)
		mockResult := generateMockSOAP(interviewText)
		latency := time.Since(start)
		return mockResult, true, latency, nil
	}

	latency := time.Since(start)
	slog.Info("SOAP提案完了(SLM)", "latency_ms", latency.Milliseconds())
	return &suggestion, false, latency, nil
}

// GenerateSummary は家族歴/社会歴の要約提案を生成する
// 返り値: 提案、isMock、レイテンシ、エラー
func (c *Client) GenerateSummary(ctx context.Context, interviewText string, category string) (*SummarySuggestion, bool, time.Duration, error) {
	start := time.Now()
	truncated := truncateText(interviewText, 50)
	slog.Info("サマリー提案リクエスト", "category", category, "text_preview", truncated)

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		result := generateMockSummary(interviewText, category)
		latency := time.Since(start)
		slog.Info("サマリー提案完了(モック)", "category", category, "latency_ms", latency.Milliseconds())
		return result, true, latency, nil
	}

	var systemPrompt string
	switch category {
	case "family_history":
		systemPrompt = "あなたは医療記録の作成を支援するAIアシスタントです。入力された問診内容から、家族歴の情報を抽出してください。以下のJSON形式で返してください:\n{\"suggestions\": [{\"field\": \"relation\", \"value\": \"続柄\", \"confidence\": 0.95}, {\"field\": \"condition\", \"value\": \"疾患名\", \"confidence\": 0.90}]}"
	case "social_history":
		systemPrompt = "あなたは医療記録の作成を支援するAIアシスタントです。入力された問診内容から、社会歴の情報（喫煙、飲酒、職業、運動など）を抽出してください。以下のJSON形式で返してください:\n{\"suggestions\": [{\"field\": \"category\", \"value\": \"カテゴリ\", \"confidence\": 0.95}, {\"field\": \"description\", \"value\": \"詳細\", \"confidence\": 0.88}]}"
	default:
		return nil, false, 0, fmt.Errorf("不明なカテゴリ: %s", category)
	}

	result, err := c.callChatCompletion(ctx, systemPrompt, interviewText)
	if err != nil {
		slog.Warn("SLM API呼び出し失敗。モックにフォールバック", "error", err)

		mockResult := generateMockSummary(interviewText, category)
		latency := time.Since(start)
		slog.Info("サマリー提案完了(フォールバックモック)", "category", category, "latency_ms", latency.Milliseconds())
		return mockResult, true, latency, nil
	}

	var suggestion SummarySuggestion
	if err := json.Unmarshal([]byte(result), &suggestion); err != nil {
		slog.Warn("SLMレスポンスのパースに失敗。モックにフォールバック", "error", err, "raw", result)
		mockResult := generateMockSummary(interviewText, category)
		latency := time.Since(start)
		return mockResult, true, latency, nil
	}

	latency := time.Since(start)
	slog.Info("サマリー提案完了(SLM)", "category", category, "latency_ms", latency.Milliseconds())
	return &suggestion, false, latency, nil
}

// callChatCompletion はOpenAI互換APIを呼び出す
func (c *Client) callChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := ChatCompletionRequest{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   1024,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("リクエストのシリアライズに失敗: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("リクエスト作成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API呼び出しに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("APIがステータス %d を返しました: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("レスポンスのデコードに失敗: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("レスポンスにChoicesが含まれていません")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// truncateText はテキストを指定文字数で切り詰める
func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}
