package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/example/ehr-demo/internal/model"
	"github.com/example/ehr-demo/internal/repository"
	"github.com/labstack/echo/v4"
)

const experimentAttemptHeader = "X-Experiment-Attempt"
const experimentWarmupHeader = "X-Experiment-Warmup"

func getExperimentAttemptID(c echo.Context) string {
	if v := strings.TrimSpace(c.Request().Header.Get(experimentAttemptHeader)); v != "" {
		return strings.ToUpper(v)
	}
	if v := strings.TrimSpace(c.QueryParam("attempt_id")); v != "" {
		return strings.ToUpper(v)
	}
	return ""
}

func isExperimentWarmup(c echo.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.Request().Header.Get(experimentWarmupHeader)))
	return v == "1" || v == "true" || v == "yes"
}

func ensureExperimentAIAllowed(
	c echo.Context,
	repo *repository.ExperimentRepository,
	encounterID int64,
) (*model.ExperimentAttempt, error) {
	if repo == nil {
		return nil, nil
	}

	ctx := c.Request().Context()
	attemptID := getExperimentAttemptID(c)
	var attempt *model.ExperimentAttempt
	var err error

	if attemptID != "" {
		attempt, err = repo.GetAttempt(ctx, attemptID)
		if err != nil {
			return nil, err
		}
		if encounterID > 0 && attempt.EncounterID != encounterID {
			return attempt, fmt.Errorf("attempt %s does not own encounter %d", attempt.AttemptID, encounterID)
		}
	} else if encounterID > 0 {
		attempt, err = repo.GetAttemptByEncounter(ctx, encounterID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	if attempt.Intervention != "ai" {
		return attempt, errExperimentAIBlocked
	}
	if attempt.Status == "finished" {
		return attempt, errExperimentFinished
	}
	return attempt, nil
}

var errExperimentAIBlocked = errors.New("experiment ai disabled")
var errExperimentFinished = errors.New("experiment attempt finished")

func experimentGuardResponse(c echo.Context, err error) error {
	if errors.Is(err, errExperimentAIBlocked) {
		return c.JSON(http.StatusForbidden, model.NewErrorResponse("AI_DISABLED", "このattemptはControl条件のためAI機能は使用できません"))
	}
	if errors.Is(err, errExperimentFinished) {
		return c.JSON(http.StatusConflict, model.NewErrorResponse("ATTEMPT_FINISHED", "このattemptは終了済みのためAI機能は使用できません"))
	}
	if errors.Is(err, repository.ErrNotFound) {
		return c.JSON(http.StatusNotFound, model.NewErrorResponse("EXPERIMENT_ATTEMPT_NOT_FOUND", "指定された実験attemptが見つかりません"))
	}
	return c.JSON(http.StatusBadRequest, model.NewErrorResponse("EXPERIMENT_GUARD_ERROR", err.Error()))
}
