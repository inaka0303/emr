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
	"time"
)

// RAGClient は RAG 検索サーバー (port 8082) に対する軽量クライアント。
// SLM 生成時にガイドライン snippets を自動注入するために使う。
// 2026-04-24 追加 (M3 field notes B アーキ改善 ⭐⭐):
//
//	4B の薬剤ハルシネーション対策として SOAP P 生成時に RAG 結果を system prompt に混ぜる。
type RAGClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRAGClient は指定 URL の RAG サーバーへのクライアントを返す。
// baseURL が空なら nil を返す（= 機能無効化）。
func NewRAGClient(baseURL string) *RAGClient {
	if baseURL == "" {
		return nil
	}
	return &RAGClient{
		baseURL: baseURL,
		// SOAP 生成中のブロッキング呼出を想定、短めのタイムアウト。
		// RAG が遅い/死んでたら諦めて RAG 無しで続行する。
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type ragRequest struct {
	Query string `json:"query"`
	N     int    `json:"n,omitempty"`
}

type ragResult struct {
	ParentID  string  `json:"parent_id"`
	Text      string  `json:"text"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
	ChildHits int     `json:"child_hits"`
}

type ragResponse struct {
	Query     string      `json:"query"`
	Results   []ragResult `json:"results"`
	ElapsedMs int         `json:"elapsed_ms"`
}

// Search は query で RAG 検索し、上位 n 件の parent chunks を返す。
// 失敗時は nil, err を返す（呼び出し側で RAG 無し続行を判断）。
func (r *RAGClient) Search(ctx context.Context, query string, n int) ([]ragResult, error) {
	if r == nil {
		return nil, nil
	}
	if n <= 0 {
		n = 3
	}
	body, err := json.Marshal(ragRequest{Query: query, N: n})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rag http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rag status %d: %s", resp.StatusCode, string(b))
	}
	var out ragResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("rag decode: %w", err)
	}
	return out.Results, nil
}

// FormatRAGContext は RAG 検索結果を SLM system prompt に付加しやすい形に整形する。
// n 件をタイトル付き snippet として結合する。snippet 本体は長すぎると prompt 爆発するため
// 各 1000 chars で切る。
func FormatRAGContext(results []ragResult) string {
	if len(results) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n【参考ガイドライン】 (以下の記述は根拠のある情報源から抜粋、薬剤名・用量はこの範囲で記載すること):\n")
	for i, r := range results {
		if i >= 3 { // 最大 3 件
			break
		}
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = "(無題)"
		}
		text := strings.TrimSpace(r.Text)
		if len([]rune(text)) > 1000 {
			text = string([]rune(text)[:1000]) + "..."
		}
		fmt.Fprintf(&b, "\n[%d] %s\n%s\n", i+1, title, text)
	}
	return b.String()
}

// tryAugmentWithRAG は interview text を query として RAG を叩き、
// 見つかれば system prompt に追加する snippet を返す。
// 失敗時は "" を返す（エラーは debug ログのみ、SLM 呼出は通常通り続行）。
// ctx は短時間 (RAGClient httpClient 側で 5s cap) のため呼び出し側の ctx を使って良い。
func (c *Client) tryAugmentWithRAG(ctx context.Context, queryHint string) string {
	if c == nil || c.rag == nil {
		return ""
	}
	// クエリは長すぎると embedding が遅くなるので先頭 400 文字に絞る
	q := strings.TrimSpace(queryHint)
	if len([]rune(q)) > 400 {
		q = string([]rune(q)[:400])
	}
	if q == "" {
		return ""
	}
	start := time.Now()
	results, err := c.rag.Search(ctx, q, 3)
	if err != nil {
		slog.Debug("RAG 自動注入 skip: RAG query 失敗", "error", err)
		return ""
	}
	if len(results) == 0 {
		slog.Debug("RAG 自動注入 skip: 結果なし", "query_len", len([]rune(q)))
		return ""
	}
	snippet := FormatRAGContext(results)
	slog.Info("RAG 自動注入",
		"n_results", len(results),
		"snippet_len", len([]rune(snippet)),
		"elapsed_ms", time.Since(start).Milliseconds())
	return snippet
}
