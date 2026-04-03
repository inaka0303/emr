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
