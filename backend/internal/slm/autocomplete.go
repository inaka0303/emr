package slm

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"
)

// CompletionEntry は補完辞書のエントリ
type CompletionEntry struct {
	Prefix     string // 入力テキストの前方一致プレフィックス
	Completion string // 補完後のフルテキスト
}

// AutocompleteResult はオートコンプリートの結果
type AutocompleteResult struct {
	Completion string `json:"completion"` // 補完テキスト（入力テキストの続き）
	FullText   string `json:"full_text"`  // 入力テキスト + 補完を結合したもの
}

// AutocompleteMeta はオートコンプリートのメタ情報
type AutocompleteMeta struct {
	Model     string `json:"model"`
	IsMock    bool   `json:"is_mock"`
	LatencyMs int64  `json:"latency_ms"`
}

// family_history コンテキストの補完辞書
var familyHistoryCompletions = []CompletionEntry{
	// 続柄 + 疾患パターン
	{Prefix: "父方", Completion: "父方の祖父：胃がん（70歳で死亡）"},
	{Prefix: "母方", Completion: "母方の祖母：骨粗鬆症"},
	{Prefix: "祖父", Completion: "祖父：脳梗塞（75歳で発症、左半身麻痺）"},
	{Prefix: "祖母", Completion: "祖母：アルツハイマー型認知症（80歳頃から）"},
	{Prefix: "父", Completion: "父：2型糖尿病（60歳で発症、インスリン療法中）"},
	{Prefix: "ち", Completion: "父：2型糖尿病（60歳で発症、インスリン療法中）"},
	{Prefix: "chi", Completion: "父：家族性大腸腺腫"},
	{Prefix: "母", Completion: "母：高血圧症（55歳で発症、降圧剤服用中）"},
	{Prefix: "は", Completion: "母：高血圧症（55歳で発症、降圧剤服用中）"},
	{Prefix: "ha", Completion: "母：関節リウマチ（40代で発症）"},
	{Prefix: "兄", Completion: "兄：気管支喘息（小児期から）"},
	{Prefix: "姉", Completion: "姉：橋本病（30歳で診断）"},
	// 疾患名プレフィックス
	{Prefix: "糖尿", Completion: "糖尿病（2型）：父"},
	{Prefix: "高血圧", Completion: "高血圧症：母（55歳で発症）"},
	{Prefix: "がん", Completion: "がん：祖父（胃がん、70歳）"},
	{Prefix: "心筋", Completion: "心筋梗塞：父（65歳で発症）"},
	{Prefix: "脳", Completion: "脳血管疾患：祖父（脳梗塞）"},
}

// social_history コンテキストの補完辞書
var socialHistoryCompletions = []CompletionEntry{
	{Prefix: "喫煙", Completion: "喫煙：20本/日×20年（ブリンクマン指数400）"},
	{Prefix: "ki", Completion: "喫煙：20本/日×20年（ブリンクマン指数400）"},
	{Prefix: "禁煙", Completion: "禁煙：3年前に禁煙（以前は20本/日×15年）"},
	{Prefix: "飲酒", Completion: "飲酒：ビール350ml×2本/日、休肝日なし"},
	{Prefix: "in", Completion: "飲酒：ビール350ml×2本/日"},
	{Prefix: "職業", Completion: "職業：会社員（デスクワーク中心、残業月40時間）"},
	{Prefix: "運動", Completion: "運動：週2回ウォーキング30分"},
	{Prefix: "睡眠", Completion: "睡眠：平均6時間、中途覚醒あり"},
	{Prefix: "アレ", Completion: "アレルギー：花粉症（スギ、ヒノキ）、薬剤アレルギーなし"},
	{Prefix: "ス", Completion: "ストレス：仕事のストレス高い、最近不眠気味"},
}

