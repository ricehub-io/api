package handlers

import (
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/security"
	"ricehub/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LinkHandler struct {
	svc *services.LinkService
}

func NewLinkHandler(svc *services.LinkService) *LinkHandler {
	return &LinkHandler{svc}
}

func (h *LinkHandler) GetLinkByName(c *gin.Context) {
	link, err := h.svc.GetLinkByName(c.Request.Context(), c.Param("name"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, link.ToDTO())
}

func (h *LinkHandler) GetSubscriptionLink(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	userID, _ := uuid.Parse(token.Subject)

	checkoutURL, err := h.svc.GetSubscriptionLink(c.Request.Context(), userID, config.Config.Polar.SubscriptionProductID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}
