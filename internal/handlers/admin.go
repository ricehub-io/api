package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc *services.AdminService
}

func NewAdminHandler(svc *services.AdminService) *AdminHandler {
	return &AdminHandler{svc}
}

// @Summary Get service-wide statistics (admin only)
// @Tags admin
// @Produce json
// @Success 200 {object} models.ServiceStatisticsDTO
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Security BearerAuth
// @Router /admin/stats [get]
func (h *AdminHandler) ServiceStatistics(c *gin.Context) {
	stats, err := h.svc.ServiceStatistics(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stats.ToDTO())
}
