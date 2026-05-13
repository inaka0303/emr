package seed

import (
	"database/sql"
	"sort"
	"strings"
	"testing"

	"github.com/example/ehr-demo/db/migrations"

	_ "modernc.org/sqlite"
)

func TestRunSeedsExperimentAttempts(t *testing.T) {
	caseDir := "../../../../data/aci_jp_cardio/outpatient/cases"
	if !isDir(caseDir) {
		t.Skip("ACI-JP-Cardio outpatient cases are not available")
	}
	t.Setenv("ACI_CASE_DIR", caseDir)

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := runTestMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := Run(db); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	assertCount(t, db, "patients", 33)
	assertCount(t, db, "encounters", 33)
	assertCount(t, db, "interview_notes", 32)
	assertCount(t, db, "experiment_attempts", 32)
	assertWhereCount(t, db, "experiment_attempts", "intervention = 'ai'", 16)
	assertWhereCount(t, db, "experiment_attempts", "intervention = 'control'", 16)
	assertWhereCount(t, db, "patients", "mrn = 'MRN-0021'", 1)
	assertWhereCount(t, db, "patients", "mrn = 'MRN-0001'", 0)
	assertWhereCount(t, db, "experiment_attempts", "attempt_id = 'A01' AND subject_id = 'S1' AND case_id = 'C1' AND intervention = 'ai'", 1)
}

func runTestMigrations(db *sql.DB) error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		b, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(b)); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") || strings.Contains(err.Error(), "already exists") {
				continue
			}
			return err
		}
	}
	return nil
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	assertWhereCount(t, db, table, "1 = 1", want)
}

func assertWhereCount(t *testing.T, db *sql.DB, table, where string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + where).Scan(&got); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	if got != want {
		t.Fatalf("count %s where %s = %d, want %d", table, where, got, want)
	}
}