// soap_subjective コンテキストの補完辞書
var soapSubjectiveCompletions = []CompletionEntry{
	{Prefix: "頭痛", Completion: "頭痛：3日前から持続、拍動性、右側頭部"},
	{Prefix: "腹痛", Completion: "腹痛：昨日夕食後から、心窩部、持続性"},
	{Prefix: "発熱", Completion: "発熱：本日朝から38.5℃、悪寒あり、咽頭痛あり"},
	{Prefix: "咳", Completion: "咳嗽：1週間前から、乾性、夜間増悪"},
	{Prefix: "腰痛", Completion: "腰痛：2日前から、前屈時増悪、下肢放散痛なし"},
	{Prefix: "め", Completion: "めまい：起床時に回転性、嘔気を伴う"},
	{Prefix: "胸", Completion: "胸痛：労作時、前胸部の圧迫感、安静で改善"},
	{Prefix: "息", Completion: "息切れ：階段昇降時、最近増悪傾向"},
	{Prefix: "動悸", Completion: "動悸：安静時にも出現、持続時間は数分"},
	{Prefix: "倦怠", Completion: "倦怠感：2週間前から持続、食欲低下を伴う"},
	{Prefix: "関節", Completion: "関節痛：両手指の朝のこわばり、30分以上持続"},
}

// soap_objective コンテキストの補完辞書
var soapObjectiveCompletions = []CompletionEntry{
	{Prefix: "バイタル", Completion: "バイタル：BP 130/85mmHg, HR 76bpm, BT 36.5℃, SpO2 98%"},
	{Prefix: "BP", Completion: "BP 130/85mmHg, HR 76bpm, BT 36.5℃, SpO2 98%"},
	{Prefix: "体温", Completion: "体温 36.5℃、血圧 130/85mmHg、脈拍 76/分"},
	{Prefix: "胸部", Completion: "胸部聴診：呼吸音清、ラ音(-)、心音整、雑音(-)"},
	{Prefix: "腹部", Completion: "腹部：平坦・軟、圧痛(-)、反跳痛(-)、筋性防御(-)"},
	{Prefix: "神経", Completion: "神経学的所見：意識清明、瞳孔左右同大、対光反射正常、四肢筋力低下なし"},
	{Prefix: "心電図", Completion: "心電図：洞調律、ST変化なし、不整脈なし"},
	{Prefix: "血液", Completion: "血液検査：WBC 6500/μL, CRP 0.3mg/dL, Hb 13.5g/dL"},
}

// soap_assessment コンテキストの補完辞書
var soapAssessmentCompletions = []CompletionEntry{
	{Prefix: "緊張", Completion: "緊張性頭痛の疑い。片頭痛との鑑別が必要"},
	{Prefix: "急性", Completion: "急性上気道炎（普通感冒）の疑い"},
	{Prefix: "高血圧", Completion: "高血圧症、血圧コントロール不良。生活習慣の見直しが必要"},
	{Prefix: "2型", Completion: "2型糖尿病、血糖コントロール悪化傾向。HbA1c目標値未達"},
	{Prefix: "機能性", Completion: "機能性ディスペプシア疑い。器質的疾患の除外が必要"},
	{Prefix: "労作性", Completion: "労作性狭心症の疑い。冠動脈疾患の精査が必要"},
}

// soap_plan コンテキストの補完辞書
var soapPlanCompletions = []CompletionEntry{
	{Prefix: "ロキソ", Completion: "ロキソプロフェン60mg 3T3× 5日分処方"},
	{Prefix: "アセト", Completion: "アセトアミノフェン500mg 頓用（6時間以上あけて1日3回まで）"},
	{Prefix: "処方", Completion: "処方箋発行。服薬指導実施。"},
	{Prefix: "再診", Completion: "再診：2週間後。症状悪化時は早期受診を指示"},
	{Prefix: "検査", Completion: "検査オーダー：血液検査（CBC, CRP, 生化学）、胸部X線"},
	{Prefix: "生活", Completion: "生活指導：十分な睡眠、ストレス管理、適度な運動を推奨"},
	{Prefix: "紹介", Completion: "紹介状作成：専門科へのコンサルテーション依頼"},
}

