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

func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tags.ToDTO())
}

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
