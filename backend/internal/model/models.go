package model

import "encoding/json"

// Patient は患者情報を表す構造体
type Patient struct {
	ID                    int64  `json:"id"`
	MRN                   string `json:"mrn"`
	Name                  string `json:"name"`
	NameKana              string `json:"name_kana"`
	BirthDate             string `json:"birth_date"`
	Gender                string `json:"gender"`
	BloodType             string `json:"blood_type"`
	Phone                 string `json:"phone"`
	Address               string `json:"address"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// Encounter は受診情報を表す構造体
type Encounter struct {
	ID              int64  `json:"id"`
	PatientID       int64  `json:"patient_id"`
	EncounterDate   string `json:"encounter_date"`
	EncounterType   string `json:"encounter_type"`
	Department      string `json:"department"`
	AttendingDoctor string `json:"attending_doctor"`
	Status          string `json:"status"`
	ChiefComplaint  string `json:"chief_complaint"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// SOAPNote はSOAP形式の記録を表す構造体
type SOAPNote struct {
	ID              int64  `json:"id"`
	EncounterID     int64  `json:"encounter_id"`
	Author          string `json:"author"`
	Subjective      string `json:"subjective"`
	Objective       string `json:"objective"`
	Assessment      string `json:"assessment"`
	Plan            string `json:"plan"`
	IsSLMSuggested  bool   `json:"is_slm_suggested"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// MedicalHistory は既往歴を表す構造体
type MedicalHistory struct {
	ID        int64  `json:"id"`
	PatientID int64  `json:"patient_id"`
	Condition string `json:"condition"`
	OnsetDate string `json:"onset_date"`
	Status    string `json:"status"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FamilyHistory は家族歴を表す構造体
type FamilyHistory struct {
	ID             int64  `json:"id"`
	PatientID      int64  `json:"patient_id"`
	Relation       string `json:"relation"`
	Condition      string `json:"condition"`
	Notes          string `json:"notes"`
	IsSLMSuggested bool   `json:"is_slm_suggested"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// SocialHistory は社会歴を表す構造体
type SocialHistory struct {
	ID             int64  `json:"id"`
	PatientID      int64  `json:"patient_id"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Notes          string `json:"notes"`
	IsSLMSuggested bool   `json:"is_slm_suggested"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// InterviewNote は問診記録（4セクション構造）を表す構造体
//   RawText       : 問診記録（患者から聞く。主訴・現病歴・既往・家族歴・社会歴・アレルギー）
//   MedicationList: お薬手帳（持参薬の正確な名前・用量）
//   ExamFindings  : 診察所見（医師が視触聴診で取る客観的身体所見）
//   LabResults    : 検査結果（バイタル含む、採血・画像・心電図など）
type InterviewNote struct {
	ID             int64           `json:"id"`
	EncounterID    int64           `json:"encounter_id"`
	RawText        string          `json:"raw_text"`
	MedicationList string          `json:"medication_list"`
	ExamFindings   string          `json:"exam_findings"`
	LabResults     string          `json:"lab_results"`
	StructuredData json.RawMessage `json:"structured_data"`
	CreatedAt      string          `json:"created_at"`
}
