package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct{}

func NewLeaderboardHandler() *LeaderboardHandler {
	return &LeaderboardHandler{}
}

func (h *LeaderboardHandler) GetWeeklyLeaderboard(c *gin.Context) {
	rices, err := h.fetchLeaderboard(c, models.Week)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}

func (h *LeaderboardHandler) GetMonthlyLeaderboard(c *gin.Context) {
	rices, err := h.fetchLeaderboard(c, models.Month)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}

func (h *LeaderboardHandler) GetYearlyLeaderboard(c *gin.Context) {
	rices, err := h.fetchLeaderboard(c, models.Year)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}

func (h *LeaderboardHandler) fetchLeaderboard(c *gin.Context, period models.LeaderboardPeriod) ([]models.LeaderboardRiceDTO, errs.AppError) {
	lead, err := services.FetchLeaderboard(period, GetUserIDFromRequest(c))
	if err != nil {
		return nil, err
	}
	return lead.ToDTO(), nil
}
