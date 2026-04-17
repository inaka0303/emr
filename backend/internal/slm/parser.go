package slm

import (
	"strings"
)

// parseSOAPFromText はSLMの自然文出力からSOAPの各セクションを抽出する
// 入力例:
//   S: 1週間前から息切れ増悪。起座呼吸あり。
//   O: BP 148/88, HR 92...
//   A: #1 慢性心不全急性増悪
//   P: フロセミド増量...
func parseSOAPFromText(text string) *SOAPSuggestion {
	suggestion := &SOAPSuggestion{}

	// <think>...</think> を除去
	if idx := strings.Index(text, "</think>"); idx >= 0 {
		text = strings.TrimSpace(text[idx+len("</think>"):])
	}

	// S: / O: / A: / P: の区切りでパース
	// 各種表記に対応: "S:", "S：", "S)", "**S**", "【S】" etc.
	sections := map[string]*string{
		"S": &suggestion.Subjective,
		"O": &suggestion.Objective,
		"A": &suggestion.Assessment,
		"P": &suggestion.Plan,
	}

	// セクションマーカーの検出パターン（長い順で検索）
	markers := []string{
		// 太字マークダウン + 日本語ラベル
		"**S（主観的情報）**", "**O（客観的情報）**", "**A（評価）**", "**P（計画）**",
		"**S（主観）**", "**O（客観）**", "**A（アセスメント）**", "**P（プラン）**",
		// 太字マークダウン
		"**S**:", "**O**:", "**A**:", "**P**:",
		"**S**", "**O**", "**A**", "**P**",
		// 【】括弧
		"【S】", "【O】", "【A】", "【P】",
		"【S：", "【O：", "【A：", "【P：",
		// 括弧 + ラベル
		"S（主観的情報）:", "O（客観的情報）:", "A（評価）:", "P（計画）:",
		"S（主観的情報）", "O（客観的情報）", "A（評価）", "P（計画）",
		// 基本パターン
		"S:", "S：", "S)", "S（", "S.",
		"O:", "O：", "O)", "O（", "O.",
		"A:", "A：", "A)", "A（", "A.",
		"P:", "P：", "P)", "P（", "P.",
	}

	// テキストを行ごとに処理
	lines := strings.Split(text, "\n")
	currentSection := ""
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentSection != "" {
				currentLines = append(currentLines, "")
			}
			continue
		}

		// セクションマーカーを検出（長いマーカーから先にチェック）
		foundSection := ""
		contentAfterMarker := ""
		for _, marker := range markers {
			if strings.HasPrefix(trimmed, marker) {
				// マーカーからセクション名を抽出（最初のS/O/A/Pを探す）
				for _, ch := range marker {
					if ch == 'S' || ch == 'O' || ch == 'A' || ch == 'P' {
						foundSection = string(ch)
						break
					}
				}
				contentAfterMarker = strings.TrimSpace(trimmed[len(marker):])
				// 後続の区切り文字を除去
				for _, sep := range []string{":", "：", "**", "】"} {
					contentAfterMarker = strings.TrimPrefix(contentAfterMarker, sep)
				}
				contentAfterMarker = strings.TrimSpace(contentAfterMarker)
				break
			}
		}

		if foundSection != "" {
			// 前のセクションを保存
			if currentSection != "" {
				if ptr, ok := sections[currentSection]; ok {
					*ptr = strings.TrimSpace(strings.Join(currentLines, "\n"))
				}
			}
			// 新しいセクション開始
			currentSection = foundSection
			currentLines = nil
			if contentAfterMarker != "" {
				currentLines = append(currentLines, contentAfterMarker)
			}
		} else if currentSection != "" {
			currentLines = append(currentLines, trimmed)
		} else {
			// セクションマーカーがまだ出ていない場合、Sとして扱う
			currentSection = "S"
			currentLines = append(currentLines, trimmed)
		}
	}

	// 最後のセクションを保存
	if currentSection != "" {
		if ptr, ok := sections[currentSection]; ok {
			*ptr = strings.TrimSpace(strings.Join(currentLines, "\n"))
		}
	}

	return suggestion
}

// parseSummaryFromText はSLMの自然文出力から家族歴/社会歴の情報を抽出する
// 入力例（family_history）:
//   父：高血圧症（60歳で発症）
//   母：2型糖尿病
//   祖父：胃がん（70歳で死亡）
func parseSummaryFromText(text string, category string) *SummarySuggestion {
	suggestion := &SummarySuggestion{}

	// <think>...</think> を除去
	if idx := strings.Index(text, "</think>"); idx >= 0 {
		text = strings.TrimSpace(text[idx+len("</think>"):])
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// マークダウンの箇条書き記号を除去
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimPrefix(trimmed, "・")
		trimmed = strings.TrimSpace(trimmed)

		// 「：」または「:」で分割
		var field, value string
		if idx := strings.Index(trimmed, "："); idx >= 0 {
			field = strings.TrimSpace(trimmed[:idx])
			value = strings.TrimSpace(trimmed[idx+len("："):])
		} else if idx := strings.Index(trimmed, ":"); idx >= 0 {
			field = strings.TrimSpace(trimmed[:idx])
			value = strings.TrimSpace(trimmed[idx+1:])
		} else {
			continue // 分割できない行はスキップ
		}

		if field == "" || value == "" {
			continue
		}

		// カテゴリに応じたフィールド名を設定
		if category == "family_history" {
			suggestion.Suggestions = append(suggestion.Suggestions, SummarySuggestionItem{
				Field:      "relation",
				Value:      field,
				Confidence: 0.85,
			})
			suggestion.Suggestions = append(suggestion.Suggestions, SummarySuggestionItem{
				Field:      "condition",
				Value:      value,
				Confidence: 0.85,
			})
		} else {
			suggestion.Suggestions = append(suggestion.Suggestions, SummarySuggestionItem{
				Field:      "category",
				Value:      field,
				Confidence: 0.85,
			})
			suggestion.Suggestions = append(suggestion.Suggestions, SummarySuggestionItem{
				Field:      "description",
				Value:      value,
				Confidence: 0.85,
			})
		}
	}

	return suggestion
}
