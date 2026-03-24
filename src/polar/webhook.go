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
	logger := zap.L()

	bytes, err := c.GetRawData()
	if err != nil {
		logger.Error(
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
		logger.Error(
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
		logger.Error(
			"Failed to verify webhook",
			zap.Error(err),
		)
		c.Status(http.StatusForbidden)
		return
	}

	// try to parse the request body
	var event webhookEvent
	if err := json.Unmarshal(bytes, &event); err != nil {
		logger.Error(
			"Failed to json parse webhook event body",
			zap.Error(err),
			zap.ByteString("body", bytes),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case components.WebhookEventTypeOrderPaid:
		if ok := handleOrderPaid(event.Data); !ok {
			c.Status(http.StatusBadRequest)
			return
		}
	case components.WebhookEventTypeSubscriptionActive:
		if ok := handleSubscriptionActive(event.Data); !ok {
			c.Status(http.StatusBadRequest)
			return
		}
	default:
		logger.Debug(
			"Unhandled polar webhook event type received",
			zap.String("type", string(event.Type)),
		)
	}

	// unaimeds: not sure if this response is required
	// it was like that in an example so ;ppp
	c.JSON(http.StatusOK, bytes)
}

func handleSubscriptionActive(rawData json.RawMessage) bool {
	logger := zap.L()

	var data components.Subscription
	if err := data.UnmarshalJSON(rawData); err != nil {
		logger.Error(
			"Failed to json parse subscription payload",
			zap.Error(err),
		)
		return false
	}

	if data.ProductID != utils.Config.Polar.SubscriptionProductID.String() {
		logger.Warn(
			"Received 'subscription.active' event for invalid product ID",
			zap.String("data", string(rawData)),
		)
		return true
	}

	// userID := data.Customer.ExternalID
	// if userID == nil {
	// 	logger.Warn(
	// 		"Received 'customer.state_changed' for nil external user",
	// 		zap.String("data", string(rawData)),
	// 	)
	// 	return true
	// }

	// ok, err := repository.InsertUserSubscription(*userID, data.CurrentPeriodEnd)
	// if err != nil {
	// 	logger.Error(
	// 		"Failed to insert user subscription",
	// 		zap.Error(err),
	// 		zap.String("event_data", string(rawData)),
	// 	)
	// 	return false
	// }
	// if !ok {
	// 	logger.Error(
	// 		"Unexpected false returned when inserting user subscription",
	// 		zap.String("event_data", string(rawData)),
	// 	)
	// 	return false
	// }

	// logger.Info(
	// 	"New user subscription",
	// 	zap.Stringp("user_id", userID),
	// )

	return true
}

// TODO: gracefully handle required nil fields in payload
func handleOrderPaid(rawData json.RawMessage) bool {
	logger := zap.L()

	var data components.Order
	if err := data.UnmarshalJSON(rawData); err != nil {
		logger.Error(
			"Failed to json parse paid order event body",
			zap.Error(err),
		)
		return false
	}

	userID := data.Customer.ExternalID
	if userID == nil {
		logger.Error("External customer ID from order data is nil")
		return false
	}

	riceID := data.Metadata["rice_id"].Str
	if riceID == nil {
		logger.Error("Rice ID from order's metadata is nil")
		return false
	}

	paid_amount := float32(data.TotalAmount) / 100.0
	logger.Info(
		"Received order.paid event",
		zap.Stringp("user_id", userID),
		zap.Stringp("rice_id", riceID),
		zap.Float32("paid_amount", paid_amount),
	)

	err := repository.InsertDotfilesPurchase(*userID, *riceID, paid_amount)
	if err != nil {
		logger.Error(
			"Unexpected error received from database when inserting dotfiles purchase",
			zap.Error(err),
			zap.Stringp("user_id", userID),
			zap.Stringp("rice_id", riceID),
			zap.Float32("paid_amount", paid_amount),
		)
		return false
	}

	return true
}
