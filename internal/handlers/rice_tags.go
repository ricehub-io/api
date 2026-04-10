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

func AddRiceTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

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

	if err := services.AddRiceTags(riceID, userID, token.IsAdmin, body.Tags); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func RemoveRiceTags(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

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

	if err := services.RemoveRiceTags(riceID, userID, token.IsAdmin, body.Tags); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
