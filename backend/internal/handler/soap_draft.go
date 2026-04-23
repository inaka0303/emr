package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/example/ehr-demo/internal/service"
	"github.com/example/ehr-demo/internal/slm"
	"github.com/labstack/echo/v4"
)

// buildHybridInput は 4セクション (raw_text / medication_list / exam_findings / lab_results) を
// 訓練データと同じ hybrid format の 1つの user 入力に結合する。
// 空のセクションは省略する。
func buildHybridInput(n *model.InterviewNote) string {
	var sb strings.Builder
	addSection := func(header, body string) {
		body = strings.TrimSpace(body)
		if body == "" {
			return
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(header)
		sb.WriteString("\n")
		sb.WriteString(body)
	}
	addSection("【問診記録】", n.RawText)
	addSection("【お薬手帳より】", n.MedicationList)
	addSection("【診察所見メモ】", n.ExamFindings)
	addSection("【検査結果】", n.LabResults)
	return sb.String()
}

// SOAPDraftHandler はSOAPドラフト（キャッシュ付き）のHTTPハンドラ
type SOAPDraftHandler struct {
	client       *slm.Client
	draftRepo    *repository.SOAPDraftRepository
	interviewSvc *service.InterviewService
	encounterSvc *service.EncounterService
	patientSvc   *service.PatientService
}

func NewSOAPDraftHandler(
	client *slm.Client,
	draftRepo *repository.SOAPDraftRepository,
	interviewSvc *service.InterviewService,
	encounterSvc *service.EncounterService,
	patientSvc *service.PatientService,
) *SOAPDraftHandler {
	return &SOAPDraftHandler{
		client:       client,
		draftRepo:    draftRepo,
		interviewSvc: interviewSvc,
		encounterSvc: encounterSvc,
		patientSvc:   patientSvc,
	}
}

// lookupPatientHeader は encounter_id から患者を解決して「【患者情報】N歳 性別」ヘッダを返す。
// 失敗時は空文字（注入スキップ）。
func (h *SOAPDraftHandler) lookupPatientHeader(ctx context.Context, encounterID int64) string {
	enc, err := h.encounterSvc.GetByID(ctx, encounterID)
	if err != nil || enc == nil {
		return ""
	}
	p, err := h.patientSvc.GetPatient(ctx, enc.PatientID)
	if err != nil || p == nil {
		return ""
	}
	return buildPatientHeader(p, enc.EncounterDate)
}

type soapDraftRequest struct {
	Force bool `json:"force"`
}

type soapDraftMeta struct {
	Model     string `json:"model"`
	IsMock    bool   `json:"is_mock"`
	LatencyMs int64  `json:"latency_ms"`
	Cached    bool   `json:"cached"`
}

type soapDraftResponse struct {
	Data slm.SOAPSuggestion `json:"data"`
	Meta soapDraftMeta      `json:"meta"`
}

// GetOrGenerate は encounter_id に対するSOAPドラフトを返す
// 初回はSLMで生成してDBにキャッシュ、以降は即座に返す
// body に {"force": true} を渡すと再生成
// POST /api/encounters/:id/soap-draft
func (h *SOAPDraftHandler) GetOrGenerate(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "無効な受診IDです"})
	}

	var req soapDraftRequest
	// body が空の可能性もあるのでバインドエラーは無視
	_ = c.Bind(&req)

	ctx := c.Request().Context()

	// キャッシュを確認
	if !req.Force {
		if d, err := h.draftRepo.GetByEncounterID(ctx, encounterID); err == nil {
			return c.JSON(http.StatusOK, soapDraftResponse{
				Data: slm.SOAPSuggestion{
					Subjective: d.Subjective,
					Objective:  d.Objective,
					Assessment: d.Assessment,
					Plan:       d.Plan,
				},
				Meta: soapDraftMeta{
					Model:     d.Model,
					IsMock:    false,
					LatencyMs: d.GenerationMS,
					Cached:    true,
				},
			})
		} else if !errors.Is(err, repository.ErrSOAPDraftNotFound) {
			slog.Error("SOAPドラフトキャッシュ取得エラー", "error", err)
		}
	}

	// キャッシュなし or force=true → SLM で生成
	notes, err := h.interviewSvc.ListByEncounterID(ctx, encounterID)
	if err != nil {
		slog.Error("問診取得エラー", "error", err, "encounter_id", encounterID)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "問診データの取得に失敗しました"})
	}
	if len(notes) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "この受診には問診データがありません。問診を入力してから再試行してください。"})
	}

	// ListByEncounterID は created_at DESC で返るので、先頭が最新。
	// 4セクション（問診/お薬/所見/検査）を訓練データと同じhybrid formatに結合。
	// 空のセクションは省略する（訓練データにもセクションが全部揃ってるとは限らないため）。
	interviewText := buildHybridInput(&notes[0])

	// 患者属性（年齢・性別）を冒頭に注入: SLM が性別を仮定して誤記載する問題を防ぐ
	if header := h.lookupPatientHeader(ctx, encounterID); header != "" {
		interviewText = header + "\n" + interviewText
	}

	suggestion, isMock, latency, err := h.client.GenerateSOAP(ctx, interviewText)
	if err != nil {
		slog.Error("SOAP生成エラー", "error", err, "encounter_id", encounterID)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "SOAPドラフトの生成に失敗しました"})
	}

	// 成功かつモックでない場合のみ永続キャッシュへ
	if !isMock && suggestion != nil {
		if err := h.draftRepo.Upsert(ctx, &repository.SOAPDraft{
			EncounterID:  encounterID,
			Subjective:   suggestion.Subjective,
			Objective:    suggestion.Objective,
			Assessment:   suggestion.Assessment,
			Plan:         suggestion.Plan,
			Model:        h.client.ModelName(),
			GenerationMS: latency.Milliseconds(),
		}); err != nil {
			slog.Warn("SOAPドラフトキャッシュ保存失敗（続行）", "error", err)
		}
	}

	return c.JSON(http.StatusOK, soapDraftResponse{
		Data: *suggestion,
		Meta: soapDraftMeta{
			Model:     h.client.ModelName(),
			IsMock:    isMock,
			LatencyMs: latency.Milliseconds(),
			Cached:    false,
		},
	})
}

