package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct{}

func NewProfileHandler() *ProfileHandler {
	return &ProfileHandler{}
}

type profilesPath struct {
	Username string `uri:"username" binding:"required,alphanum"`
}

func (h *ProfileHandler) GetProfileByUsername(c *gin.Context) {
	var path profilesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.UserError(
			"Invalid username path parameter. It must be an alphanumeric string.",
			http.StatusBadRequest,
		))
		return
	}

	callerID := GetUserIDFromRequest(c)
	res, err := services.GetProfileByUsername(path.Username, callerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  res.User.ToDTO(),
		"rices": res.Rices.ToDTO(),
	})
}
