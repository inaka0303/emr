package slm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const modelName = "qwen3.5-4b-medical"

// LoRA ID constants (llama-server --lora の順序に対応)
//   id 0: sft_4b_nocpt_A_lora       → suggest（カルテ途中→続き補完）
//   id 1: sft_soap_full_4b_lora     → SOAP全体（問診→SOAP全文）
//   id 2: sft_admission_summary_4b  → 入院時サマリ
const (
	LoRASuggestID    = 0
	LoRASOAPFullID   = 1
	LoRAAdmissionID  = 2
)

// Client はSLM推論サーバーとの通信を担当する
//
// LoRA切替方式: llama-serverはper-request `lora`を受け付けるが実効しないため、
// POST /lora-adapters でグローバルscaleを書き換える方式を採る。
// 全SLM呼び出しは loraMu でserialize（同時実行中のLoRA差し替え事故を防ぐ）。
// activeLoRAID はキャッシュ用で、前回と同じなら再設定しない。
type Client struct {
	baseURL       string
	httpClient    *http.Client
	useMock       bool
	mu            sync.RWMutex
	loraMu        sync.Mutex
	activeLoRAID  int
}

// LoRAAdapterConfig はglobal scale切替のリクエストboy要素
type LoRAAdapterConfig struct {
	ID    int     `json:"id"`
	Scale float64 `json:"scale"`
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
			Timeout: 60 * time.Second,
		},
		useMock:      true, // 初期はモック
		activeLoRAID: -1,   // 未設定
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
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

// SectionCallback は GenerateSOAPStreaming が各セクション生成完了時に呼ぶコールバック
//   section: "S" | "O" | "A" | "P"
//   text: そのセクションの本文（冒頭の "S:" 等は除去済み）
type SectionCallback func(section, text string)

// GenerateSOAPStreaming は GenerateSOAP と同じ 4 セクション逐次生成を行うが、
// 各セクション完了時に onSection コールバックを呼ぶ。
// 呼び出し元は各コールバックで SSE イベントを送出するなどして、
// 8 秒を待たずに S (約 2 秒) から順次 UI に反映できる。
func (c *Client) GenerateSOAPStreaming(ctx context.Context, interviewText string, onSection SectionCallback) (*SOAPSuggestion, bool, time.Duration, error) {
	start := time.Now()

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		result := generateMockSOAP(interviewText)
		if onSection != nil {
			onSection("S", result.Subjective)
			onSection("O", result.Objective)
			onSection("A", result.Assessment)
			onSection("P", result.Plan)
		}
		return result, true, time.Since(start), nil
	}

	const systemPrompt = "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。問診情報を元に、カルテの続きを提案します。"

	userS := interviewText + "\n\n上記の問診記録から、S（主観的情報）を記載してください。"
	sText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userS, "S: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return generateMockSOAP(interviewText), true, time.Since(start), nil
	}
	s := cleanSectionOutput(sText, "S")
	if onSection != nil {
		onSection("S", s)
	}

	userO := interviewText + "\n\n【記載済み】\nS: " + s + "\n\n上記に続けてO（客観的情報）を記載してください。"
	oText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userO, "O: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s}, false, time.Since(start), nil
	}
	o := cleanSectionOutput(oText, "O")
	if onSection != nil {
		onSection("O", o)
	}

	userA := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\n\n上記に続けてA（評価）を記載してください。"
	aText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userA, "A: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s, Objective: o}, false, time.Since(start), nil
	}
	a := cleanSectionOutput(aText, "A")
	if onSection != nil {
		onSection("A", a)
	}

	userP := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\nA: " + a + "\n\n上記に続けてP（計画）を記載してください。"
	pText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userP, "P: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s, Objective: o, Assessment: a}, false, time.Since(start), nil
	}
	p := cleanSectionOutput(pText, "P")
	if onSection != nil {
		onSection("P", p)
	}

	latency := time.Since(start)
	slog.Info("SOAP生成完了(streaming)", "latency_ms", latency.Milliseconds())
	return &SOAPSuggestion{Subjective: s, Objective: o, Assessment: a, Plan: p}, false, latency, nil
}

