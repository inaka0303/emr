package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/ehr-demo/internal/model"
)

// PatientHistoryService は患者の過去受診歴を集約して SLM 用の "要約コンテキスト" を生成する。
//
// 設計動機 (2026-04-24, M3 field notes B アーキ改善 ⭐⭐):
// 山本隆型の多 encounter 患者で、現在の受診 SOAP を生成する際に
// 過去の診療方針・診断を踏まえた記載にしたい。ナイーブに全 encounter の
// interview + SOAP を結合すると prompt が長大になり 4B prefill 時間が爆発する。
// そこで「brief で意思決定に必要な要点のみ」を各 encounter から抽出して
// 連結する rule-based 要約レイヤーを先に scaffold 実装する（後に LoRA 要約へ置換可）。
//
// 現時点の実装戦略 (scaffold):
//   - 過去 encounter を日付降順で取得（最大 5 件）
//   - 各 encounter につき、date + chief_complaint + 既存 SOAP の A/P を 1-2 行で抜粋
//   - 120 文字程度の箇条書きに整形
//   - 合計で 800 文字を超えないよう上限制御
//
// 未対応 (TODO):
//   - LoRA 要約モデル（過去 interview 全文 → 200 文字の臨床サマリ）
//   - 薬剤履歴・検査結果の時系列変化トラッキング
//   - 診療科横断の文脈統合
type PatientHistoryService struct {
	encounterSvc *EncounterService
	soapSvc      *SOAPService
	interviewSvc *InterviewService
}

// NewPatientHistoryService は PatientHistoryService を生成する。
func NewPatientHistoryService(
	encounterSvc *EncounterService,
	soapSvc *SOAPService,
	interviewSvc *InterviewService,
) *PatientHistoryService {
	return &PatientHistoryService{
		encounterSvc: encounterSvc,
		soapSvc:      soapSvc,
		interviewSvc: interviewSvc,
	}
}

// BuildCrossEncounterSummary は patientID の過去受診履歴を要約し、
// SOAP / admission 生成時に interview_text 先頭に prepend するためのブロックを返す。
// excludeEncounterID に一致する encounter は除外（= 現在生成中のものを含めない）。
// 戻り値が空文字ならば要約すべき過去履歴が無い（= 初診）。
//
// フォーマット:
//
//	【過去受診歴サマリ】(直近 N 件、古→新)
//	- YYYY-MM-DD [診療科]: 主訴 "..." / A: ... / P: ...
//	- ...
func (s *PatientHistoryService) BuildCrossEncounterSummary(
	ctx context.Context,
	patientID int64,
	excludeEncounterID int64,
) (string, error) {
	const maxEncounters = 5
	const softCharLimit = 800

	encs, err := s.encounterSvc.ListByPatientID(ctx, patientID)
	if err != nil {
		return "", fmt.Errorf("list encounters: %w", err)
	}
	// 除外 + 日付降順で maxEncounters 件まで
	filtered := make([]model.Encounter, 0, len(encs))
	for _, e := range encs {
		if e.ID == excludeEncounterID {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == 0 {
		return "", nil
	}
	// ListByPatientID は通常日付降順で返す想定。保険で手動 sort しても良い。
	// 新しい順に maxEncounters 件取得 → summary は古→新の順で表示する
	if len(filtered) > maxEncounters {
		filtered = filtered[:maxEncounters]
	}

	// 古→新の順で出力（医学的文脈は時系列が自然）
	reverse(filtered)

	var b strings.Builder
	b.WriteString("【過去受診歴サマリ】(直近 ")
	fmt.Fprintf(&b, "%d", len(filtered))
	b.WriteString(" 件、古→新)\n")

	for _, e := range filtered {
		// 1 encounter = 1 行記述
		line := fmt.Sprintf("- %s [%s]: 主訴「%s」",
			e.EncounterDate, e.Department, truncateFirstLine(e.ChiefComplaint, 40))

		// 既存 SOAP から A/P を抜粋（存在すれば）
		soaps, err := s.soapSvc.GetByEncounterID(ctx, e.ID)
		if err == nil && len(soaps) > 0 {
			// 最新の SOAP を使う
			s := soaps[len(soaps)-1]
			if a := truncateFirstLine(s.Assessment, 60); a != "" {
				line += " / A: " + a
			}
			if p := truncateFirstLine(s.Plan, 60); p != "" {
				line += " / P: " + p
			}
		}
		line += "\n"

		// 上限チェック（途中で打ち切る）
		if b.Len()+len(line) > softCharLimit {
			b.WriteString("- ... (以前の受診記録は省略)\n")
			break
		}
		b.WriteString(line)
	}

	return b.String(), nil
}

// truncateFirstLine は string を 1 行目だけ取り、指定 rune 数で切り詰める。
// 改行や句点で切って末尾に "..." を足す。
func truncateFirstLine(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if idx := strings.IndexAny(s, "\n\r"); idx > 0 {
		s = s[:idx]
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

// reverse は slice を in-place で反転する。
func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
