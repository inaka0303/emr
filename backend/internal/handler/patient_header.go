package handler

import (
	"fmt"
	"time"

	"github.com/example/ehr-demo/internal/model"
)

// buildPatientHeader は SLM の入力 prompt 先頭に付与する患者属性ヘッダを生成する。
// 例: "【患者情報】62歳 女性\n"
//
// referenceDate が "YYYY-MM-DD" なら受診日時点の年齢、空なら今日時点で計算。
// patient が nil、または必要情報が欠けている場合は空文字（注入スキップ）。
//
// 目的: SLM が性別・年齢を仮定してしまう問題（例: 4B が女性患者を「男性」と
// 誤認して CHA₂DS₂-VASc を誤計算）を防ぐ。訓練時データの多くが患者属性を
// 冒頭に明示しているため、この形で注入すると分布内になり精度が上がる。
func buildPatientHeader(p *model.Patient, referenceDate string) string {
	if p == nil {
		return ""
	}
	age := calculateAge(p.BirthDate, referenceDate)
	gender := p.Gender
	if age <= 0 && gender == "" {
		return ""
	}
	if gender == "" {
		return fmt.Sprintf("【患者情報】%d歳\n", age)
	}
	if age <= 0 {
		return fmt.Sprintf("【患者情報】%s\n", gender)
	}
	return fmt.Sprintf("【患者情報】%d歳 %s\n", age, gender)
}

// calculateAge は birthDate 時点から referenceDate (空なら今日) までの年齢を返す。
// birthDate / referenceDate は "YYYY-MM-DD"。
// パースエラーや不正な日付の場合は 0 を返す。
func calculateAge(birthDate, referenceDate string) int {
	if birthDate == "" {
		return 0
	}
	bd, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return 0
	}
	var ref time.Time
	if referenceDate == "" {
		ref = time.Now()
	} else {
		ref, err = time.Parse("2006-01-02", referenceDate)
		if err != nil {
			ref = time.Now()
		}
	}
	age := ref.Year() - bd.Year()
	// 誕生日が未来なら 1 歳引く
	if ref.Month() < bd.Month() || (ref.Month() == bd.Month() && ref.Day() < bd.Day()) {
		age--
	}
	if age < 0 {
		return 0
	}
	return age
}
