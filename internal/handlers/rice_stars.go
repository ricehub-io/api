package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/security"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type RiceStarHandler struct{}

func NewRiceStarHandler() *RiceStarHandler {
	return &RiceStarHandler{}
}

func (h *RiceStarHandler) CreateRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := services.CreateRiceStar(path.RiceID, token.Subject); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *RiceStarHandler) DeleteRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := services.DeleteRiceStar(path.RiceID, token.Subject); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
