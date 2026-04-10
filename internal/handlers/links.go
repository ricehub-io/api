package handlers

import (
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/security"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
)

type LinkHandler struct {
	svc *services.LinkService
}

func NewLinkHandler(svc *services.LinkService) *LinkHandler {
	return &LinkHandler{svc}
}

func (h *LinkHandler) GetLinkByName(c *gin.Context) {
	link, err := h.svc.GetLinkByName(c.Param("name"))
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

	checkoutURL, err := h.svc.GetSubscriptionLink(userID, config.Config.Polar.SubscriptionProductID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}
