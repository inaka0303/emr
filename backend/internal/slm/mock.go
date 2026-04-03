package slm

import (
	"math/rand"
	"strings"
	"time"
)

// mockDelay はリアルな推論時間を再現するための遅延を入れる（0.5〜1秒）
func mockDelay() {
	delay := 500 + rand.Intn(500)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// generateMockSOAP は問診テキストのキーワードに応じたSOAP提案を生成する
func generateMockSOAP(interviewText string) *SOAPSuggestion {
	mockDelay()

	text := strings.ToLower(interviewText)

	// 頭痛パターン
	if containsAny(text, "頭痛", "頭が痛い", "偏頭痛", "片頭痛", "ずつう") {
		return &SOAPSuggestion{
			Subjective: "3日前から持続する頭痛。拍動性で、特に右側頭部に集中。市販鎮痛剤（ロキソニンS）を服用するも改善なし。嘔気を伴うこともある。光過敏・音過敏あり。前兆としての視覚症状なし。",
			Objective:  "バイタル: BP 128/82mmHg, HR 76bpm, BT 36.4℃, SpO2 98%。意識清明。神経学的所見: 瞳孔左右同大・対光反射正常、四肢筋力低下なし、感覚異常なし。髄膜刺激徴候: 項部硬直(-)、Kernig徴候(-)、Brudzinski徴候(-)。側頭筋・後頸筋に圧痛あり。",
			Assessment: "緊張性頭痛の疑い。片頭痛との鑑別が必要。二次性頭痛（くも膜下出血、脳腫瘍等）の可能性は神経学的所見正常から低いが、持続期間を考慮し経過観察要。",
			Plan:       "ロキソプロフェン60mg 3T3× 5日分処方。筋弛緩薬（チザニジン1mg）2T2× 就寝前。頭痛日記の記載を指導。生活指導（睡眠・ストレス管理）。2週間後再診。改善なければ頭部MRI検討。",
		}
	}

	// 腹痛パターン
	if containsAny(text, "腹痛", "お腹が痛い", "胃痛", "腹部", "胃が痛い", "みぞおち") {
		return &SOAPSuggestion{
			Subjective: "昨日夕食後から心窩部痛を自覚。食後に増悪し、空腹時にも鈍痛が持続。嘔気あり、嘔吐なし。排便は昨日1回、普通便。下痢・血便なし。最近の食事内容に特記事項なし。ストレスが多い状況が続いている。",
			Objective:  "バイタル: BP 122/78mmHg, HR 72bpm, BT 36.6℃。腹部: 平坦・軟、心窩部に軽度圧痛あり、反跳痛(-)、筋性防御(-)。腸蠕動音: 正常。Murphy徴候(-)。CVA叩打痛(-)。",
			Assessment: "機能性ディスペプシア（FD）疑い。胃・十二指腸潰瘍の鑑別も必要。NSAIDs服用歴、H.pylori感染の確認要。",
			Plan:       "プロトンポンプ阻害薬（ランソプラゾール15mg）1T1× 朝食前 14日分処方。制酸薬（酸化マグネシウム330mg）3T3× 頓用。食事指導（刺激物・アルコール制限）。H.pylori抗体検査オーダー。2週間後再診。症状持続時は上部消化管内視鏡検討。",
		}
	}

	// 発熱パターン
	if containsAny(text, "発熱", "熱がある", "風邪", "咳", "のどが痛い", "咽頭痛", "感冒", "38度", "39度") {
		return &SOAPSuggestion{
			Subjective: "2日前からの発熱（最高38.5℃）、咽頭痛、全身倦怠感を主訴に来院。鼻汁・鼻閉あり。咳嗽は乾性で軽度。食欲低下あり。関節痛・筋肉痛を伴う。周囲にインフルエンザの流行あり。ワクチン未接種。",
			Objective:  "バイタル: BP 118/72mmHg, HR 88bpm, BT 38.2℃, SpO2 97%。咽頭: 発赤あり、白苔(-)、扁桃腫大(-)。頸部リンパ節: 軽度腫脹あり、圧痛(+)。胸部聴診: 呼吸音清、ラ音(-)。インフルエンザ迅速検査: A型(-)、B型(-)。",
			Assessment: "急性上気道炎（普通感冒）の疑い。インフルエンザは迅速検査陰性だが、発症早期のため偽陰性の可能性あり。溶連菌性咽頭炎の除外も考慮。",
			Plan:       "アセトアミノフェン500mg 頓用（6時間以上あけて1日3回まで）。トラネキサム酸250mg 3T3× 5日分。含嗽用アズレン処方。安静・水分補給指導。症状増悪時（高熱持続、呼吸困難等）は早期再診を指示。3日後再診。",
		}
	}

	// 腰痛パターン
	if containsAny(text, "腰痛", "腰が痛い", "ぎっくり腰", "腰部", "背中が痛い") {
		return &SOAPSuggestion{
			Subjective: "5日前に重い荷物を持ち上げた際に腰部に急性の痛みが出現。前屈時に増悪し、安静時にも鈍痛が持続。下肢のしびれ・放散痛なし。排尿障害なし。既往に腰痛あり（1年前にも同様のエピソード）。デスクワーク中心の業務。",
			Objective:  "バイタル: BP 130/80mmHg, HR 74bpm, BT 36.5℃。腰部: L4/5傍脊柱筋に圧痛あり、棘突起叩打痛(-)。SLRテスト: 左右とも陰性。下肢筋力: 正常。深部腱反射: 正常。膀胱直腸障害: なし。前屈制限あり（指床間距離30cm）。",
			Assessment: "急性腰痛症（筋筋膜性疼痛）。椎間板ヘルニアの神経根症状なし。Red flag signs（体重減少、発熱、膀胱直腸障害、進行性の神経症状）なし。",
			Plan:       "ロキソプロフェン60mg 3T3× 7日分処方。ミオナール50mg 3T3× 7日分。湿布薬（ロキソプロフェンテープ100mg）処方。安静は最小限とし、痛みの範囲で日常活動を継続するよう指導。腰痛体操のパンフレット配布。1週間後再診。改善なければ腰椎X線検査検討。",
		}
	}

	// 糖尿病パターン
	if containsAny(text, "血糖", "hba1c", "糖尿", "糖尿病", "口渇", "多飲", "多尿") {
		return &SOAPSuggestion{
			Subjective: "定期受診。最近口渇感が増し、飲水量が増えている。夜間尿2-3回。体重がこの1ヶ月で2kg減少。食事療法は概ね守れているが、間食が増えている自覚あり。運動は週2回のウォーキング（30分程度）を継続中。足のしびれなし。視力低下の自覚なし。",
			Objective:  "バイタル: BP 138/86mmHg, HR 78bpm, BT 36.4℃, 体重72kg（前回74kg）。HbA1c: 7.8%（前回7.2%）。空腹時血糖: 168mg/dL。尿検査: 糖(2+)、蛋白(-)。足背動脈触知可、足部感覚異常なし。アキレス腱反射: 正常。",
			Assessment: "2型糖尿病、血糖コントロール悪化。HbA1c 0.6%上昇は食事療法の遵守不十分が主因と考えられる。合併症スクリーニング（眼底検査、腎機能）の定期確認要。",
			Plan:       "メトホルミン500mg→750mgに増量（2T2× 朝夕食後）。食事指導の再強化（間食制限、炭水化物量の見直し）。管理栄養士による栄養指導を予約。自己血糖測定（SMBG）を2週間実施し食後血糖を記録。眼科受診を依頼（眼底検査）。4週間後再診。",
		}
	}

	// デフォルトパターン
	return &SOAPSuggestion{
		Subjective: "患者は上記症状を主訴に来院。発症時期、症状の経過、随伴症状について問診を実施。既往歴・服薬歴を確認。アレルギー歴: 特記事項なし。",
		Objective:  "バイタルサイン: BP 124/78mmHg, HR 72bpm, BT 36.5℃, SpO2 98%。全身状態: 良好。意識清明。一般身体所見にて著明な異常を認めず。",
		Assessment: "現時点で緊急性の高い所見は認めない。症状の経過観察と必要に応じた追加検査を検討。",
		Plan:       "対症療法を開始。生活指導実施。症状の変化があれば早期再診するよう指示。1-2週間後に再診予約。経過により追加の検査を検討。",
	}
}

// generateMockSummary は問診テキストからサマリー提案を生成する
func generateMockSummary(interviewText string, category string) *SummarySuggestion {
	mockDelay()

	text := strings.ToLower(interviewText)

	switch category {
	case "family_history":
		return generateMockFamilyHistory(text)
	case "social_history":
		return generateMockSocialHistory(text)
	default:
		return &SummarySuggestion{
			Suggestions: []SummarySuggestionItem{},
		}
	}
}

// familyHistoryMatch は家族歴マッチの中間表現
type familyHistoryMatch struct {
	relation           string
	relationConfidence float64
	condition          string
	conditionConfidence float64
}

// generateMockFamilyHistory は家族歴のモック提案を生成する
func generateMockFamilyHistory(text string) *SummarySuggestion {
	var matches []familyHistoryMatch

	// 父の疾患
	if strings.Contains(text, "父") {
		if containsAny(text, "糖尿", "血糖") {
			matches = append(matches, familyHistoryMatch{"父", 0.95, "2型糖尿病", 0.90})
		}
		if containsAny(text, "高血圧", "血圧") {
			matches = append(matches, familyHistoryMatch{"父", 0.95, "高血圧症", 0.92})
		}
		if containsAny(text, "心筋梗塞", "心臓", "狭心症") {
			matches = append(matches, familyHistoryMatch{"父", 0.95, "心筋梗塞", 0.88})
		}
		if containsAny(text, "脳卒中", "脳梗塞", "脳出血") {
			matches = append(matches, familyHistoryMatch{"父", 0.95, "脳血管疾患", 0.87})
		}
		if containsAny(text, "がん", "癌") {
			matches = append(matches, familyHistoryMatch{"父", 0.95, "悪性腫瘍", 0.85})
		}
	}

	// 母の疾患
	if strings.Contains(text, "母") {
		if containsAny(text, "糖尿", "血糖") {
			matches = append(matches, familyHistoryMatch{"母", 0.95, "2型糖尿病", 0.90})
		}
		if containsAny(text, "高血圧", "血圧") {
			matches = append(matches, familyHistoryMatch{"母", 0.95, "高血圧症", 0.91})
		}
		if containsAny(text, "骨粗鬆", "骨折") {
			matches = append(matches, familyHistoryMatch{"母", 0.93, "骨粗鬆症", 0.88})
		}
		if containsAny(text, "リウマチ", "関節") {
			matches = append(matches, familyHistoryMatch{"母", 0.93, "関節リウマチ", 0.86})
		}
	}

	// 兄弟姉妹の疾患
	if containsAny(text, "兄", "弟", "姉", "妹", "兄弟", "姉妹", "きょうだい") {
		relation := "同胞"
		if strings.Contains(text, "兄") {
			relation = "兄"
		} else if strings.Contains(text, "弟") {
			relation = "弟"
		} else if strings.Contains(text, "姉") {
			relation = "姉"
		} else if strings.Contains(text, "妹") {
			relation = "妹"
		}

		if containsAny(text, "喘息", "ぜんそく") {
			matches = append(matches, familyHistoryMatch{relation, 0.92, "気管支喘息", 0.88})
		}
		if containsAny(text, "アトピー", "湿疹") {
			matches = append(matches, familyHistoryMatch{relation, 0.92, "アトピー性皮膚炎", 0.86})
		}
	}

	// 祖父母の疾患
	if containsAny(text, "祖父", "祖母", "おじいちゃん", "おばあちゃん") {
		relation := "祖父母"
		if containsAny(text, "祖父", "おじいちゃん") {
			relation = "祖父"
		} else if containsAny(text, "祖母", "おばあちゃん") {
			relation = "祖母"
		}

		if containsAny(text, "認知症", "アルツハイマー", "物忘れ") {
			matches = append(matches, familyHistoryMatch{relation, 0.90, "認知症", 0.85})
		}
	}

	// マッチしなかった場合のデフォルト
	if len(matches) == 0 {
		return &SummarySuggestion{
			Suggestions: []SummarySuggestionItem{
				{Field: "relation", Value: "不明", Confidence: 0.50},
				{Field: "condition", Value: "問診テキストから家族歴情報を特定できませんでした", Confidence: 0.30},
			},
		}
	}

	// 重複チェック: relation+condition の組み合わせが同一のエントリを除外
	suggestions := []SummarySuggestionItem{}
	seen := make(map[string]bool)
	for _, m := range matches {
		key := m.relation + ":" + m.condition
		if seen[key] {
			continue
		}
		seen[key] = true
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "relation", Value: m.relation, Confidence: m.relationConfidence},
			SummarySuggestionItem{Field: "condition", Value: m.condition, Confidence: m.conditionConfidence},
		)
	}

	return &SummarySuggestion{Suggestions: suggestions}
}

