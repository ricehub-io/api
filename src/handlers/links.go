package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/repository"
	"ricehub/src/utils"

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

// this is the only link thats fetched not from db but config,
// in case we get sql injection pwnd or somehow
// someone gains access to database, the bad threat actor
// cant change it to their own and steal money from people
func GetSubscriptionLink(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"checkoutLink": utils.Config.Polar.SubscriptionLink,
	})
}
