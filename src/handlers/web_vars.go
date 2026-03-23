package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
)

func GetWebsiteVariable(c *gin.Context) {
	key := c.Param("key")

	v, err := repository.FindWebsiteVariable(key)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserError(
			"Website variable with provided key not found",
			http.StatusNotFound,
		)))
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
