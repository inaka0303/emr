package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/example/ehr-demo/internal/service"
	"github.com/labstack/echo/v4"
)

// PatientHandler は患者APIのHTTPハンドラ
type PatientHandler struct {
	svc *service.PatientService
}

// NewPatientHandler は新しいPatientHandlerを生成する
func NewPatientHandler(svc *service.PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// RegisterRoutes はルーティングを登録する
func (h *PatientHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
}

// List は患者一覧を返す
func (h *PatientHandler) List(c echo.Context) error {
	query := c.QueryParam("q")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	patients, total, err := h.svc.ListPatients(c.Request().Context(), query, page, limit)
	if err != nil {
		slog.Error("患者一覧取得エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "患者一覧の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(patients, &model.Meta{
		Total: total,
		Page:  page,
		Limit: limit,
	}))
}

// Get は指定IDの患者を返す
func (h *PatientHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	patient, err := h.svc.GetPatient(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "患者が見つかりません"))
		}
		slog.Error("患者取得エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "患者情報の取得に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(patient, nil))
}

// Create は新しい患者を登録する
func (h *PatientHandler) Create(c echo.Context) error {
	var p model.Patient
	if err := c.Bind(&p); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if p.MRN == "" || p.Name == "" || p.NameKana == "" || p.BirthDate == "" || p.Gender == "" {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("VALIDATION_ERROR", "必須フィールド（mrn, name, name_kana, birth_date, gender）が不足しています"))
	}

	id, err := h.svc.CreatePatient(c.Request().Context(), &p)
	if err != nil {
		slog.Error("患者登録エラー", "error", err)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "患者の登録に失敗しました"))
	}

	p.ID = id
	return c.JSON(http.StatusCreated, model.NewSuccessResponse(p, nil))
}

// Update は患者情報を更新する
func (h *PatientHandler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_ID", "無効な患者IDです"))
	}

	var p model.Patient
	if err := c.Bind(&p); err != nil {
		return c.JSON(http.StatusBadRequest, model.NewErrorResponse("INVALID_BODY", "リクエストボディが不正です"))
	}

	if err := h.svc.UpdatePatient(c.Request().Context(), id, &p); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, model.NewErrorResponse("NOT_FOUND", "患者が見つかりません"))
		}
		slog.Error("患者更新エラー", "error", err, "id", id)
		return c.JSON(http.StatusInternalServerError, model.NewErrorResponse("INTERNAL_ERROR", "患者情報の更新に失敗しました"))
	}

	return c.JSON(http.StatusOK, model.NewSuccessResponse(map[string]string{"status": "updated"}, nil))
}
