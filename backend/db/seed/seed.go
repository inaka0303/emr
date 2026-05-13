package seed

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type attemptSeed struct {
	AttemptID    string
	SubjectID    string
	CaseID       string
	SourceCaseID string
	Intervention string
	Order        int
}

var experimentAttempts = []attemptSeed{
	{"A01", "S1", "C1", "JC-AMI-A", "ai", 1},
	{"A02", "S1", "C2", "JC-PE-B", "control", 2},
	{"A03", "S1", "C3", "JC-AD-A", "ai", 3},
	{"A04", "S1", "C4", "JC-AHF-B", "control", 4},
	{"A05", "S2", "C2", "JC-PE-B", "ai", 1},
	{"A06", "S2", "C3", "JC-AD-A", "control", 2},
	{"A07", "S2", "C4", "JC-AHF-B", "ai", 3},
	{"A08", "S2", "C1", "JC-AMI-A", "control", 4},
	{"A09", "S3", "C3", "JC-AD-A", "control", 1},
	{"A10", "S3", "C4", "JC-AHF-B", "ai", 2},
	{"A11", "S3", "C1", "JC-AMI-A", "control", 3},
	{"A12", "S3", "C2", "JC-PE-B", "ai", 4},
	{"A13", "S4", "C4", "JC-AHF-B", "control", 1},
	{"A14", "S4", "C1", "JC-AMI-A", "ai", 2},
	{"A15", "S4", "C2", "JC-PE-B", "control", 3},
	{"A16", "S4", "C3", "JC-AD-A", "ai", 4},
	{"A17", "S5", "C5", "JC-AMI-B", "ai", 1},
	{"A18", "S5", "C6", "JC-PE-A", "control", 2},
	{"A19", "S5", "C7", "JC-AD-B", "ai", 3},
	{"A20", "S5", "C8", "JC-AHF-A", "control", 4},
	{"A21", "S6", "C6", "JC-PE-A", "ai", 1},
	{"A22", "S6", "C7", "JC-AD-B", "control", 2},
	{"A23", "S6", "C8", "JC-AHF-A", "ai", 3},
	{"A24", "S6", "C5", "JC-AMI-B", "control", 4},
	{"A25", "S7", "C7", "JC-AD-B", "control", 1},
	{"A26", "S7", "C8", "JC-AHF-A", "ai", 2},
	{"A27", "S7", "C5", "JC-AMI-B", "control", 3},
	{"A28", "S7", "C6", "JC-PE-A", "ai", 4},
	{"A29", "S8", "C8", "JC-AHF-A", "control", 1},
	{"A30", "S8", "C5", "JC-AMI-B", "ai", 2},
	{"A31", "S8", "C6", "JC-PE-A", "control", 3},
	{"A32", "S8", "C7", "JC-AD-B", "ai", 4},
}

type outpatientCase struct {
	EncounterID     string                  `json:"encounter_id"`
	Scenario        string                  `json:"scenario"`
	DiseaseLabelJP  string                  `json:"disease_label_jp"`
	Patient         outpatientCasePatient   `json:"patient"`
	Encounter       outpatientCaseEncounter `json:"encounter"`
	ReceptionVitals map[string]interface{}  `json:"reception_vitals"`
	InputPatternA   json.RawMessage         `json:"input_pattern_A"`
	InputPatternB   *string                 `json:"input_pattern_B"`
}

type outpatientCasePatient struct {
	Age                int      `json:"age"`
	Gender             string   `json:"gender"`
	BloodType          *string  `json:"blood_type"`
	Comorbidities      []string `json:"comorbidities"`
	CurrentMedications []string `json:"current_medications"`
	Allergies          []string `json:"allergies"`
	FamilyHistory      []string `json:"family_history"`
	SocialHistory      string   `json:"social_history"`
}

type outpatientCaseEncounter struct {
	Type                string   `json:"type"`
	Department          string   `json:"department"`
	EncounterDate       string   `json:"encounter_date"`
	ChiefComplaint      string   `json:"chief_complaint"`
	SecondaryComplaints []string `json:"secondary_complaints"`
}

