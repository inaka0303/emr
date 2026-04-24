package slm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 出力の後処理で使う正規表現（init時にコンパイル）
var (
	// 補足説明ブロック: 末尾の「---」以降の解説を丸ごと削除
	reSupplementBlock = regexp.MustCompile(`(?ms)\n+-{3,}\s*\n+\*?\*?(補足|解説|ポイント|【補足|【解説)[\s\S]*$`)
	// 末尾の「■ 補足・注意点」「■ 補足説明」「■ ポイント」等（---無し版）の見出しから末尾までを削除
	// 例: "P: ...\n\n■ 補足・注意点\n・抗凝固薬の選択: ..." → "P: ..."
	reTailSupplementHeading = regexp.MustCompile(`(?ms)\n+\s*■\s*\*{0,2}(?:補足(?:説明|事項|・注意点|・注意|・ポイント)?|注意点|注意事項|解説|ポイント|提案ポイント|留意点|参考情報|臨床上の補足|医師への[^\n]{0,20}|医療者向け[^\n]{0,20})\*{0,2}[\s\S]*$`)
	// 末尾の「補足: …」「補足説明: …」等（見出し記号無しでコロン付き）
	reTailSupplementPlain = regexp.MustCompile(`(?ms)\n+\s*\*{0,2}(?:補足説明|補足事項|補足|注意事項|注意点|留意点|参考情報)\*{0,2}\s*[:：][\s\S]*$`)
	// 先頭の 【解説】/【注釈】 等の開始を検知する prefix 集合
	// stripPrefaceExplanation で使用（lookahead が必要なため regex 単独では実装できず関数に切り出し）
	prefaceHeaders = []string{"【解説】", "【注釈】", "【説明】", "【前置き】", "【前置】", "【解釈】"}
	// 本文セクションマーカーの行頭パターン（Go RE2 は lookahead 非対応のため個別に検索）
	reSectionMarkerAtLine = regexp.MustCompile(`(?m)^\s*(?:\*{0,2}[SOAP]\*{0,2}[\s:：\)\.]|■\s*[SOAP]|#{1,6}\s+[SOAP]|【[SOAP])`)
	// 丸括弧注釈 ※系: 「（※〜）」を削除（1行）
	reParenNote = regexp.MustCompile(`（※[^）\n]{0,300}）`)
	// 丸括弧メタ説明ブロック（複数行対応）: 池田里奈型
	//   （問診記録の冒頭にある「自覚症状なし」をそのまま記載するのが適切です。
	//    電子カルテでは、問診で確認した主観的情報として簡潔に記録します。）
	// キーワード「記載するのが適切／電子カルテでは／カルテには／記録します／参考まで／推奨されます／を意味します／注意が必要」を
	// 含む (...) ブロックを丸ごと削除。※無し版（reParenNote でカバーされない）を想定。
	reMetaParenBlock = regexp.MustCompile(`（[^）]{0,600}?(?:記載するのが適切|電子カルテでは|カルテには|記録します|参考まで|推奨されます|を意味します|注意が必要|適切に記載|を表します|意味しています)[^）]{0,600}?）`)
	// 太字 **xxx** → xxx
	reBold = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	// Markdown ヘッダー `#{1,6} xxx` → ■ xxx (SOAP/admission 共通の推奨形式)
	reMdHeader = regexp.MustCompile(`(?m)^#{1,6}\s+(\d+\.\s*)?(.+?)\s*$`)
	// 行頭アスタリスク箇条書き `* ` / `*   ` → `・`
	reStarBullet = regexp.MustCompile(`(?m)^\s*\*\s+`)
	// 行頭ハイフン箇条書き `- xxx` → `・`（`---` は reHr が先に消すのでここは `- ` のみで安全）
	reDashBullet = regexp.MustCompile(`(?m)^-\s+`)
	// 単独の `---` 区切り線（行全体）を削除
	reHr = regexp.MustCompile(`(?m)^\s*-{3,}\s*$`)
	// 連続空行を2行に圧縮
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
	// 残留マークダウン安全網: 不整合な連続アスタリスク **/***/**** を除去
	// （reBold で捕捉されなかった ** の単独出現に対応）
	reResidualAsterisks = regexp.MustCompile(`\*{2,}`)
	// インライン italic `*text*` → `text` (bullet と区別するため前後が非空白かつ非 * であること)
	reInlineItalic = regexp.MustCompile(`(^|[^*\s])\*([^*\n\s][^*\n]{0,200}?[^*\n\s]|[^*\n\s])\*($|[^*])`)
	// インラインコード `` `text` `` → `text` (改行を跨がない)
	reInlineCode = regexp.MustCompile("`([^`\n]+?)`")
	// 取り消し線 ~~text~~ → text
	reStrikethrough = regexp.MustCompile(`~~([^~\n]+?)~~`)
	// 代替 bold __text__ → text / italic _text_ → text (全角文字境界には緩めに適用)
	reAltBold   = regexp.MustCompile(`__([^_\n]+?)__`)
	reAltItalic = regexp.MustCompile(`(^|[^_\w])_([^_\n\s][^_\n]{0,200}?[^_\n\s]|[^_\n\s])_($|[^_\w])`)
	// 残留単独 `*` を各行末など明らかに装飾的な位置で除去（multiline: 各行の末尾で発動）
	reTrailingAsterisk = regexp.MustCompile(`(?m)\s+\*+\s*$`)
	// 前置き文: 出力冒頭の「ご提示〜」「以下〜」型の挨拶を削除
	rePreamble = regexp.MustCompile(`^(?:ご提示いただいた[^\n]{0,200}|以下[^\n]{0,200}(?:提案|作成|示し|記載)[^\n]{0,80}|上記[^\n]{0,80}に基づ[^\n]{0,80})[。\.\n]*\n+`)
	// 末尾「作成者: AI...」フッター
	reFooter = regexp.MustCompile(`(?ms)\n+\*?\*?作成者[：:][\s\S]*$`)
	// AI 自己言及・責任放棄型メタコメント（行単位で削除）
	reMetaAIComment = regexp.MustCompile(`(?m)^.*(?:AIとして|AI として|医療専門家の意見|医師の診察|専門医.*ご相談|ご自身の判断|これは医療行為に代わる|一般的な参考情報|注意事項|医療的判断|最終的(?:な|には)医師).*$\n?`)
	// 「参考情報として」系
	reReferenceInfo = regexp.MustCompile(`(?m)^.*参考情報として(?:伝達|記載|併記|添付).*$\n?`)
	// 「〜したものです」「〜に過ぎません」式の解説文
	reExplanatoryEnd = regexp.MustCompile(`(?m)^.*(?:記述したものです|記載したものです|説明したものです|過ぎません|ご参考まで)[。.]?\s*$\n?`)
	// Markdown ブロッククォート `> `
	reBlockquote = regexp.MustCompile(`(?m)^>\s+`)
	// コードブロック ```
	reCodeBlock = regexp.MustCompile("(?s)```[\\s\\S]*?```")
)

