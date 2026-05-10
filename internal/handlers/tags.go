package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	svc *services.TagService
}

func NewTagHandler(svc *services.TagService) *TagHandler {
	return &TagHandler{svc}
}

type tagsPath struct {
	TagID int `uri:"id" binding:"required,gt=0"`
}

// @Summary Create a new tag (admin only)
// @Tags tags
// @Accept json
// @Produce json
// @Param body body models.TagNameDTO true "Tag name"
// @Success 201 {object} models.TagDTO
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 409 {object} models.ErrorDTO "Tag already exists"
// @Security BearerAuth
// @Router /tags [post]
func (h *TagHandler) CreateTag(c *gin.Context) {
	var body models.TagNameDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	tag, err := h.svc.CreateTag(c.Request.Context(), body.Name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, tag.ToDTO())
}

// @Summary List all tags
// @Tags tags
// @Produce json
// @Success 200 {array} models.TagDTO
// @Router /tags [get]
func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tags.ToDTO())
}

// @Summary Rename a tag (admin only)
// @Tags tags
// @Accept json
// @Produce json
// @Param id path int true "Tag ID"
// @Param body body models.TagNameDTO true "New tag name"
// @Success 200 {object} models.TagDTO
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 404 {object} models.ErrorDTO "Tag not found"
// @Security BearerAuth
// @Router /tags/{id} [patch]
func (h *TagHandler) UpdateTag(c *gin.Context) {
	var path tagsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidTagID)
		return
	}

	var body models.TagNameDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	tag, err := h.svc.UpdateTag(c.Request.Context(), path.TagID, body.Name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tag.ToDTO())
}

// @Summary Delete a tag (admin only)
// @Tags tags
// @Param id path int true "Tag ID"
// @Success 204 "Deleted"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 404 {object} models.ErrorDTO "Tag not found"
// @Security BearerAuth
// @Router /tags/{id} [delete]
func (h *TagHandler) DeleteTag(c *gin.Context) {
	var path tagsPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidTagID)
		return
	}

	if err := h.svc.DeleteTag(c.Request.Context(), path.TagID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
