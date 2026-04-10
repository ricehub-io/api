package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) ServiceStatistics(c *gin.Context) {
	stats, err := services.ServiceStatistics()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats.ToDTO())
}
