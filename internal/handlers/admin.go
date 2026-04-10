package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc *services.AdminService
}

func NewAdminHandler(svc *services.AdminService) *AdminHandler {
	return &AdminHandler{svc}
}

func (h *AdminHandler) ServiceStatistics(c *gin.Context) {
	stats, err := h.svc.ServiceStatistics()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats.ToDTO())
}
