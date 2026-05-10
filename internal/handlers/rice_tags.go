package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RiceTagHandler struct {
	svc *services.RiceTagService
}

func NewRiceTagHandler(svc *services.RiceTagService) *RiceTagHandler {
	return &RiceTagHandler{svc}
}

// @Summary Attach tags to a rice
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.AttachTagsDTO true "Tag IDs to attach"
// @Success 200 "Tags attached"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices/{id}/tags [post]
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

// @Summary Remove tags from a rice
// @Tags rices
// @Accept json
// @Param id path string true "Rice ID (UUID)"
// @Param body body models.UnattachTagsDTO true "Tag IDs to remove"
// @Success 204 "Tags removed"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /rices/{id}/tags [delete]
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