// cleanModelOutput は LoRA 出力から余計なマークダウン・メタ説明を除去する。
// eval で残留が確認された次のノイズを一掃する:
//   - 末尾の「補足説明（AIの提案ポイント）」ブロック
//   - （※〜）型の但し書き
//   - **太字** → 裸text
//   - ### 見出し → ■ 見出し
//   - 行頭 `* ` / `- ` → `・`
//   - 単独の `---`
//   - 前置き文（「ご提示いただいた〜」「以下の通り〜」）
//   - 末尾の「作成者: AI」フッター
// stripPrefaceExplanation は先頭の 【解説】 等の前置きブロックを、
// 最初の SOAP セクションマーカー直前までまとめて削除する。
// lookahead が RE2 で使えないため関数で実装。
// 前置きが無い / SOAP マーカーが無い場合は何もしない。
func stripPrefaceExplanation(s string) string {
	trimmed := strings.TrimLeft(s, " \t\n")
	leadingWs := len(s) - len(trimmed)
	found := false
	for _, h := range prefaceHeaders {
		if strings.HasPrefix(trimmed, h) || strings.HasPrefix(trimmed, "**"+h+"**") {
			found = true
			break
		}
	}
	if !found {
		return s
	}
	loc := reSectionMarkerAtLine.FindStringIndex(s[leadingWs:])
	if loc == nil {
		return s // セクションマーカーが見つからない: 削除すると本文が消えるので保持
	}
	return s[leadingWs+loc[0]:]
}

