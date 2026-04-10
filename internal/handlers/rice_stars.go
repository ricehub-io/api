package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/security"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type RiceStarHandler struct {
	svc *services.RiceStarService
}

func NewRiceStarHandler(svc *services.RiceStarService) *RiceStarHandler {
	return &RiceStarHandler{svc}
}

func (h *RiceStarHandler) CreateRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}

	if err := h.svc.CreateRiceStar(path.RiceID, token.Subject); err != nil {
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

	if err := h.svc.DeleteRiceStar(path.RiceID, token.Subject); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
