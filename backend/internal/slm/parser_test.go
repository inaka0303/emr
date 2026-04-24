package slm

import (
	"strings"
	"testing"
)

// TestParseSOAPFromText は実際の LoRA 出力で観測された各種フォーマット変化に対する
// parseSOAPFromText の耐性を検証する。parse 失敗は 4-call fallback (5x 計算コスト) を
// 誘発するため、ここの網羅性が M3 field notes 2026-04-23 の最大の UX 改善ポイント。
func TestParseSOAPFromText(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantS    string // 含まれるべき断片 (部分一致)
		wantO    string
		wantA    string
		wantP    string
	}{
		{
			// 2026-04-24 実地観測: `■ 【S - Subjective】` 形式 + 前置き + ■ SOAP 章見出し
			name: "■ 【S - Subjective】 形式 + ■ SOAP 章見出し",
			input: `特に心房細動の診断について論理的に整理しています。

■ SOAP 形式カルテ記載案
■ 【S - Subjective】
患者は先週から胸がドキドキする症状を訴える。

■ 【O - Objective】
BP 148/92, HR 88 不整

■ 【A - Assessment】
心房細動疑い

■ 【P - Plan】
抗凝固療法検討`,
			wantS: "胸がドキドキする",
			wantO: "BP 148/92",
			wantA: "心房細動",
			wantP: "抗凝固療法",
		},
		{
			// cleanModelOutput 後の形式 (太字→裸化)
			name: "S (Subjective) 形式 (cleaned)",
			input: `S (Subjective)
3 日前より動悸発症。

O (Objective)
BP 128/78, HR 92

A (Assessment)
AF 疑い

P (Plan)
心電図、エコー検査`,
			wantS: "3 日前より動悸",
			wantO: "BP 128/78",
			wantA: "AF 疑い",
			wantP: "心電図",
		},
		{
			// 基本形 S:/O:/A:/P:
			name: "S: O: A: P: 基本形",
			input: `S: 主訴は頭痛
O: BP 140/80
A: 緊張性頭痛
P: 鎮痛剤処方`,
			wantS: "頭痛",
			wantO: "140/80",
			wantA: "緊張性",
			wantP: "鎮痛剤",
		},
		{
			// ■ S: (行末スペースなし、marker に :: 付き)
			name: "■ S: 形式",
			input: `■ S:
発熱 38.5℃

■ O:
咽頭発赤

■ A:
急性咽頭炎

■ P:
抗菌薬`,
			wantS: "発熱",
			wantO: "咽頭",
			wantA: "急性",
			wantP: "抗菌薬",
		},
		{
			// 【S】【O】【A】【P】 形式
			name: "【S】【O】【A】【P】 形式",
			input: `【S】腹痛
【O】腹部圧痛
【A】急性胃腸炎
【P】安静指示`,
			wantS: "腹痛",
			wantO: "圧痛",
			wantA: "胃腸炎",
			wantP: "安静",
		},
		{
			// 太字 **S** ** O** 等（cleanModelOutput 通っていない場合の保険）
			name: "**S** 太字 markdown",
			input: `**S**
めまい

**O**
起立性低血圧

**A**
起立性調節障害

**P**
昇圧剤処方`,
			wantS: "めまい",
			wantO: "起立性",
			wantA: "調節障害",
			wantP: "昇圧剤",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSOAPFromText(tc.input)
			if !strings.Contains(got.Subjective, tc.wantS) {
				t.Errorf("S miss: want fragment %q, got %q", tc.wantS, got.Subjective)
			}
			if !strings.Contains(got.Objective, tc.wantO) {
				t.Errorf("O miss: want fragment %q, got %q", tc.wantO, got.Objective)
			}
			if !strings.Contains(got.Assessment, tc.wantA) {
				t.Errorf("A miss: want fragment %q, got %q", tc.wantA, got.Assessment)
			}
			if !strings.Contains(got.Plan, tc.wantP) {
				t.Errorf("P miss: want fragment %q, got %q", tc.wantP, got.Plan)
			}
			// 全セクションが空でないこと
			if got.Subjective == "" || got.Objective == "" || got.Assessment == "" || got.Plan == "" {
				t.Errorf("missing section: S=%q O=%q A=%q P=%q",
					truncateForTest(got.Subjective), truncateForTest(got.Objective),
					truncateForTest(got.Assessment), truncateForTest(got.Plan))
			}
		})
	}
}

func truncateForTest(s string) string {
	if len([]rune(s)) > 40 {
		return string([]rune(s)[:40]) + "..."
	}
	return s
}