// contextCompletionMap はコンテキストと補完辞書のマッピング
var contextCompletionMap = map[string][]CompletionEntry{
	"family_history":  familyHistoryCompletions,
	"social_history":  socialHistoryCompletions,
	"soap_subjective": soapSubjectiveCompletions,
	"soap_objective":  soapObjectiveCompletions,
	"soap_assessment": soapAssessmentCompletions,
	"soap_plan":       soapPlanCompletions,
}

// contextToSystemPrompt はコンテキストタイプに応じたシステムプロンプトを返す
//
// SOAP系は suggest LoRA の訓練時と同一のシステムプロンプトを使う（訓練分布と揃える）。
// 家族歴/社会歴は既存のままにしておく（別用途のため）。
func contextToSystemPrompt(contextType string) string {
	switch contextType {
	case "soap_subjective", "soap_objective", "soap_assessment", "soap_plan":
		return "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。問診情報を元に、カルテの続きを提案します。"
	case "family_history":
		return "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。家族歴の記載を続けてください。続きのテキストのみを出力してください。"
	case "social_history":
		return "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。社会歴の記載を続けてください。続きのテキストのみを出力してください。"
	default:
		return "あなたは日本語の電子カルテ記載を支援する医療AIアシスタントです。入力テキストの続きを提案してください。続きのテキストのみを出力してください。"
	}
}

// soapSectionMeta はSOAPコンテキストタイプからセクション情報を返す
func soapSectionMeta(contextType string) (letter, fullName string, ok bool) {
	switch contextType {
	case "soap_subjective":
		return "S", "S（主観的情報）", true
	case "soap_objective":
		return "O", "O（客観的情報）", true
	case "soap_assessment":
		return "A", "A（評価）", true
	case "soap_plan":
		return "P", "P（計画）", true
	}
	return "", "", false
}

// buildPrefillPrompt は suggest LoRA 向けに user prompt と assistant prefill を組み立てる
//
// 訓練データと整合するパターン:
//   [user]      【問診記録】...
//               【記載済み】S: ... O: ...    （現セクションより前のもの）
//               上記に続けて、{現セクション名}を記載してください。
//   [assistant] {letter}: {入力中partialText}      ← ここが prefill
//   [生成]       partial の続きの文
//
// llama.cpp の --prefill-assistant デフォルト有効機能により、
// 末尾が assistant role のメッセージは「続き生成」モードに入る。
// → 入力 echo / section ラベル混入 / 重複 が構造的に発生しなくなる。
//
// interviewText が空 or 非SOAPコンテキストの場合は旧挙動（textをuserに入れるだけ、prefill無し）。
func buildPrefillPrompt(contextType, text, interviewText string, priorSections map[string]string) (userPrompt, assistantPrefill string) {
	letter, fullName, ok := soapSectionMeta(contextType)
	if !ok || interviewText == "" {
		return text, ""
	}

	var u strings.Builder
	u.WriteString("【問診記録】\n")
	u.WriteString(strings.TrimSpace(interviewText))
	u.WriteString("\n")

	// 現セクションより前の記載済み内容を【記載済み】ブロックで提示
	hasPrior := false
	for _, prev := range []string{"S", "O", "A", "P"} {
		if prev == letter {
			break
		}
		if v, has := priorSections[prev]; has && strings.TrimSpace(v) != "" {
			if !hasPrior {
				u.WriteString("\n【記載済み】\n")
				hasPrior = true
			}
			u.WriteString(prev)
			u.WriteString(": ")
			u.WriteString(strings.TrimSpace(v))
			u.WriteString("\n")
		}
	}

	if hasPrior {
		u.WriteString("\n上記に続けて、")
		u.WriteString(fullName)
		u.WriteString("を記載してください。")
	} else {
		u.WriteString("\n上記の問診記録から、")
		u.WriteString(fullName)
		u.WriteString("を記載してください。")
	}

	// assistant prefill: 「S: {ユーザー入力中}」のように前置きを書く
	// → モデルは prefill の続きを生成する（入力 echo や section ラベル混入が起きない）
	prefill := letter + ": " + text
	return u.String(), prefill
}

