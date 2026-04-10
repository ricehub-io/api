package handlers

import (
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/security"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type LinkHandler struct{}

func NewLinkHandler() *LinkHandler {
	return &LinkHandler{}
}

func (h *LinkHandler) GetLinkByName(c *gin.Context) {
	link, err := services.GetLinkByName(c.Param("name"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, link.ToDTO())
}

func (h *LinkHandler) GetSubscriptionLink(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	checkoutURL, err := services.GetSubscriptionLink(userID, config.Config.Polar.SubscriptionProductID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}