// GenerateSOAP はSOAP提案を4セクション順次生成する（suggest LoRA使用）
//
// soap_full LoRAは学習データ40件で品質不十分のため、512件で訓練されたsuggest LoRAを
// セクション毎に4回呼び出す方式を採用する。訓練データの形式:
//   Sのみ (80件): 問診 → "S（主観的情報）を記載してください" → S文
//   Oのみ (80件): 問診+S → "上記に続けてO（客観的情報）を記載してください" → O文
//   Aのみ (32件): 問診+S+O → "上記に続けてA（評価）を記載してください" → A文
//   Pのみ (64件): 問診+S+O+A → "上記に続けてP（計画）を記載してください" → P文
// この構造に完全一致したプロンプトで呼び出し、訓練分布内で高品質を得る。
func (c *Client) GenerateSOAP(ctx context.Context, interviewText string) (*SOAPSuggestion, bool, time.Duration, error) {
	start := time.Now()
	slog.Info("SOAP提案リクエスト", "text_preview", truncateText(interviewText, 50))

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		result := generateMockSOAP(interviewText)
		return result, true, time.Since(start), nil
	}

	const systemPrompt = "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。問診情報を元に、カルテの続きを提案します。"

	// S生成
	userS := interviewText + "\n\n上記の問診記録から、S（主観的情報）を記載してください。"
	sText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userS, "S: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		slog.Warn("S生成失敗", "error", err)
		return generateMockSOAP(interviewText), true, time.Since(start), nil
	}
	s := cleanSectionOutput(sText, "S")

	// O生成（S済み）
	userO := interviewText + "\n\n【記載済み】\nS: " + s + "\n\n上記に続けてO（客観的情報）を記載してください。"
	oText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userO, "O: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		slog.Warn("O生成失敗", "error", err)
		return &SOAPSuggestion{Subjective: s}, false, time.Since(start), nil
	}
	o := cleanSectionOutput(oText, "O")

	// A生成（S,O済み）
	userA := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\n\n上記に続けてA（評価）を記載してください。"
	aText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userA, "A: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		slog.Warn("A生成失敗", "error", err)
		return &SOAPSuggestion{Subjective: s, Objective: o}, false, time.Since(start), nil
	}
	a := cleanSectionOutput(aText, "A")

	// P生成（S,O,A済み）
	userP := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\nA: " + a + "\n\n上記に続けてP（計画）を記載してください。"
	pText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userP, "P: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		slog.Warn("P生成失敗", "error", err)
		return &SOAPSuggestion{Subjective: s, Objective: o, Assessment: a}, false, time.Since(start), nil
	}
	p := cleanSectionOutput(pText, "P")

	latency := time.Since(start)
	slog.Info("SOAP提案完了(SLM, 4セクション逐次)",
		"latency_ms", latency.Milliseconds(),
		"s_len", len([]rune(s)), "o_len", len([]rune(o)), "a_len", len([]rune(a)), "p_len", len([]rune(p)))

	return &SOAPSuggestion{Subjective: s, Objective: o, Assessment: a, Plan: p}, false, latency, nil
}

// cleanSectionOutput はSLM応答の冒頭にある "S:" / "O:" / "A:" / "P:" プレフィックスや
// マークダウン装飾を除去して本文のみを返す
func cleanSectionOutput(text, section string) string {
	t := strings.TrimSpace(text)
	for _, prefix := range []string{
		section + ":", section + "：", section + ".",
		"**" + section + ":**", "**" + section + "：**",
	} {
		if strings.HasPrefix(t, prefix) {
			t = strings.TrimSpace(strings.TrimPrefix(t, prefix))
			break
		}
	}
	return t
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
		systemPrompt = "あなたは医療記録の作成を支援するAIアシスタントです。入力された問診内容から、家族歴の情報を抽出してください。「続柄：疾患名」の形式で1行ずつ記載してください。"
	case "social_history":
		systemPrompt = "あなたは医療記録の作成を支援するAIアシスタントです。入力された問診内容から、社会歴の情報（喫煙、飲酒、職業、運動など）を抽出してください。「カテゴリ：詳細」の形式で1行ずつ記載してください。"
	default:
		return nil, false, 0, fmt.Errorf("不明なカテゴリ: %s", category)
	}

	// 家族歴/社会歴の要約はsuggest LoRAで応答（専用LoRA未学習のため）
	result, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, interviewText, "", 512, 0.3, LoRASuggestID)
	if err != nil {
		slog.Warn("SLM API呼び出し失敗。モックにフォールバック", "error", err)

		mockResult := generateMockSummary(interviewText, category)
		latency := time.Since(start)
		slog.Info("サマリー提案完了(フォールバックモック)", "category", category, "latency_ms", latency.Milliseconds())
		return mockResult, true, latency, nil
	}

	// 自然文からサマリーをパース
	suggestion := parseSummaryFromText(result, category)

	latency := time.Since(start)
	slog.Info("サマリー提案完了(SLM)", "category", category, "latency_ms", latency.Milliseconds())
	return suggestion, false, latency, nil
}

// AdmissionSummary は入院時サマリ応答
type AdmissionSummary struct {
	Text string `json:"text"`
}

