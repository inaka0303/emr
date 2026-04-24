package slm

import (
	"strings"
	"testing"
)

func TestCleanModelOutput(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantNot []string // 出力に含まれてはいけない文字列
		want    []string // 含まれるべき文字列
	}{
		{
			name: "markdown bold/header/hr を変換・除去",
			input: `# 入院時サマリ
**主訴:** 息苦しい
### 1. 主訴
* 起座呼吸
- 下腿浮腫
---
本文`,
			wantNot: []string{"**", "###", "* ", "\n- ", "---"},
			want:    []string{"■", "・起座呼吸", "・下腿浮腫"},
		},
		{
			name: "前置き文除去",
			input: `ご提示いただいた問診情報に基づきSOAPを作成しました。

S: 主訴は〜`,
			wantNot: []string{"ご提示"},
			want:    []string{"S: 主訴"},
		},
		{
			name: "補足説明ブロック末尾除去",
			input: `■ 治療方針
・入院
・抗菌薬

---
**補足説明（AI の提案ポイント）：**
*   これは補足です`,
			wantNot: []string{"補足説明", "AI の提案"},
			want:    []string{"抗菌薬"},
		},
		{
			name: "（※〜）丸括弧注釈除去",
			input: `所見は正常（※SpO2 97%、BP 118/72 はバイタルとして S に含まれる）。続き。`,
			wantNot: []string{"※", "バイタルとして"},
			want:    []string{"所見は正常", "続き"},
		},
		{
			name: "AI 自己言及コメント除去",
			input: `■ 治療方針
入院適応。
AIとしての最終判断は医師に委ねます。
専門医にご相談ください。`,
			wantNot: []string{"AIとして", "専門医にご相談"},
			want:    []string{"入院適応"},
		},
		{
			name: "作成者フッター除去",
			input: `■ 治療方針
入院。

---
**作成者:** 医療 AI アシスタント
**日付:** 202X 年 X 月`,
			wantNot: []string{"作成者", "医療 AI", "202X"},
			want:    []string{"入院"},
		},
		{
			name: "SOAP の S:/O: は保持",
			input: `S: 主訴 息苦しい
O: BP 158/94
A:
#1 急性心不全
P: 入院加療`,
			wantNot: []string{"**", "###"},
			want:    []string{"S: 主訴", "O: BP", "A:", "#1", "P: 入院"},
		},
		{
			name: "連続空行圧縮",
			input: `行1


行2




行3`,
			wantNot: []string{"\n\n\n"},
			want:    []string{"行1", "行2", "行3"},
		},
		{
			name: "残留 ** / *** を除去",
			input: `心機能低下 ** 認める。
利尿薬増量 *** 必要。
BNP上昇`,
			wantNot: []string{"**", "***"},
			want:    []string{"心機能低下", "利尿薬増量", "BNP上昇"},
		},
		{
			name: "インラインコード `text` を裸化",
			input: "バイタル `BP 148/88` 記録。\n検査値 `BNP 685` 参照。",
			wantNot: []string{"`"},
			want:    []string{"BP 148/88", "BNP 685"},
		},
		{
			name: "取り消し線 ~~text~~ を裸化",
			input: "~~中止~~ 継続に変更。",
			wantNot: []string{"~~"},
			want:    []string{"中止", "継続に変更"},
		},
		{
			name: "行末の装飾アスタリスクを除去",
			input: "治療方針決定 *\n次回再診予定 **",
			wantNot: []string{"方針決定 *", "再診予定 *"},
			want:    []string{"治療方針決定", "次回再診予定"},
		},
		{
			name: "【解説】前置きブロックを削除",
			input: `【解説】
電子カルテの「S（主観的情報）」欄は、患者本人が訴えている症状（主訴）と、その詳細を客観的な所見や検査結果が入る前に記述するものです。

ご提示いただいた問診記録を整理し、医療現場で一般的に用いられる簡潔な表現に書き換えました。

S: 2 日前より 38.5℃台の発熱を伴う咳嗽、咽頭痛を主訴。`,
			wantNot: []string{"【解説】", "電子カルテの", "書き換えました"},
			want:    []string{"S: 2 日前"},
		},
		{
			name: "■ 補足・注意点 末尾ブロックを削除（--- 無し版）",
			input: `P: 抗凝固薬DOAC開始、1週間後再診予定。

■ 補足・注意点
・抗凝固薬の選択: 患者の年齢、出血リスク、腎機能を考慮してDOACを選択
・血栓付着: 心房細動発症から3週間以内であれば心エコーで血栓評価
・服薬指導: 家族への協力依頼を併せて行う`,
			wantNot: []string{"補足・注意点", "血栓付着", "家族への協力依頼"},
			want:    []string{"P: 抗凝固薬DOAC"},
		},
		{
			name: "末尾 補足: プレーンテキスト版も削除",
			input: `P: フロセミド増量、2週間後再診。

補足説明：本症例は心腎連関に注意が必要です。利尿薬調整時は腎機能を頻回にチェックしてください。`,
			wantNot: []string{"補足説明", "心腎連関", "頻回にチェック"},
			want:    []string{"P: フロセミド増量"},
		},
		{
			name: "■ ポイント 末尾見出しも削除",
			input: `A: #1 慢性心不全急性増悪

■ ポイント
BNP著明上昇と塩分過多が診断の決め手`,
			wantNot: []string{"■ ポイント", "診断の決め手"},
			want:    []string{"慢性心不全急性増悪"},
		},
		{
			// 2026-04-23 M3 field notes: 池田里奈で観測されたメタ括弧ブロック
			// （※無し、複数行にまたがる、「記載するのが適切」「電子カルテでは」キーワード付き）
			name: "メタ括弧ブロック（複数行、池田里奈型）を削除",
			input: `S: 自覚症状なし。
（問診記録の冒頭にある「自覚症状なし」をそのまま記載するのが適切です。
電子カルテでは、問診で確認した主観的情報として簡潔に記録します。）
O: BP 128/78, HR 72`,
			wantNot: []string{"問診記録の冒頭", "記載するのが適切", "電子カルテでは", "簡潔に記録します"},
			want:    []string{"S: 自覚症状なし", "O: BP 128/78"},
		},
		{
			name: "メタ括弧ブロック（単一行）も削除",
			input: `A: 急性冠症候群の疑い。（この記載は臨床的に推奨されますが、確定診断には精査が必要です。）
P: 緊急入院。`,
			wantNot: []string{"推奨されます", "精査が必要"},
			want:    []string{"急性冠症候群", "P: 緊急入院"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanModelOutput(tc.input)
			for _, s := range tc.wantNot {
				if strings.Contains(got, s) {
					t.Errorf("含まれるべきでない文字列が残留: %q\n\n全出力:\n%s", s, got)
				}
			}
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("含まれるべき文字列が消えた: %q\n\n全出力:\n%s", s, got)
				}
			}
		})
	}
}