func cleanModelOutput(s string) string {
	t := s
	// 0. 先頭 【解説】 前置きブロックを最初のセクションマーカー直前まで削除
	t = stripPrefaceExplanation(t)
	// 1. 末尾のメタ説明ブロック・フッター（--- 有無両方に対応）
	t = reSupplementBlock.ReplaceAllString(t, "")
	t = reTailSupplementHeading.ReplaceAllString(t, "")
	t = reTailSupplementPlain.ReplaceAllString(t, "")
	t = reFooter.ReplaceAllString(t, "")
	// 2. コードブロックを丸ごと除去（Python/JSON のleak対策）
	t = reCodeBlock.ReplaceAllString(t, "")
	// 3. AI責任放棄コメント / 参考情報系 / 解説文末尾
	t = reMetaAIComment.ReplaceAllString(t, "")
	t = reReferenceInfo.ReplaceAllString(t, "")
	t = reExplanatoryEnd.ReplaceAllString(t, "")
	// 4. 前置き文
	t = rePreamble.ReplaceAllString(t, "")
	// 5. （※〜）注釈 / 丸括弧メタ説明ブロック（複数行）
	t = reParenNote.ReplaceAllString(t, "")
	t = reMetaParenBlock.ReplaceAllString(t, "")
	// 6. Markdown変換
	t = reMdHeader.ReplaceAllStringFunc(t, func(m string) string {
		sub := reMdHeader.FindStringSubmatch(m)
		if len(sub) >= 3 {
			return "■ " + sub[2]
		}
		return m
	})
	t = reBold.ReplaceAllString(t, "$1")
	t = reAltBold.ReplaceAllString(t, "$1")
	t = reStrikethrough.ReplaceAllString(t, "$1")
	t = reInlineCode.ReplaceAllString(t, "$1")
	// italic は bullet と紛らわしいので bullet 変換後に実行
	t = reStarBullet.ReplaceAllString(t, "・")
	t = reDashBullet.ReplaceAllString(t, "・")
	t = reInlineItalic.ReplaceAllString(t, "$1$2$3")
	t = reAltItalic.ReplaceAllString(t, "$1$2$3")
	t = reHr.ReplaceAllString(t, "")
	t = reBlockquote.ReplaceAllString(t, "")
	// 7. 残留アスタリスク安全網 (**/***, 行末 * 等)
	t = reResidualAsterisks.ReplaceAllString(t, "")
	t = reTrailingAsterisk.ReplaceAllString(t, "")
	// 8. 空行圧縮・端trim
	t = reMultiNewline.ReplaceAllString(t, "\n\n")
	return strings.TrimSpace(t)
}

const modelName = "qwen3.5-4b-medical"

// LoRA ID constants (4B llama-server --lora の順序に対応、3-LoRA本番構成)
//   id 0: sft_4b_nocpt_A              → suggest（prefill 補完・SOAPストリーミング） ★
//   id 1: sft_soap_full_v2_r64_lora   → SOAP全体 (Phase5 eval で 4B best) ★
//   id 2: sft_admission_v2_r32_lora   → admission fallback (9B未接続時のみ使用)
//
// 9B admission サーバー (port 8083, optional) の LoRA 構成:
//   id 0: sft_admission_9b_v3_clean_r32 ★ admission 本番 (Phase5 eval で md=87 最良)
const (
	LoRASuggestID     = 0 // 4B: インライン補完・SOAPストリーミング4-call
	LoRASOAPFullID    = 1 // 4B: SOAP全体 (v2_r64, Phase5 eval best)
	LoRAAdmissionID   = 2 // 4B: admission fallback (9B server 未起動時のみ)
	LoRAAdmissionOn9B = 0 // 9B 専用サーバー上の唯一の LoRA
	DefaultNLoras4B   = 3
	DefaultNLoras9B   = 1
)

// AdmissionServerLabel は GenerateAdmissionSummary 結果がどのサーバーで生成されたかを示す。
// UI で "4B-fallback" の場合は品質低下の警告を表示する。
const (
	AdmissionServer9B         = "9B"
	AdmissionServer4BFallback = "4B-fallback"
)

// Client はSLM推論サーバーとの通信を担当する
//
// LoRA切替方式: llama-serverはper-request `lora`を受け付けるが実効しないため、
// POST /lora-adapters でグローバルscaleを書き換える方式を採る。
// 全SLM呼び出しは loraMu でserialize（同時実行中のLoRA差し替え事故を防ぐ）。
// activeLoRAID はキャッシュ用で、前回と同じなら再設定しない。
//
// admClient は admission タスク用の独立サーバー（典型: 9B）がある場合のみ設定される。
// 未設定時は自身(4B server)の LoRAAdmissionID を fallback として使う。
type Client struct {
	baseURL      string
	httpClient   *http.Client
	useMock      bool
	mu           sync.RWMutex
	loraMu       sync.Mutex
	activeLoRAID int
	nLoras       int // /lora-adapters でリセットすべき LoRA 総数
	admClient    *Client
}

// LoRAAdapterConfig はglobal scale切替のリクエストboy要素
type LoRAAdapterConfig struct {
	ID    int     `json:"id"`
	Scale float64 `json:"scale"`
}