// Autocomplete は入力テキストに対するインライン補完を返す
// SLM接続時はllama-server APIを使い、非接続時はモック辞書にフォールバック
//
// interviewText / priorSections を渡すと、suggest LoRAの訓練分布に近いプロンプトを構築し、
// 患者コンテキストに基づいたハルシネーションの少ない補完が得られる。
func (c *Client) Autocomplete(text, contextType, interviewText string, priorSections map[string]string) (*AutocompleteResult, bool, time.Duration) {
	start := time.Now()

	// 空テキストの場合は空の結果を返す
	if text == "" {
		latency := time.Since(start)
		return &AutocompleteResult{
			Completion: "",
			FullText:   "",
		}, true, latency
	}

	// 短すぎるテキスト（2文字未満）はモック辞書で対応
	if len([]rune(text)) < 2 {
		return c.autocompleteFromDict(text, contextType, start)
	}

	c.mu.RLock()
	useMock := c.useMock
	c.mu.RUnlock()

	if useMock {
		return c.autocompleteFromDict(text, contextType, start)
	}

	// SLM APIで補完を生成（suggest LoRA使用、assistant prefill方式）
	systemPrompt := contextToSystemPrompt(contextType)
	userPrompt, assistantPrefill := buildPrefillPrompt(contextType, text, interviewText, priorSections)

	// timeout: cache miss 型の初回リクエストは M3 実測 6-9s、長 interview では 15s を超える
	// ケースがあるため 30s に延長（field notes 2026-04-23 による）。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// コンテキストに応じてmax_tokensを調整
	// インライン補完は「続きの短い 1 文」が目的なので 48 を基本値に圧縮。
	// A 記載のみ臨床推論が長くなりがちなので 96 を維持。
	maxTokens := 48
	switch contextType {
	case "soap_assessment":
		maxTokens = 96 // A記載はやや長めの臨床推論まで含む
	case "soap_objective":
		maxTokens = 64
	}
	// temperature は 0.5 → 0.3 に下げて決定的に。補完 UX では一貫性を優先。
	result, err := c.callChatCompletionWithLoRA(ctx, systemPrompt, userPrompt, assistantPrefill, maxTokens, 0.3, LoRASuggestID)
	if err != nil {
		log.Println("SLM autocomplete失敗、モック辞書にフォールバック", "error", err)
		return c.autocompleteFromDict(text, contextType, start)
	}

	// SLMの出力をクリーンアップ
	completion := strings.TrimSpace(result)
	// 入力テキストの繰り返しを除去（完全一致）
	if strings.HasPrefix(completion, text) {
		completion = strings.TrimSpace(completion[len(text):])
	}
	// 入力テキストの末尾部分が重複している場合を除去
	// 例: 入力「労作時の」→ SLM出力「労作時の胸やけ」→ 「胸やけ」だけ取る
	textRunes := []rune(text)
	compRunes := []rune(completion)
	for overlapLen := len(textRunes); overlapLen > 0; overlapLen-- {
		suffix := string(textRunes[len(textRunes)-overlapLen:])
		if strings.HasPrefix(completion, suffix) {
			completion = strings.TrimSpace(string(compRunes[len([]rune(suffix)):]))
			break
		}
	}
	// "S:" "O:" "A:" "P:" など不要なプレフィックスを除去
	for _, prefix := range []string{"S:", "O:", "A:", "P:", "S.", "O.", "A.", "P.", "S：", "O：", "A：", "P："} {
		completion = strings.TrimPrefix(completion, prefix)
		completion = strings.TrimSpace(completion)
	}
	// meta-commentary（ベースモデル特性の漏れ）を検出してカット
	// 例: 「※AI として提案した」「これに基づいて、以下の情報を追加して...」
	// 訓練不足の LoRA では分布外入力時にベースモデル特性が出やすい
	// 2026-04-23 M3 field notes で発見された新パターン:
	//   「（問診記録の冒頭にある〜そのまま記載するのが適切です」
	//   「電子カルテでは、問診で確認した主観的情報として〜」
	for _, marker := range []string{
		"※AI", "※A I", "※上記", "※この", "※なお",
		"これに基づいて、以下",
		"以下の情報を追加",
		"以下のような情報",
		"以下のような内容",
		"以下の内容を記載",
		"以下の点",
		"上記を参考に",
		"提案します：",
		"提案します:",
		"ご確認ください",
		"必要に応じて",
		"AI として",
		"AIとして",
		"注意:",
		"注意：",
		"以下は",
		"補足：",
		"補足:",
		// 2026-04-23 M3 field notes: メタ説明の丸括弧ブロック先頭を捕捉
		"（問診記録の",
		"（以下の",
		"（電子カルテでは",
		"（この記載は",
		"記載するのが適切",
		"電子カルテでは",
		"そのまま記載する",
		"簡潔に記録します",
		"適切に記載します",
		"■ 提案",
		// 2026-04-24 短入力メタ化対策:
		// `動悸` → `「動悸」が主訴として記録されましたね...■ パターン 1:`
		// `胸痛` → `「胸痛」は非常に一般的な主訴ですが...`
		"が主訴として記録されましたね",
		"として記録されましたね",
		"記録されましたね",
		"非常に一般的な主訴",
		"提案させていただきます",
		"させていただきます",
		"詳細な記述が変わる",
		"いくつかのパターン",
		"状況に合わせて調整",
		"ご使用ください",
		"患者様の状態",
		"■ パターン",
		"パターン 1",
		"パターン1",
		"パターン2",
		"パターン3",
		// model が入力を「」で囲んで meta 説明するパターン
		"」が主訴",
		"」は非常に",
		"」について",
	} {
		if idx := strings.Index(completion, marker); idx >= 0 {
			completion = strings.TrimRight(completion[:idx], " 　、。\n\t")
			break
		}
	}
	// 現セクションを超えて次セクション(O: / A: / P: 等)に進んでしまった応答をカット
	// 例: S入力中に「症状なし。\n\nO: ...」と返ってきたら「症状なし。」までに切る
	if currentLetter, _, isSOAP := soapSectionMeta(contextType); isSOAP {
		nextLetters := []string{"O", "A", "P"}
		if currentLetter == "O" {
			nextLetters = []string{"A", "P"}
		} else if currentLetter == "A" {
			nextLetters = []string{"P"}
		} else if currentLetter == "P" {
			nextLetters = nil
		}
		earliest := -1
		for _, nl := range nextLetters {
			for _, sep := range []string{"\n" + nl + ":", "\n" + nl + "：", "\n\n" + nl + ":", "\n\n" + nl + "："} {
				if idx := strings.Index(completion, sep); idx >= 0 {
					if earliest < 0 || idx < earliest {
						earliest = idx
					}
				}
			}
		}
		if earliest >= 0 {
			completion = strings.TrimSpace(completion[:earliest])
		}
	}
	// メタ応答の全面破棄: marker cut 後でも残骸が意味をなさないケース。
	// 2026-04-24: 「動悸」→「『動悸』が主訴として記録されましたね」等の短入力メタ化で
	// marker cut 後 `「動悸` という無意味な残骸が出る。こういうのは空返しが良い。
	//
	// 判定:
	//   (a) strongMeta (丁寧語・AI 自己言及系) がひとつでも含まれる → 破棄
	//   (b) 冒頭が `「` や `『` で始まる + 続きに意味のある clinical 表現が無い → 破棄
	// 単独の `「` ` 」` は real clinical text でも使われる（例: 「労作時」の胸痛）ので weak 扱い。
	strongMetaSignatures := []string{
		"患者様",
		"いただきます",
		"ご使用",
		"ご提案",
		"ご参照",
		"アシスタント",
		"記録されました",
		"検討くださ",
		"お役に立て",
		"いくつかのパターン",
		"状況に合わせて",
		"ご確認くださ",
	}
	for _, sig := range strongMetaSignatures {
		if strings.Contains(completion, sig) {
			completion = ""
			break
		}
	}
	// 残骸判定: 冒頭が `「` or `『` で、かつ短すぎる（5 文字以内）→ 破棄
	if completion != "" {
		if r := []rune(completion); len(r) <= 5 && (strings.HasPrefix(completion, "「") || strings.HasPrefix(completion, "『")) {
			completion = ""
		}
	}
	// 入力が短い (< 6 文字) かつ completion が `「` で始まる → 入力を「」で引用する meta パターン
	// 例: 入力「動悸」→ completion「『動悸』の症状」。clinical 続きではないので破棄。
	if completion != "" {
		if len([]rune(text)) < 6 && (strings.HasPrefix(completion, "「") || strings.HasPrefix(completion, "『")) {
			completion = ""
		}
	}

	// 反復検出: 同じ20文字以上のフレーズが2回以上出たらカット
	compRunesCheck := []rune(completion)
	if len(compRunesCheck) > 40 {
		for checkLen := 20; checkLen <= len(compRunesCheck)/2; checkLen++ {
			chunk := string(compRunesCheck[:checkLen])
			rest := string(compRunesCheck[checkLen:])
			if strings.Contains(rest, chunk) {
				completion = string(compRunesCheck[:checkLen])
				break
			}
		}
	}
	// 長すぎる補完を切り詰め（1文程度に）
	if runes := []rune(completion); len(runes) > 100 {
		// 句点・改行で切る
		for i, r := range runes {
			if i > 20 && (r == '。' || r == '\n' || r == '．') {
				completion = string(runes[:i+1])
				break
			}
		}
		if len([]rune(completion)) > 100 {
			completion = string(runes[:100])
		}
	}

	latency := time.Since(start)
	log.Println("SLM autocomplete完了",
		"context", contextType,
		"input_len", len([]rune(text)),
		"completion_len", len([]rune(completion)),
		"latency_ms", latency.Milliseconds())

	return &AutocompleteResult{
		Completion: completion,
		FullText:   text + completion,
	}, false, latency
}

