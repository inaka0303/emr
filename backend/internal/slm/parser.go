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
	// 注: cleanModelOutput は既に `**` / `###` → `■` / `---` を除去・置換しているため、
	//     bold系マーカーも残留ケースとして残す + cleaned出力向けマーカーを足す
	// 2026-04-24 追加: `■ 【S - Subjective】` 型が実出力に現れるケースに対応
	markers := []string{
		// 2026-04-24 追加: 【】+ ダッシュ付きラベル
		"■ 【S - Subjective】", "■ 【O - Objective】", "■ 【A - Assessment】", "■ 【P - Plan】",
		"■ 【S-Subjective】", "■ 【O-Objective】", "■ 【A-Assessment】", "■ 【P-Plan】",
		"【S - Subjective】", "【O - Objective】", "【A - Assessment】", "【P - Plan】",
		"【S-Subjective】", "【O-Objective】", "【A-Assessment】", "【P-Plan】",
		// 2026-04-24 追加: ■ + 【】だけの形式
		"■ 【S】", "■ 【O】", "■ 【A】", "■ 【P】",
		"■ 【S：", "■ 【O：", "■ 【A：", "■ 【P：",
		// cleaned出力: `■ S (Subjective)` 等の形式（post-process後）
		"■ S (Subjective)", "■ O (Objective)", "■ A (Assessment)", "■ P (Plan)",
		"■ S（主観的情報）", "■ O（客観的情報）", "■ A（評価）", "■ P（計画）",
		"■ S:", "■ O:", "■ A:", "■ P:",
		"■ S：", "■ O：", "■ A：", "■ P：",
		"■ S ", "■ O ", "■ A ", "■ P ", // 末尾スペース必須 → "■ SOAP" 誤マッチ防止
		// 英語フル表記
		"S (Subjective)", "O (Objective)", "A (Assessment)", "P (Plan)",
		// 太字マークダウン + 日本語ラベル (未cleanの保険)
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
		// 半角括弧 prefix（`S (xxx)` 型を細かく拾う）
		"S (", "O (", "A (", "P (",
		// 基本パターン
		"S:", "S：", "S)", "S（", "S.",
		"O:", "O：", "O)", "O（", "O.",
		"A:", "A：", "A)", "A（", "A.",
		"P:", "P：", "P)", "P（", "P.",
	}

	// 2026-04-24: SOAP セクションマーカーではない「見出し行」は skip する
	// LoRA が `■ SOAP 形式カルテ記載案` のような章見出しを出すケースがあり、
	// これらを section marker と誤認すると parse が崩れる。
	// 注: 単独行で「■ XXX」の XXX が SOAP の何かの章タイトル的な場合のみ skip。
	skipHeaderPrefixes := []string{
		"■ SOAP",
		"■ 【SOAP",
		"■ カルテ記載",
		"■ 電子カルテ",
		"■ 診療記録",
		"■ 医療記録",
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

		// 「■ SOAP ...」型の章見出しは skip（section marker と誤マッチ防止）
		isSkipHeader := false
		for _, h := range skipHeaderPrefixes {
			if strings.HasPrefix(trimmed, h) {
				isSkipHeader = true
				break
			}
		}
		if isSkipHeader {
			continue
		}

		// セクションマーカーを検出（長いマーカーから先にチェック）
		foundSection := ""
		contentAfterMarker := ""

		// 1. 「■ S」「S:」等のラインだけの section marker を先にチェック（trimmed 完全一致）
		//    これは marker loop の HasPrefix で誤マッチを起こしがちなパターンを先に catch する。
		for _, bareMarker := range []struct{ pattern, letter string }{
			{"■ S", "S"}, {"■ O", "O"}, {"■ A", "A"}, {"■ P", "P"},
			{"S:", "S"}, {"O:", "O"}, {"A:", "A"}, {"P:", "P"},
			{"S：", "S"}, {"O：", "O"}, {"A：", "A"}, {"P：", "P"},
			{"**S**", "S"}, {"**O**", "O"}, {"**A**", "A"}, {"**P**", "P"},
		} {
			if trimmed == bareMarker.pattern {
				foundSection = bareMarker.letter
				contentAfterMarker = ""
				break
			}
		}

		// 2. ラインだけ marker でなければ通常の HasPrefix 検索
		if foundSection == "" {
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
					for _, sep := range []string{":", "：", "**", "】", "-"} {
						contentAfterMarker = strings.TrimPrefix(contentAfterMarker, sep)
					}
					contentAfterMarker = strings.TrimSpace(contentAfterMarker)
					break
				}
			}
		}

		if foundSection != "" {
			// 前のセクションを保存（空文字で上書きしないよう check）
			// ※ 2026-04-24: 同じ section marker が 2 回出現する場合 (例: ■ SOAP ... ■ 【S - Subjective】 S ...) に
			//    後から出た方で意味ある content を持つ場合は上書きしたいので、「新 content が非空 OR 既存が空」
			//    の場合のみ上書きするロジックに変更。
			if currentSection != "" {
				if ptr, ok := sections[currentSection]; ok {
					newContent := strings.TrimSpace(strings.Join(currentLines, "\n"))
					if *ptr == "" || newContent != "" {
						*ptr = newContent
					}
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