// ChatCompletionRequest はOpenAI互換リクエスト
// RepeatPenalty: llama.cpp の repeat_penalty (1.0 = 無効)。
// degeneration（同一フレーズ反復ループ）対策で 1.1 程度を推奨。
// 訓練分布から外れた短入力で A/P が無限ループする問題（M3 field notes 2026-04-23）の対処。
type ChatCompletionRequest struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	Temperature   float64   `json:"temperature"`
	MaxTokens     int       `json:"max_tokens"`
	RepeatPenalty float64   `json:"repeat_penalty,omitempty"`
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
		nLoras:       DefaultNLoras4B,
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

// SetAdmissionServer は admission タスク専用の別推論サーバーを登録する（典型: 9B, port 8083）。
// 呼び出し後、GenerateAdmissionSummary はこの別サーバーにルーティングされる。
// baseURL が空文字 or 既存の baseURL と同じなら何もしない。
func (c *Client) SetAdmissionServer(baseURL string) {
	if baseURL == "" || baseURL == c.baseURL {
		return
	}
	adm := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 9B admission は 4B より時間かかる
		},
		useMock:      true,
		activeLoRAID: -1,
		nLoras:       DefaultNLoras9B,
	}
	if adm.checkHealth() {
		adm.useMock = false
		slog.Info("admission 専用 SLM サーバー接続成功", "url", baseURL, "nLoras", DefaultNLoras9B)
	} else {
		slog.Warn("admission 専用 SLM サーバー起動中 or 未接続、4B fallback を使用（後で復帰検知）", "url", baseURL)
		// 登録は行う: backgroundHealthCheck が後から疎通確認して promote する
	}
	go adm.backgroundHealthCheck()
	c.admClient = adm
}

// AdmissionClientHealthy は admission サーバーが設定済みかつ疎通可能かを返す。
// nil Client や未設定なら false。
func (c *Client) AdmissionClientHealthy() bool {
	if c == nil || c.admClient == nil {
		return false
	}
	return c.admClient.IsHealthy()
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

// GenerateSOAP はSOAP提案を soap_full LoRA (v2_r64, Phase5 eval 4B best) で単一呼出生成する。
//
// 従来は suggest LoRA を 4 回呼ぶ方式だったが、185件で訓練された v2_r64 が
// 単一呼出で 4-call と同等以上の品質 (md=68, len=1302) を達成したため切替。
// SSE 用の GenerateSOAPStreaming は段階的 UI のため 4-call suggest を維持。
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

	systemPrompt := "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。問診情報からSOAP形式のカルテ記載を提案します。"
	userPrompt := interviewText + "\n\n上記の情報から、SOAP形式のカルテ記載を提案してください。"

	result, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userPrompt, "", 1536, 0.3, LoRASOAPFullID)
	if err != nil {
		slog.Warn("SOAP単一呼出失敗、4-call suggest にフォールバック", "error", err)
		return c.generateSOAPFallback(ctx, interviewText, start)
	}

	suggestion := parseSOAPFromText(result)
	// parser が全セクション埋められなかった場合のみ fallback（片側だけ欠けたら fallback）
	if suggestion.Subjective == "" || suggestion.Objective == "" ||
		suggestion.Assessment == "" || suggestion.Plan == "" {
		// 2026-04-24: raw output を dump して parse 失敗の原因調査を可能にする
		rawPreview := result
		if len([]rune(rawPreview)) > 400 {
			rawPreview = string([]rune(rawPreview)[:400]) + "...(truncated)"
		}
		slog.Warn("SOAPパース不完全、4-call suggest にフォールバック",
			"has_s", suggestion.Subjective != "", "has_o", suggestion.Objective != "",
			"has_a", suggestion.Assessment != "", "has_p", suggestion.Plan != "",
			"raw_preview", rawPreview,
			"raw_len", len([]rune(result)))
		return c.generateSOAPFallback(ctx, interviewText, start)
	}

	latency := time.Since(start)
	slog.Info("SOAP提案完了(SLM, soap_full v2_r64 単一呼出)", "latency_ms", latency.Milliseconds())
	return suggestion, false, latency, nil
}