type seedPatient struct {
	MRN, Name, NameKana, BirthDate, Gender, BloodType, Phone, Address, ECName, ECPhone string
}

// Run は実験用データをDBに投入する。
// 既存の一般デモ患者は削除し、MRN-0021 (新規太郎) と A01-A32 の独立attemptだけを作る。
func Run(db *sql.DB) error {
	cases, err := loadExperimentCases()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := clearAll(tx); err != nil {
		return err
	}

	if _, err := insertPatient(tx, seedPatient{
		MRN:       "MRN-0021",
		Name:      "新規 太郎",
		NameKana:  "シンキ タロウ",
		BirthDate: "1979-09-09",
		Gender:    "男性",
		BloodType: "O",
		Phone:     "03-9999-0001",
		Address:   "東京都新宿区（仮）21-21-21",
		ECName:    "新規 花子",
		ECPhone:   "03-9999-0002",
	}); err != nil {
		return err
	}

	newPatientID, err := patientIDByMRN(tx, "MRN-0021")
	if err != nil {
		return err
	}
	if _, err := insertEncounter(tx, newPatientID, outpatientCaseEncounter{
		Type:           "外来",
		Department:     "内科",
		EncounterDate:  "2026-04-20",
		ChiefComplaint: "（初診・主訴は問診入力中）",
	}); err != nil {
		return err
	}

	for _, def := range experimentAttempts {
		c, ok := cases[def.SourceCaseID]
		if !ok {
			return fmt.Errorf("case %s not loaded", def.SourceCaseID)
		}
		if err := insertExperimentAttempt(tx, def, c); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func clearAll(tx *sql.Tx) error {
	stmts := []string{
		"DELETE FROM experiment_events",
		"DELETE FROM experiment_attempts",
		"DELETE FROM admission_summaries",
		"DELETE FROM soap_drafts",
		"DELETE FROM interview_notes",
		"DELETE FROM soap_notes",
		"DELETE FROM social_history",
		"DELETE FROM family_history",
		"DELETE FROM medical_history",
		"DELETE FROM encounters",
		"DELETE FROM patients",
		"DELETE FROM sqlite_sequence WHERE name IN ('patients','encounters','interview_notes','soap_notes','medical_history','family_history','social_history','experiment_events')",
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("clear seed data: %w", err)
		}
	}
	return nil
}

func loadExperimentCases() (map[string]outpatientCase, error) {
	dir, err := findCaseDir()
	if err != nil {
		return nil, err
	}
	needed := map[string]bool{}
	for _, a := range experimentAttempts {
		needed[a.SourceCaseID] = true
	}

	out := make(map[string]outpatientCase, len(needed))
	for id := range needed {
		b, err := os.ReadFile(filepath.Join(dir, id+".json"))
		if err != nil {
			return nil, fmt.Errorf("read outpatient case %s: %w", id, err)
		}
		var c outpatientCase
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("parse outpatient case %s: %w", id, err)
		}
		out[id] = c
	}
	return out, nil
}

