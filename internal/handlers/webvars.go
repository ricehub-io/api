package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/services"

	"github.com/gin-gonic/gin"
)

type WebVarHandler struct {
	svc *services.WebVarService
}

func NewWebVarHandler(svc *services.WebVarService) *WebVarHandler {
	return &WebVarHandler{svc}
}

// @Summary Get a website variable by key
// @Tags vars
// @Produce json
// @Param key path string true "Variable key"
// @Success 200 {object} models.WebsiteVariableDTO
// @Failure 404 {object} models.ErrorDTO "Variable not found"
// @Router /vars/{key} [get]
func (h *WebVarHandler) GetWebVarByKey(c *gin.Context) {
	key := c.Param("key")

	v, err := h.svc.GetWebVarByKey(c.Request.Context(), key)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
