package handler

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/example/ehr-demo/db/seed"
	"github.com/example/ehr-demo/internal/model"
	"github.com/labstack/echo/v4"
)

// SeedHandler はダミーデータ投入用ハンドラ
type SeedHandler struct {
	db *sql.DB
}

// NewSeedHandler は新しいSeedHandlerを生成する
func NewSeedHandler(db *sql.DB) *SeedHandler {
	return &SeedHandler{db: db}
}

// Seed はダミーデータを投入する
func (h *SeedHandler) Seed(c echo.Context) error {
	if err := seed.Run(h.db); err != nil {
		slog.Error("シードデータ投入エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("SEED_ERROR", "ダミーデータの投入に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "seeded"}, nil))
}

// ResetTestPatient はデモ用テスト患者 (MRN-0021, 新規太郎) の
// interview_notes と soap_drafts を全削除する。
// ブラウザリロード時に呼ばれ、毎回ゼロ状態からテストできるようにする。
// POST /api/test-patient/reset
func (h *SeedHandler) ResetTestPatient(c echo.Context) error {
	const targetMRN = "MRN-0021"
	ctx := c.Request().Context()

	// 対象患者のencounter IDを取得
	rows, err := h.db.QueryContext(ctx,
		`SELECT e.id FROM encounters e JOIN patients p ON p.id = e.patient_id WHERE p.mrn = ?`, targetMRN)
	if err != nil {
		slog.Error("テスト患者取得エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("RESET_ERROR", "リセットに失敗しました"))
	}
	defer rows.Close()

	var encounterIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			encounterIDs = append(encounterIDs, id)
		}
	}

	if len(encounterIDs) == 0 {
		return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
			"reset":      true,
			"encounters": 0,
			"note":       "対象患者が見つかりません（MRN-0021）",
		}, nil))
	}

	for _, eid := range encounterIDs {
		if _, err := h.db.ExecContext(ctx, `DELETE FROM interview_notes WHERE encounter_id = ?`, eid); err != nil {
			slog.Error("問診削除エラー", "error", err, "encounter_id", eid)
		}
		if _, err := h.db.ExecContext(ctx, `DELETE FROM soap_drafts WHERE encounter_id = ?`, eid); err != nil {
			slog.Error("SOAPドラフト削除エラー", "error", err, "encounter_id", eid)
		}
		if _, err := h.db.ExecContext(ctx, `DELETE FROM admission_summaries WHERE encounter_id = ?`, eid); err != nil {
			slog.Error("入院時サマリ削除エラー", "error", err, "encounter_id", eid)
		}
	}

	slog.Info("テスト患者リセット完了", "mrn", targetMRN, "encounters", len(encounterIDs))
	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]interface{}{
		"reset":      true,
		"encounters": len(encounterIDs),
		"mrn":        targetMRN,
	}, nil))
}
