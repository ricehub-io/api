package handlers

import (
	"net/http"

	"github.com/ricehub-io/api/internal/services"

	"github.com/gin-gonic/gin"
)

type LinkHandler struct {
	svc *services.LinkService
}

func NewLinkHandler(svc *services.LinkService) *LinkHandler {
	return &LinkHandler{svc}
}

// @Summary Get a link by name
// @Tags links
// @Produce json
// @Param name path string true "Link name"
// @Success 200 {object} models.LinkDTO
// @Failure 404 {object} models.ErrorDTO "Link not found"
// @Router /links/{name} [get]
func (h *LinkHandler) GetLinkByName(c *gin.Context) {
	link, err := h.svc.GetLinkByName(c.Request.Context(), c.Param("name"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, link.ToDTO())
}

// @Summary Get the checkout URL for a subscription
// @Tags links
// @Produce json
// @Success 200 {object} object "Returns checkoutUrl (string)"
// @Failure 409 {object} models.ErrorDTO "Already has active subscription"
// @Security BearerAuth
// @Router /links/subscription [get]
func (h *LinkHandler) GetSubscriptionLink(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
	// TODO: call payments service via gRPC
	// token := c.MustGet("token").(*security.AccessToken)
	// userID, _ := uuid.Parse(token.Subject)

	// checkoutURL, err := h.svc.GetSubscriptionLink(c.Request.Context(), userID, config.Config.Polar.SubscriptionProductID)
	// if err != nil {
	// 	c.Error(err)
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{"checkoutUrl": checkoutURL})
}
