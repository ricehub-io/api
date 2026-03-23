package polar

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"ricehub/src/repository"
	"ricehub/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/polarsource/polar-go/models/components"
	svix "github.com/svix/svix-webhooks/go"
	"go.uber.org/zap"
)

// afaik Polar's Go SDK has no generic event payload type
type webhookEvent struct {
	Type components.WebhookEventType `json:"type"`
	Data json.RawMessage             `json:"data"`
}

// polar webhook endpoint
func WebhookListener(c *gin.Context) {
	log := zap.L()

	bytes, err := c.GetRawData()
	if err != nil {
		log.Error(
			"Error reading webhook request body",
			zap.Error(err),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	// verify webhook
	webhookID := c.GetHeader("webhook-id")
	webhookTimestamp := c.GetHeader("webhook-timestamp")
	webhookSignature := c.GetHeader("webhook-signature")
	base64Secret := base64.StdEncoding.EncodeToString([]byte(utils.Config.Polar.WebhookSecret))

	wh, err := svix.NewWebhook(base64Secret)
	if err != nil {
		log.Error(
			"Failed to create webhook verifier",
			zap.Error(err),
		)
		c.Status(http.StatusForbidden)
		return
	}

	headers := http.Header{}
	headers.Set("webhook-id", webhookID)
	headers.Set("webhook-timestamp", webhookTimestamp)
	headers.Set("webhook-signature", webhookSignature)

	err = wh.Verify(bytes, headers)
	if err != nil {
		log.Error(
			"Failed to verify webhook",
			zap.Error(err),
		)
		c.Status(http.StatusForbidden)
		return
	}

	// try to parse the request body
	var event webhookEvent
	if err := json.Unmarshal(bytes, &event); err != nil {
		log.Error(
			"Failed to json parse webhook event body",
			zap.Error(err),
			zap.ByteString("body", bytes),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	// we only care about 'order.paid' events
	if event.Type != components.WebhookEventTypeOrderPaid {
		c.JSON(http.StatusOK, bytes)
		return
	}

	// try to decode the event data
	var data components.Order
	if err := data.UnmarshalJSON(event.Data); err != nil {
		log.Error(
			"Failed to json parse paid order event body",
			zap.Error(err),
			zap.ByteString("body", bytes),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	userID := data.Customer.ExternalID
	if userID == nil {
		log.Error(
			"External customer ID from order data is nil",
			zap.ByteString("body", bytes),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	riceID := data.Metadata["rice_id"].Str
	if riceID == nil {
		log.Error(
			"Rice ID from order's metadata is nil",
			zap.ByteString("body", bytes),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	paid_amount := float32(data.TotalAmount) / 100.0
	log.Info(
		"Received order.paid event",
		zap.Stringp("user_id", userID),
		zap.Stringp("rice_id", riceID),
		zap.Float32("paid_amount", paid_amount),
	)

	// insert purchase into database
	err = repository.InsertDotfilesPurchase(*userID, *riceID, paid_amount)
	if err != nil {
		log.Error(
			"Unexpected error received from database when inserting dotfiles purchase",
			zap.Error(err),
			zap.Stringp("user_id", userID),
			zap.Stringp("rice_id", riceID),
			zap.Float32("paid_amount", paid_amount),
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, bytes)
}
