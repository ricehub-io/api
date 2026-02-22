package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetLinkByName(c *gin.Context) {
	name := c.Param("name")

	link, err := repository.FetchLink(name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.UserError("Link with provided name not found", http.StatusNotFound))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, link.ToDTO())
}
