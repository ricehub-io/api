package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type WebVarHandler struct {
	svc *services.WebVarService
}

func NewWebVarHandler(svc *services.WebVarService) *WebVarHandler {
	return &WebVarHandler{svc}
}

func (h *WebVarHandler) GetWebVarByKey(c *gin.Context) {
	key := c.Param("key")

	v, err := h.svc.GetWebVarByKey(key)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
