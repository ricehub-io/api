package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type WebVarHandler struct{}

func NewWebVarHandler() *WebVarHandler {
	return &WebVarHandler{}
}

func (h *WebVarHandler) GetWebVarByKey(c *gin.Context) {
	key := c.Param("key")

	v, err := services.GetWebVarByKey(key)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
