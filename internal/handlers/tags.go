package handlers

import (
	"errors"
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/validation"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateTag(c *gin.Context) {
	var newTag *models.TagNameDTO
	if err := validation.ValidateJSON(c, &newTag); err != nil {
		c.Error(err)
		return
	}

	tag, err := repository.InsertTag(newTag.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			c.Error(errs.UserError("Tag with that name already exists!", http.StatusConflict))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, tag.ToDTO())
}

func ListTags(c *gin.Context) {
	tags, err := repository.FetchTags()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	dtos := make([]models.TagDTO, len(tags))
	for i, tag := range tags {
		dtos[i] = tag.ToDTO()
	}

	c.JSON(http.StatusOK, dtos)
}

func UpdateTag(c *gin.Context) {
	tagID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(errs.InvalidTagID)
		return
	}

	var update *models.TagNameDTO
	if err := validation.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	tag, err := repository.UpdateTag(tagID, update.Name)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.TagNotFound))
		return
	}

	c.JSON(http.StatusOK, tag.ToDTO())
}

func DeleteTag(c *gin.Context) {
	tagID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(errs.InvalidTagID)
		return
	}

	deleted, err := repository.DeleteTag(tagID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(errs.TagNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}