// generateSOAPFallback は 4-call suggest 方式での従来生成（単一呼出失敗時の保険）
func (c *Client) generateSOAPFallback(ctx context.Context, interviewText string, start time.Time) (*SOAPSuggestion, bool, time.Duration, error) {
	const systemPrompt = "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。問診情報を元に、カルテの続きを提案します。"

	userS := interviewText + "\n\n上記の問診記録から、S（主観的情報）を記載してください。"
	sText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userS, "S: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return generateMockSOAP(interviewText), true, time.Since(start), nil
	}
	s := cleanSectionOutput(sText, "S")

	userO := interviewText + "\n\n【記載済み】\nS: " + s + "\n\n上記に続けてO（客観的情報）を記載してください。"
	oText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userO, "O: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s}, false, time.Since(start), nil
	}
	o := cleanSectionOutput(oText, "O")

	userA := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\n\n上記に続けてA（評価）を記載してください。"
	aText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userA, "A: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s, Objective: o}, false, time.Since(start), nil
	}
	a := cleanSectionOutput(aText, "A")

	userP := interviewText + "\n\n【記載済み】\nS: " + s + "\nO: " + o + "\nA: " + a + "\n\n上記に続けてP（計画）を記載してください。"
	pText, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userP, "P: ", 384, 0.3, LoRASuggestID)
	if err != nil {
		return &SOAPSuggestion{Subjective: s, Objective: o, Assessment: a}, false, time.Since(start), nil
	}
	p := cleanSectionOutput(pText, "P")

	latency := time.Since(start)
	slog.Info("SOAP提案完了(fallback 4-call suggest)",
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
// Server: "9B" | "4B-fallback" | "" (mock時は空)
// 9B 未接続時に 4B admission LoRA に自動フォールバックした場合は "4B-fallback" が入り、
// frontend がこれを検出して「品質低下モード」警告を表示する。
type AdmissionSummary struct {
	Text   string `json:"text"`
	Server string `json:"server,omitempty"`
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

	// 専用 9B admission サーバーがあればそちらを試す
	// HTTP レベルで失敗した場合は即座に 4B fallback で retry（backgroundHealthCheck を待たない）
	if c.admClient != nil && c.admClient.IsHealthy() {
		result, err := c.admClient.callChatCompletionWithLoRA(ctx, systemPrompt, interviewText, "", 1536, 0.3, LoRAAdmissionOn9B)
		if err == nil {
			latency := time.Since(start)
			slog.Info("入院時サマリ生成完了(SLM)", "server", AdmissionServer9B, "latency_ms", latency.Milliseconds())
			return &AdmissionSummary{Text: strings.TrimSpace(result), Server: AdmissionServer9B}, false, latency, nil
		}
		slog.Warn("9B admission 呼出失敗、4B fallback で再試行", "error", err)
		// 即座に 9B を unhealthy マーク（次回 backgroundHealthCheck まで待たず fallback）
		c.admClient.mu.Lock()
		c.admClient.useMock = true
		c.admClient.mu.Unlock()
	}

	// 4B fallback（9B未接続 or 呼出失敗時）
	result, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, interviewText, "", 1536, 0.3, LoRAAdmissionID)
	if err != nil {
		slog.Warn("入院時サマリ生成失敗（4B fallback も失敗）", "error", err)
		latency := time.Since(start)
		return &AdmissionSummary{
			Text: "【入院時サマリ】\n（生成失敗: " + err.Error() + "）",
		}, true, latency, nil
	}

	latency := time.Since(start)
	slog.Warn("入院時サマリ 4B fallback で生成（品質低下）", "server", AdmissionServer4BFallback, "latency_ms", latency.Milliseconds())
	return &AdmissionSummary{Text: strings.TrimSpace(result), Server: AdmissionServer4BFallback}, false, latency, nil
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
// nLoras は Client 起動時に確定したサーバーのロード済み LoRA 総数を使う
// （ハードコードすると新 LoRA 追加時に旧 LoRA が残留する bug になる）。
func (c *Client) setActiveLoRA(ctx context.Context, loraID int) error {
	if c.activeLoRAID == loraID {
		return nil // キャッシュヒット: 切替不要
	}
	n := c.nLoras
	if n <= 0 {
		n = DefaultNLoras4B
	}
	scales := make([]LoRAAdapterConfig, n)
	for i := 0; i < n; i++ {
		scales[i] = LoRAAdapterConfig{ID: i, Scale: 0.0}
	}
	if loraID >= 0 && loraID < n {
		scales[loraID].Scale = 1.0
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
		Model:         modelName,
		Messages:      messages,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		RepeatPenalty: 1.1, // degeneration 対策（1.0=無効、1.1 が穏当）
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

	return cleanModelOutput(stripThinkPrefix(chatResp.Choices[0].Message.Content)), nil
}

// truncateText はテキストを指定文字数で切り詰める
func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}
