package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceTagHandler struct {
	svc *services.RiceTagService
}

func NewRiceTagHandler(svc *services.RiceTagService) *RiceTagHandler {
	return &RiceTagHandler{svc}
}

func (h *RiceTagHandler) AddRiceTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	var body models.AttachTagsDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.AddRiceTags(c.Request.Context(), riceID, userID, token.IsAdmin, body.Tags); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *RiceTagHandler) RemoveRiceTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidRiceID)
		return
	}
	riceID, _ := uuid.Parse(path.RiceID)

	var body models.UnattachTagsDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.RemoveRiceTags(c.Request.Context(), riceID, userID, token.IsAdmin, body.Tags); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