// generateMockSocialHistory は社会歴のモック提案を生成する
func generateMockSocialHistory(text string) *SummarySuggestion {
	suggestions := []SummarySuggestionItem{}

	// 喫煙
	if containsAny(text, "喫煙", "タバコ", "たばこ", "煙草", "本/日", "本吸") {
		description := "喫煙歴あり"
		if containsAny(text, "20本", "１箱", "1箱") {
			description = "20本/日"
		} else if containsAny(text, "10本") {
			description = "10本/日"
		} else if containsAny(text, "禁煙", "やめた", "辞めた") {
			description = "過去に喫煙歴あり（現在禁煙中）"
		}
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "喫煙", Confidence: 0.95},
			SummarySuggestionItem{Field: "description", Value: description, Confidence: 0.88},
		)
	}

	// 飲酒
	if containsAny(text, "飲酒", "アルコール", "ビール", "日本酒", "焼酎", "ワイン", "お酒") {
		description := "飲酒歴あり"
		if containsAny(text, "毎日", "毎晩", "daily") {
			description = "常習飲酒（毎日）"
		} else if containsAny(text, "週") {
			description = "機会飲酒"
		} else if containsAny(text, "禁酒", "やめた", "飲まない") {
			description = "過去に飲酒歴あり（現在禁酒中）"
		}
		if containsAny(text, "ビール") {
			description += "、ビール中心"
		} else if containsAny(text, "日本酒") {
			description += "、日本酒中心"
		} else if containsAny(text, "焼酎") {
			description += "、焼酎中心"
		}
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "飲酒", Confidence: 0.93},
			SummarySuggestionItem{Field: "description", Value: description, Confidence: 0.85},
		)
	}

	// 職業
	if containsAny(text, "職業", "仕事", "会社員", "事務", "営業", "教師", "自営", "無職", "退職", "パート", "アルバイト", "デスクワーク") {
		description := "職業情報あり"
		if containsAny(text, "会社員", "サラリーマン") {
			description = "会社員"
		} else if containsAny(text, "事務") {
			description = "事務職（デスクワーク中心）"
		} else if containsAny(text, "営業") {
			description = "営業職（外回りあり）"
		} else if containsAny(text, "教師", "教員") {
			description = "教育職"
		} else if containsAny(text, "自営") {
			description = "自営業"
		} else if containsAny(text, "無職") {
			description = "無職"
		} else if containsAny(text, "退職") {
			description = "退職済み"
		} else if containsAny(text, "デスクワーク") {
			description = "デスクワーク中心"
		}
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "職業", Confidence: 0.92},
			SummarySuggestionItem{Field: "description", Value: description, Confidence: 0.86},
		)
	}

	// 運動
	if containsAny(text, "運動", "ウォーキング", "ジョギング", "ジム", "水泳", "散歩", "スポーツ") {
		description := "運動習慣あり"
		if containsAny(text, "ウォーキング", "散歩") {
			description = "ウォーキング習慣あり"
		} else if containsAny(text, "ジョギング", "ランニング") {
			description = "ジョギング習慣あり"
		} else if containsAny(text, "ジム") {
			description = "ジム通い"
		} else if containsAny(text, "水泳") {
			description = "水泳"
		} else if containsAny(text, "しない", "なし", "していない") {
			description = "運動習慣なし"
		}
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "運動", Confidence: 0.90},
			SummarySuggestionItem{Field: "description", Value: description, Confidence: 0.84},
		)
	}

	// 睡眠
	if containsAny(text, "睡眠", "不眠", "眠れない", "寝付き", "夜更かし") {
		description := "睡眠に関する情報あり"
		if containsAny(text, "不眠", "眠れない") {
			description = "不眠傾向あり"
		} else if containsAny(text, "夜更かし") {
			description = "就寝時刻が遅い傾向"
		}
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "睡眠", Confidence: 0.88},
			SummarySuggestionItem{Field: "description", Value: description, Confidence: 0.80},
		)
	}

	// マッチしなかった場合のデフォルト
	if len(suggestions) == 0 {
		suggestions = append(suggestions,
			SummarySuggestionItem{Field: "category", Value: "不明", Confidence: 0.50},
			SummarySuggestionItem{Field: "description", Value: "問診テキストから社会歴情報を特定できませんでした", Confidence: 0.30},
		)
	}

	return &SummarySuggestion{Suggestions: suggestions}
}

// containsAny はテキストが指定したキーワードのいずれかを含むかチェックする
func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
