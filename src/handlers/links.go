package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
)

func GetLinkByName(c *gin.Context) {
	name := c.Param("name")

	link, err := repository.FindLink(name)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserError(
			"Link with provided name not found",
			http.StatusNotFound,
		)))
		return
	}

	c.JSON(http.StatusOK, link.ToDTO())
}
