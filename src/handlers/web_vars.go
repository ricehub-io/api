package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetWebsiteVariable(c *gin.Context) {
	key := c.Param("key")

	v, err := repository.FetchWebsiteVariable(key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.UserError("Website variable with provided key not found", http.StatusNotFound))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