func findCaseDir() (string, error) {
	if env := strings.TrimSpace(os.Getenv("ACI_CASE_DIR")); env != "" {
		if isDir(env) {
			return env, nil
		}
		return "", fmt.Errorf("ACI_CASE_DIR does not exist: %s", env)
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "../../data/aci_jp_cardio/outpatient/cases"),
		filepath.Join(cwd, "../data/aci_jp_cardio/outpatient/cases"),
		filepath.Join(cwd, "data/aci_jp_cardio/outpatient/cases"),
		"/home/junkanki/naka/data/aci_jp_cardio/outpatient/cases",
	}
	for _, c := range candidates {
		if isDir(c) {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}
	return "", fmt.Errorf("ACI-JP-Cardio outpatient cases directory not found; set ACI_CASE_DIR")
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func insertExperimentAttempt(tx *sql.Tx, def attemptSeed, c outpatientCase) error {
	bloodType := ""
	if c.Patient.BloodType != nil {
		bloodType = *c.Patient.BloodType
	}
	patientID, err := insertPatient(tx, seedPatient{
		MRN:       "EXP-" + def.AttemptID,
		Name:      fmt.Sprintf("実験患者 %s", def.AttemptID),
		NameKana:  fmt.Sprintf("ジッケンカンジャ %s", def.AttemptID),
		BirthDate: birthDateForAge(c.Patient.Age, c.Encounter.EncounterDate),
		Gender:    c.Patient.Gender,
		BloodType: bloodType,
		Phone:     "",
		Address:   "実験症例",
		ECName:    "",
		ECPhone:   "",
	})
	if err != nil {
		return fmt.Errorf("insert experiment patient %s: %w", def.AttemptID, err)
	}

	encounterID, err := insertEncounter(tx, patientID, c.Encounter)
	if err != nil {
		return fmt.Errorf("insert experiment encounter %s: %w", def.AttemptID, err)
	}

	rawText, medList, exam, labs, err := buildInterviewSections(c)
	if err != nil {
		return fmt.Errorf("build interview %s: %w", def.SourceCaseID, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO interview_notes (encounter_id, raw_text, medication_list, exam_findings, lab_results) VALUES (?,?,?,?,?)`,
		encounterID, rawText, medList, exam, labs,
	); err != nil {
		return fmt.Errorf("insert experiment interview %s: %w", def.AttemptID, err)
	}

	if err := insertPatientContext(tx, patientID, c.Patient); err != nil {
		return fmt.Errorf("insert patient context %s: %w", def.AttemptID, err)
	}

	if _, err := tx.Exec(
		`INSERT INTO experiment_attempts
		 (attempt_id, subject_id, case_id, source_case_id, intervention, sequence_order, patient_id, encounter_id)
		 VALUES (?,?,?,?,?,?,?,?)`,
		def.AttemptID, def.SubjectID, def.CaseID, def.SourceCaseID, def.Intervention, def.Order, patientID, encounterID,
	); err != nil {
		return fmt.Errorf("insert experiment attempt %s: %w", def.AttemptID, err)
	}
	return nil
}

func insertPatient(tx *sql.Tx, p seedPatient) (int64, error) {
	result, err := tx.Exec(
		`INSERT INTO patients (mrn, name, name_kana, birth_date, gender, blood_type, phone, address, emergency_contact_name, emergency_contact_phone)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.MRN, p.Name, p.NameKana, p.BirthDate, p.Gender, p.BloodType, p.Phone, p.Address, p.ECName, p.ECPhone,
	)
	if err != nil {
		return 0, fmt.Errorf("insert patient %s: %w", p.MRN, err)
	}
	return result.LastInsertId()
}

func patientIDByMRN(tx *sql.Tx, mrn string) (int64, error) {
	var id int64
	if err := tx.QueryRow(`SELECT id FROM patients WHERE mrn = ?`, mrn).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup patient %s: %w", mrn, err)
	}
	return id, nil
}

func insertEncounter(tx *sql.Tx, patientID int64, e outpatientCaseEncounter) (int64, error) {
	encounterType := normalizeEncounterType(e.Type)
	result, err := tx.Exec(
		`INSERT INTO encounters (patient_id, encounter_date, encounter_type, department, attending_doctor, status, chief_complaint)
		 VALUES (?,?,?,?,?,?,?)`,
		patientID, e.EncounterDate, encounterType, e.Department, "実験担当", "進行中", e.ChiefComplaint,
	)
	if err != nil {
		return 0, fmt.Errorf("insert encounter: %w", err)
	}
	return result.LastInsertId()
}

func normalizeEncounterType(v string) string {
	if strings.Contains(v, "救急") {
		return "救急"
	}
	if strings.Contains(v, "入院") {
		return "入院"
	}
	return "外来"
}

func buildInterviewSections(c outpatientCase) (rawText, medList, exam, labs string, err error) {
	if c.InputPatternB != nil && strings.TrimSpace(*c.InputPatternB) != "" {
		return strings.TrimSpace(*c.InputPatternB), "", "", formatReceptionVitals(c.ReceptionVitals), nil
	}
	if len(c.InputPatternA) == 0 || string(c.InputPatternA) == "null" {
		return "", "", "", "", fmt.Errorf("no usable input pattern")
	}
	var patternA struct {
		PhysicianSummary struct {
			RawText        string `json:"raw_text"`
			MedicationList string `json:"medication_list"`
			ExamFindings   string `json:"exam_findings"`
			LabResults     string `json:"lab_results"`
		} `json:"physician_summary"`
	}
	if err := json.Unmarshal(c.InputPatternA, &patternA); err != nil {
		return "", "", "", "", err
	}
	rawText = strings.TrimSpace(patternA.PhysicianSummary.RawText)
	if rawText == "" {
		return "", "", "", "", fmt.Errorf("pattern A raw_text is empty")
	}
	return rawText,
		trimSectionHeader(patternA.PhysicianSummary.MedicationList, "【お薬手帳より】"),
		strings.TrimSpace(patternA.PhysicianSummary.ExamFindings),
		strings.TrimSpace(patternA.PhysicianSummary.LabResults),
		nil
}

func trimSectionHeader(v, header string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, header)
	return strings.TrimSpace(v)
}