// autocompleteFromDict はモック辞書ベースの補完（フォールバック用）
func (c *Client) autocompleteFromDict(text string, contextType string, start time.Time) (*AutocompleteResult, bool, time.Duration) {
	// 短い遅延を入れてリアルな体感にする（30-80ms）
	delay := 30 + rand.Intn(50)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	entries, ok := contextCompletionMap[contextType]
	if !ok {
		latency := time.Since(start)
		return &AutocompleteResult{
			Completion: "",
			FullText:   text,
		}, true, latency
	}

	var bestMatch *CompletionEntry
	bestPrefixLen := 0
	var bestPartialMatch *CompletionEntry
	bestPartialLen := int(^uint(0) >> 1)

	for i := range entries {
		entry := &entries[i]
		prefixLen := len([]rune(entry.Prefix))

		if strings.HasPrefix(text, entry.Prefix) {
			if prefixLen > bestPrefixLen {
				bestMatch = entry
				bestPrefixLen = prefixLen
			}
		} else if strings.HasPrefix(entry.Prefix, text) {
			if prefixLen < bestPartialLen {
				bestPartialMatch = entry
				bestPartialLen = prefixLen
			}
		}
	}

	if bestMatch == nil {
		bestMatch = bestPartialMatch
	}

	if bestMatch == nil {
		latency := time.Since(start)
		return &AutocompleteResult{
			Completion: "",
			FullText:   text,
		}, true, latency
	}

	completionFullRunes := []rune(bestMatch.Completion)
	textRunes := []rune(text)

	var completion string
	if strings.HasPrefix(bestMatch.Completion, text) {
		completion = string(completionFullRunes[len(textRunes):])
	} else {
		completion = bestMatch.Completion
	}

	latency := time.Since(start)
	return &AutocompleteResult{
		Completion: completion,
		FullText:   text + completion,
	}, true, latency
}
