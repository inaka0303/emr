package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/example/ehr-demo/db/migrations"
	"github.com/example/ehr-demo/internal/handler"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/example/ehr-demo/internal/service"
	"github.com/example/ehr-demo/internal/slm"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "modernc.org/sqlite"
)

func main() {
	// ログ設定
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// DB初期化
	db, err := sql.Open("sqlite", "./ehr-demo.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		slog.Error("DB接続エラー", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// マイグレーション実行
	if err := runMigrations(db); err != nil {
		slog.Error("マイグレーションエラー", "error", err)
		os.Exit(1)
	}
	slog.Info("マイグレーション完了")

	// レイヤー初期化
	patientRepo := repository.NewPatientRepository(db)
	patientSvc := service.NewPatientService(patientRepo)
	patientHandler := handler.NewPatientHandler(patientSvc)

	encounterRepo := repository.NewEncounterRepository(db)
	encounterSvc := service.NewEncounterService(encounterRepo)
	encounterHandler := handler.NewEncounterHandler(encounterSvc)

	soapRepo := repository.NewSOAPRepository(db)
	soapSvc := service.NewSOAPService(soapRepo)
	soapHandler := handler.NewSOAPHandler(soapSvc)

	medicalHistoryRepo := repository.NewMedicalHistoryRepository(db)
	medicalHistorySvc := service.NewMedicalHistoryService(medicalHistoryRepo)
	medicalHistoryHandler := handler.NewMedicalHistoryHandler(medicalHistorySvc)

	familyHistoryRepo := repository.NewFamilyHistoryRepository(db)
	familyHistorySvc := service.NewFamilyHistoryService(familyHistoryRepo)
	familyHistoryHandler := handler.NewFamilyHistoryHandler(familyHistorySvc)

	socialHistoryRepo := repository.NewSocialHistoryRepository(db)
	socialHistorySvc := service.NewSocialHistoryService(socialHistoryRepo)
	socialHistoryHandler := handler.NewSocialHistoryHandler(socialHistorySvc)

	interviewRepo := repository.NewInterviewRepository(db)
	interviewSvc := service.NewInterviewService(interviewRepo)
	interviewHandler := handler.NewInterviewHandler(interviewSvc)

	seedHandler := handler.NewSeedHandler(db)
	experimentRepo := repository.NewExperimentRepository(db)
	experimentHandler := handler.NewExperimentHandler(experimentRepo)

	// SLMクライアント初期化
	slmAPIURL := os.Getenv("SLM_API_URL")
	if slmAPIURL == "" {
		slmAPIURL = "http://localhost:8000"
	}
	slmClient := slm.NewClient(slmAPIURL)

	// admission 専用 SLM サーバー（典型: 9B）を環境変数から読み込む
	// 設定されていれば GenerateAdmissionSummary はこちらへルーティング、未設定なら 4B fallback
	if admURL := os.Getenv("SLM_ADMISSION_URL"); admURL != "" {
		slmClient.SetAdmissionServer(admURL)
	}

	// RAG 自動注入用 URL。設定されていれば GenerateSOAP / GenerateAdmissionSummary 時に
	// interview_text を query として RAG を叩き、上位 3 件の snippet を system prompt に
	// 自動注入する。デフォルト: http://localhost:8082 (RAG_API_URL env でオーバーライド可)。
	// 無効化したい場合は RAG_API_URL= (空文字) を設定する。
	ragURL := os.Getenv("RAG_API_URL")
	if _, explicit := os.LookupEnv("RAG_API_URL"); !explicit {
		ragURL = "http://localhost:8082"
	}
	if ragURL != "" {
		slmClient.SetRAGClient(ragURL)
	}

	slmHandler := handler.NewSLMHandler(slmClient, encounterSvc, patientSvc)
	slmHandler.SetExperimentRepository(experimentRepo)

	// PatientHistoryService (過去受診サマリ、cross-encounter 要約)
	// ENABLE_CROSS_ENCOUNTER_SUMMARY=false で明示的に無効化可能、
	// デフォルトは有効 (nil なら handler 側で no-op になるので常に渡して安全)。
	var historySvc *service.PatientHistoryService
	if os.Getenv("ENABLE_CROSS_ENCOUNTER_SUMMARY") != "false" {
		historySvc = service.NewPatientHistoryService(encounterSvc, soapSvc, interviewSvc)
		slog.Info("cross-encounter 要約 有効化")
	}

	// SOAPドラフト（DBキャッシュ付き）
	soapDraftRepo := repository.NewSOAPDraftRepository(db)
	soapDraftHandler := handler.NewSOAPDraftHandler(slmClient, soapDraftRepo, interviewSvc, encounterSvc, patientSvc, historySvc)
	soapDraftHandler.SetExperimentRepository(experimentRepo)

	// Echo初期化
	e := echo.New()
	e.HideBanner = true

	// ミドルウェア
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAccept, "X-Experiment-Attempt"},
	}))

	// ルーティング
	api := e.Group("/api")
	patientHandler.RegisterRoutes(api.Group("/patients"))

	// Encounter ルート
	api.GET("/patients/:id/encounters", encounterHandler.ListByPatient)
	api.POST("/patients/:id/encounters", encounterHandler.Create)
	api.GET("/encounters/:id", encounterHandler.Get)

	// SOAP ルート
	api.GET("/encounters/:id/soap", soapHandler.GetByEncounter)
	api.POST("/encounters/:id/soap", soapHandler.Create)
	api.PUT("/soap/:id", soapHandler.Update)

	// Medical History ルート
	api.GET("/patients/:id/medical-history", medicalHistoryHandler.ListByPatient)
	api.POST("/patients/:id/medical-history", medicalHistoryHandler.Create)
	api.PUT("/medical-history/:id", medicalHistoryHandler.Update)
	api.DELETE("/medical-history/:id", medicalHistoryHandler.Delete)

	// Family History ルート
	api.GET("/patients/:id/family-history", familyHistoryHandler.ListByPatient)
	api.POST("/patients/:id/family-history", familyHistoryHandler.Create)
	api.PUT("/family-history/:id", familyHistoryHandler.Update)
	api.DELETE("/family-history/:id", familyHistoryHandler.Delete)

	// Social History ルート
	api.GET("/patients/:id/social-history", socialHistoryHandler.ListByPatient)
	api.POST("/patients/:id/social-history", socialHistoryHandler.Create)
	api.PUT("/social-history/:id", socialHistoryHandler.Update)
	api.DELETE("/social-history/:id", socialHistoryHandler.Delete)

	// Interview ルート
	api.GET("/encounters/:id/interviews", interviewHandler.ListByEncounter)
	api.POST("/encounters/:id/interviews", interviewHandler.Create)

	// SOAPドラフト（DBキャッシュ付き、SLM生成）
	api.POST("/encounters/:id/soap-draft", soapDraftHandler.GetOrGenerate)
	api.POST("/encounters/:id/soap-draft/stream", soapDraftHandler.StreamGenerate)
	api.DELETE("/encounters/:id/soap-draft", soapDraftHandler.Invalidate)

	// 入院時サマリ（編集後の永続化）
	admissionSummaryRepo := repository.NewAdmissionSummaryRepository(db)
	admissionSummaryHandler := handler.NewAdmissionSummaryHandler(admissionSummaryRepo)
	api.GET("/encounters/:id/admission-summary", admissionSummaryHandler.Get)
	api.POST("/encounters/:id/admission-summary", admissionSummaryHandler.Save)

	// RAG 検索（Python rag_server.py への proxy）
	ragHandler := handler.NewRAGHandler()
	ragHandler.SetExperimentRepository(experimentRepo)
	api.POST("/rag/search", ragHandler.Search)
	api.GET("/rag/health", ragHandler.Health)

	// 実験 attempt / イベントログ
	experimentHandler.RegisterRoutes(api.Group("/experiment"))

	// SLM ルート
	api.POST("/slm/suggest/soap", slmHandler.SuggestSOAP)
	api.POST("/slm/suggest/admission", slmHandler.SuggestAdmissionSummary)
	api.POST("/slm/suggest/summary", slmHandler.SuggestSummary)
	api.POST("/slm/autocomplete", slmHandler.Autocomplete)
	api.GET("/slm/health", slmHandler.Health)

	// シードエンドポイント（本番環境では無効）
	if os.Getenv("APP_ENV") != "production" {
		api.GET("/seed", seedHandler.Seed)
		api.POST("/test-patient/reset", seedHandler.ResetTestPatient)
	}

	// ヘルスチェック
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// サーバー起動
	slog.Info("サーバー起動", "port", 8080)
	if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
		slog.Error("サーバーエラー", "error", err)
		os.Exit(1)
	}
}

func runMigrations(db *sql.DB) error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".sql") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		sqlBytes, err := migrations.FS.ReadFile(n)
		if err != nil {
			return fmt.Errorf("read %s: %w", n, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			// SQLiteの ALTER TABLE ADD COLUMN は IF NOT EXISTS 非対応のため、
			// 既適用マイグレーションでのduplicate column エラーは冪等性として許容する
			msg := err.Error()
			if strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists") {
				slog.Info("マイグレーションスキップ（既適用）", "file", n)
				continue
			}
			return fmt.Errorf("exec %s: %w", n, err)
		}
		slog.Info("マイグレーション適用", "file", n)
	}
	return nil
}