func formatReceptionVitals(v map[string]interface{}) string {
	if len(v) == 0 {
		return ""
	}
	var parts []string
	if sys, ok := v["BP_sys"]; ok {
		if dia, ok := v["BP_dia"]; ok {
			parts = append(parts, fmt.Sprintf("BP %s/%s mmHg", valueString(sys), valueString(dia)))
		}
	}
	keys := []struct {
		Key   string
		Label string
		Unit  string
	}{
		{"HR", "HR", "/min"},
		{"RR", "RR", "/min"},
		{"BT", "BT", "℃"},
		{"SpO2", "SpO2", "%"},
	}
	for _, k := range keys {
		if val, ok := v[k.Key]; ok {
			parts = append(parts, fmt.Sprintf("%s %s%s", k.Label, valueString(val), k.Unit))
		}
	}
	sort.Strings(parts)
	return "受付バイタル: " + strings.Join(parts, ", ")
}

func valueString(v interface{}) string {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', 1, 64)
	case int:
		return strconv.Itoa(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func insertPatientContext(tx *sql.Tx, patientID int64, p outpatientCasePatient) error {
	for _, condition := range p.Comorbidities {
		if strings.TrimSpace(condition) == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO medical_history (patient_id, condition, onset_date, status, notes) VALUES (?,?,?,?,?)`,
			patientID, condition, "", "症例情報", "ACI-JP-Cardio outpatient case JSON",
		); err != nil {
			return err
		}
	}
	if len(p.Allergies) > 0 {
		if _, err := tx.Exec(
			`INSERT INTO medical_history (patient_id, condition, onset_date, status, notes) VALUES (?,?,?,?,?)`,
			patientID, "アレルギー", "", "情報", strings.Join(p.Allergies, "、"),
		); err != nil {
			return err
		}
	}
	for _, item := range p.FamilyHistory {
		relation, condition, notes := parseFamilyHistory(item)
		if _, err := tx.Exec(
			`INSERT INTO family_history (patient_id, relation, condition, notes, is_slm_suggested) VALUES (?,?,?,?,FALSE)`,
			patientID, relation, condition, notes,
		); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.SocialHistory) != "" {
		if _, err := tx.Exec(
			`INSERT INTO social_history (patient_id, category, description, notes, is_slm_suggested) VALUES (?,?,?,?,FALSE)`,
			patientID, "生活歴", p.SocialHistory, "",
		); err != nil {
			return err
		}
	}
	return nil
}

func parseFamilyHistory(item string) (relation, condition, notes string) {
	item = strings.TrimSpace(item)
	if item == "" {
		return "家族歴", "未確認", ""
	}
	idx := strings.IndexAny(item, ":：")
	if idx < 0 {
		return "家族歴", item, ""
	}
	relation = strings.TrimSpace(item[:idx])
	condition = strings.TrimSpace(item[idx+1:])
	if relation == "" {
		relation = "家族歴"
	}
	if condition == "" {
		condition = item
	}
	return relation, condition, item
}

func birthDateForAge(age int, encounterDate string) string {
	t, err := time.Parse("2006-01-02", encounterDate)
	if err != nil {
		return fmt.Sprintf("%04d-01-01", 2026-age)
	}
	return fmt.Sprintf("%04d-01-01", t.Year()-age)
}
