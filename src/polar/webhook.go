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

	if ok := processEvent(webhookID, event.Type, event.Data); !ok {
		c.Status(http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, bytes)
}

func processEvent(webhookID string, eventType components.WebhookEventType, rawData json.RawMessage) bool {
	logger := zap.L()

	insertEvent := func() {
		err := repository.InsertEvent(webhookID, string(eventType), rawData)
		if err != nil {
			logger.Error(
				"Failed to insert new webhook event into database",
				zap.Error(err),
				zap.String("webhook_id", webhookID),
			)
		}
	}

	eventProcessed := func() {
		err := repository.EventProcessed(webhookID)
		if err != nil {
			logger.Error(
				"Failed to update webhook event's processed_at timestamp in database",
				zap.Error(err),
				zap.String("webhook_id", webhookID),
			)
		}
	}

	setEventError := func(eventErr error) {
		err := repository.SetEventError(webhookID, eventErr.Error())
		if err != nil {
			logger.Error(
				"Failed to set webhook event's error in database",
				zap.Error(err),
				zap.String("webhook_id", webhookID),
			)
		}
	}

	switch eventType {
	case components.WebhookEventTypeOrderPaid:
		insertEvent()
		if err := handleOrderPaid(rawData); err != nil {
			setEventError(err)
			return false
		}
		eventProcessed()
	case components.WebhookEventTypeSubscriptionActive:
		insertEvent()
		if err := handleSubscriptionActive(rawData); err != nil {
			setEventError(err)
			return false
		}
		eventProcessed()
	default:
		logger.Debug(
			"Unhandled polar webhook event type received",
			zap.String("type", string(eventType)),
		)
	}

	return true
}

func handleSubscriptionActive(rawData json.RawMessage) error {
	logger := zap.L()

	var data components.Subscription
	if err := data.UnmarshalJSON(rawData); err != nil {
		logger.Error(
			"Failed to json parse subscription payload",
			zap.Error(err),
		)
		return err
	}

	if data.ProductID != utils.Config.Polar.SubscriptionProductID.String() {
		logger.Warn(
			"Received 'subscription.active' event for unhandled product ID",
			zap.String("data", string(rawData)),
		)
		return nil
	}

	userID := data.Customer.ExternalID
	if userID == nil {
		logger.Warn(
			"Received 'subscription.active' for nil external user",
			zap.String("data", string(rawData)),
		)
		return nil
	}

	sub, err := repository.InsertUserSubscription(*userID, data.CurrentPeriodEnd)
	if err != nil {
		logger.Error(
			"Failed to insert user subscription",
			zap.Error(err),
			zap.String("event_data", string(rawData)),
		)
		return err
	}

	logger.Info(
		"New user subscription",
		zap.Stringp("user_id", userID),
		zap.String("subscription_id", sub.ID.String()),
	)

	return nil
}

func handleOrderPaid(rawData json.RawMessage) error {
	logger := zap.L()

	var data components.Order
	if err := data.UnmarshalJSON(rawData); err != nil {
		logger.Error(
			"Failed to json parse paid order event body",
			zap.Error(err),
		)
		return err
	}

	userID := data.Customer.ExternalID
	if userID == nil {
		logger.Warn(
			"External customer ID from order data is nil",
			zap.String("raw_data", string(rawData)),
		)
		return nil
	}

	if *data.ProductID == utils.Config.Polar.SubscriptionProductID.String() {
		// subscription is handled elsewhere
		return nil
	}

	df, err := repository.FindDotfilesByProductID(*data.ProductID)
	if err != nil {
		logger.Error(
			"Unexpected database error occurred when trying to find dotfiles by product ID",
			zap.Error(err),
			zap.Stringp("product_id", data.ProductID),
		)
		return err
	}

	paid_amount := float32(data.TotalAmount) / 100.0
	logger.Info(
		"Received order.paid event",
		zap.Stringp("user_id", userID),
		zap.String("rice_id", df.RiceID.String()),
		zap.Float32("paid_amount", paid_amount),
	)

	err = repository.InsertDotfilesPurchase(*userID, df.RiceID, paid_amount)
	if err != nil {
		logger.Error(
			"Unexpected error received from database when inserting dotfiles purchase",
			zap.Error(err),
			zap.Stringp("user_id", userID),
			zap.String("rice_id", df.RiceID.String()),
			zap.Float32("paid_amount", paid_amount),
		)
		return err
	}

	return nil
}
