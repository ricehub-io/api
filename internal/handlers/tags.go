package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/services"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
)

type TagHandler struct{}

func NewTagHandler() *TagHandler {
	return &TagHandler{}
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

	tag, err := services.CreateTag(body.Name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, tag.ToDTO())
}

func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := services.ListTags()
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

	tag, err := services.UpdateTag(path.TagID, body.Name)
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

	if err := services.DeleteTag(path.TagID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
