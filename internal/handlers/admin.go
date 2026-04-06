package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

func ServiceStatistics(c *gin.Context) {
	stats, err := services.ServiceStatistics()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats.ToDTO())
}
