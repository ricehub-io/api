package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
)

func fetchLeaderboard(c *gin.Context, period models.LeaderboardPeriod) ([]models.LeaderboardRiceDTO, *errs.AppError) {
	userID := GetUserIDFromRequest(c)

	rices, err := repository.FetchLeaderboard(period, userID)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return rices.ToDTO(), nil
}

func GetWeeklyLeaderboard(c *gin.Context) {
	rices, err := fetchLeaderboard(c, models.Week)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}

func GetMonthlyLeaderboard(c *gin.Context) {
	rices, err := fetchLeaderboard(c, models.Month)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}

func GetYearlyLeaderboard(c *gin.Context) {
	rices, err := fetchLeaderboard(c, models.Year)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices)
}