// StreamGenerate は SOAP ドラフトをセクション逐次 (S→O→A→P) に SSE で返す
// 各セクション完了ごとに "event: section" で送出し、全完了時に "event: done" を送る。
// キャッシュヒット時は瞬時に全セクションを流し出して完了する。
// POST /api/encounters/:id/soap-draft/stream   body: {"force": bool}
func (h *SOAPDraftHandler) StreamGenerate(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "無効な受診IDです"})
	}

	var req soapDraftRequest
	_ = c.Bind(&req)

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
	}

	sendEvent := func(event string, data interface{}) {
		b, _ := json.Marshal(data)
		_, _ = fmt.Fprintf(w.Writer, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}

	ctx := c.Request().Context()

	// キャッシュヒット: 4セクションを即時に流し出して完了
	if !req.Force {
		if d, err := h.draftRepo.GetByEncounterID(ctx, encounterID); err == nil {
			for _, s := range []struct {
				sec, text string
			}{
				{"S", d.Subjective},
				{"O", d.Objective},
				{"A", d.Assessment},
				{"P", d.Plan},
			} {
				sendEvent("section", map[string]string{"section": s.sec, "text": s.text})
			}
			sendEvent("done", map[string]interface{}{"cached": true, "latency_ms": d.GenerationMS, "model": d.Model})
			return nil
		}
	}

	// キャッシュミス: SLM でセクション逐次生成
	notes, err := h.interviewSvc.ListByEncounterID(ctx, encounterID)
	if err != nil {
		sendEvent("error", map[string]string{"message": "問診取得エラー"})
		return nil
	}
	if len(notes) == 0 {
		sendEvent("error", map[string]string{"message": "この受診には問診データがありません。問診を入力してから再試行してください。"})
		return nil
	}
	interviewText := buildHybridInput(&notes[0])
	if interviewText == "" {
		sendEvent("error", map[string]string{"message": "問診/お薬/診察所見/検査結果のいずれにも内容がありません。"})
		return nil
	}

	// 患者属性注入（streaming 版でも同じ）
	if header := h.lookupPatientHeader(ctx, encounterID); header != "" {
		interviewText = header + "\n" + interviewText
	}

	suggestion, isMock, latency, genErr := h.client.GenerateSOAPStreaming(ctx, interviewText, func(section, text string) {
		sendEvent("section", map[string]string{"section": section, "text": text})
	})
	if genErr != nil {
		sendEvent("error", map[string]string{"message": "SOAP生成に失敗しました"})
		return nil
	}

	// DBキャッシュに保存（モックでない場合）
	if !isMock && suggestion != nil {
		_ = h.draftRepo.Upsert(ctx, &repository.SOAPDraft{
			EncounterID:  encounterID,
			Subjective:   suggestion.Subjective,
			Objective:    suggestion.Objective,
			Assessment:   suggestion.Assessment,
			Plan:         suggestion.Plan,
			Model:        h.client.ModelName(),
			GenerationMS: latency.Milliseconds(),
		})
	}

	sendEvent("done", map[string]interface{}{
		"cached":     false,
		"latency_ms": latency.Milliseconds(),
		"model":      h.client.ModelName(),
		"is_mock":    isMock,
	})
	return nil
}

// Invalidate はSOAPドラフトキャッシュを削除する
// DELETE /api/encounters/:id/soap-draft
func (h *SOAPDraftHandler) Invalidate(c echo.Context) error {
	encounterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "無効な受診IDです"})
	}
	if err := h.draftRepo.Delete(c.Request().Context(), encounterID); err != nil {
		slog.Error("SOAPドラフトキャッシュ削除エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "キャッシュ削除に失敗しました"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