// GenerateAdmissionSummary は詳細問診情報から入院時サマリを生成する
// 返り値: サマリ、isMock、レイテンシ、エラー
func (c *Client) GenerateAdmissionSummary(ctx context.Context, interviewText string) (*AdmissionSummary, bool, time.Duration, error) {
	start := time.Now()
	truncated := truncateText(interviewText, 50)
	slog.Info("入院時サマリ生成リクエスト", "text_preview", truncated)

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		latency := time.Since(start)
		return &AdmissionSummary{
			Text: "【入院時サマリ】\n（モック: SLM未接続）\n" + truncated,
		}, true, latency, nil
	}

	// admission_summary LoRAの訓練時と一致するシステムプロンプト
	systemPrompt := "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。詳細な問診・検査情報から入院時サマリを作成します。"

	result, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, interviewText, "", 1536, 0.3, LoRAAdmissionID)
	if err != nil {
		slog.Warn("入院時サマリ生成失敗", "error", err)
		latency := time.Since(start)
		return &AdmissionSummary{
			Text: "【入院時サマリ】\n（生成失敗: " + err.Error() + "）",
		}, true, latency, nil
	}

	latency := time.Since(start)
	slog.Info("入院時サマリ生成完了(SLM)", "latency_ms", latency.Milliseconds())
	return &AdmissionSummary{Text: strings.TrimSpace(result)}, false, latency, nil
}

// callChatCompletionWithParams はOpenAI互換APIを呼び出す（後方互換、suggestアダプターを使用）
func (c *Client) callChatCompletionWithParams(ctx context.Context, systemPrompt, userPrompt string, maxTokens int, temperature float64) (string, error) {
	return c.callChatCompletionWithLoRA(ctx, systemPrompt, userPrompt, "", maxTokens, temperature, LoRASuggestID)
}

// stripThinkPrefix は Qwen3 chat template が assistant prefill 時に挿入する
// `<think>\n\n</think>\n\n` を除去する。enable_thinking=false 指定でも混入するため後処理で対応。
func stripThinkPrefix(s string) string {
	t := strings.TrimLeft(s, " \n\t")
	for _, p := range []string{
		"<think>\n\n</think>\n\n",
		"<think>\n</think>\n",
		"<think></think>",
		"<think>\n\n</think>",
	} {
		if strings.HasPrefix(t, p) {
			return strings.TrimLeft(t[len(p):], " \n\t")
		}
	}
	// 任意の<think>...</think>ブロックにも対応（保険）
	if strings.HasPrefix(t, "<think>") {
		if end := strings.Index(t, "</think>"); end >= 0 {
			return strings.TrimLeft(t[end+len("</think>"):], " \n\t")
		}
	}
	return s
}

// setActiveLoRA はllama-serverのglobal LoRA scale設定を書き換える
// 指定IDのLoRAをscale=1.0、他をscale=0.0にする。loraMu保有前提で呼ぶこと。
func (c *Client) setActiveLoRA(ctx context.Context, loraID int) error {
	if c.activeLoRAID == loraID {
		return nil // キャッシュヒット: 切替不要
	}
	scales := []LoRAAdapterConfig{
		{ID: LoRASuggestID, Scale: 0.0},
		{ID: LoRASOAPFullID, Scale: 0.0},
		{ID: LoRAAdmissionID, Scale: 0.0},
	}
	for i := range scales {
		if scales[i].ID == loraID {
			scales[i].Scale = 1.0
		}
	}
	body, _ := json.Marshal(scales)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/lora-adapters", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("LoRA切替失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LoRA切替ステータス %d", resp.StatusCode)
	}
	c.activeLoRAID = loraID
	slog.Debug("LoRA切替完了", "lora_id", loraID)
	return nil
}

// callChatCompletionWithLoRA はLoRAアダプターを指定してOpenAI互換APIを呼び出す
// loraID: LoRASuggestID / LoRASOAPFullID / LoRAAdmissionID のいずれか
// assistantPrefill が空でない場合、末尾に assistant メッセージを追加して「続き生成」モードに入る
// （llama.cpp の --prefill-assistant 機能を利用。Qwen3 の <think> ブロックは stripThinkPrefix で除去）
func (c *Client) callChatCompletionWithLoRA(ctx context.Context, systemPrompt, userPrompt, assistantPrefill string, maxTokens int, temperature float64, loraID int) (string, error) {
	c.loraMu.Lock()
	defer c.loraMu.Unlock()

	if err := c.setActiveLoRA(ctx, loraID); err != nil {
		return "", err
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	if assistantPrefill != "" {
		messages = append(messages, Message{Role: "assistant", Content: assistantPrefill})
	}

	reqBody := ChatCompletionRequest{
		Model:       modelName,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
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

	return stripThinkPrefix(chatResp.Choices[0].Message.Content), nil
}

// truncateText はテキストを指定文字数で切り詰める
func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}
