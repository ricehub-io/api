package handlers

import (
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/polar"
	"ricehub/src/repository"
	"ricehub/src/security"
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

func GetSubscriptionLink(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	// check if user exists
	user, err := repository.FindUserByID(token.Subject)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserNotFound))
		return
	}

	// check if user is banned
	if user.IsBanned {
		c.Error(errs.UserError(
			"Your account has been restricted",
			http.StatusForbidden,
		))
		return
	}

	// check if user doesnt have existing subscription
	subActive, err := repository.SubscriptionActive(token.Subject)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if subActive {
		c.Error(errs.UserError(
			"You already have an active subscription",
			http.StatusConflict,
		))
		return
	}

	// create new checkout session
	res, err := polar.CreateCheckoutSession(token.Subject, utils.Config.Polar.SubscriptionProductID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// return the checkoutUrl
	c.JSON(http.StatusOK, gin.H{
		"checkoutUrl": res.Checkout.URL,
	})
}
