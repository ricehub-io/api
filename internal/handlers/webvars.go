package handlers

import (
	"net/http"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

func GetWebVarByKey(c *gin.Context) {
	key := c.Param("key")

	v, err := services.GetWebVarByKey(key)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, v.ToDTO())
}
