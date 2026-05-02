package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceStarHandler struct {
	svc *services.RiceStarService
}

func NewRiceStarHandler(svc *services.RiceStarService) *RiceStarHandler {
	return &RiceStarHandler{svc}
}

func (h *RiceStarHandler) CreateRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	if err := h.svc.CreateRiceStar(c.Request.Context(), riceID, userID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *RiceStarHandler) DeleteRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	if err := h.svc.DeleteRiceStar(c.Request.Context(), riceID, userID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
