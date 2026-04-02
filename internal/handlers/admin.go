package handlers

import (
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/gin-gonic/gin"
)

func ServiceStatistics(c *gin.Context) {
	stats, err := repository.FetchServiceStatistics()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, stats.ToDTO())
}
