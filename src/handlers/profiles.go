package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
)

type profilesPath struct {
	Username string `uri:"username" binding:"required,alphanum"`
}

func GetUserProfile(c *gin.Context) {
	var path profilesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.UserError("Invalid username path parameter. It must be an alphanumeric string.", http.StatusBadRequest))
		return
	}

	// I could create a new repo function that executes these two statements
	// in one query using `WITH ___ AS () [...]` but im tooo lazyyyyyyyyyy

	// fetch user data
	user, err := repository.FindUserByUsername(path.Username)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserNotFound))
		return
	}

	callerUserID := GetUserIDFromRequest(c)

	// fetch user rices
	rices, err := repository.FetchUserRices(user.ID, callerUserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user.ToDTO(),
		"rices": models.PartialRicesToDTO(rices),
	})
}
