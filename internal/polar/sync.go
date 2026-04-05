package polar

import (
	"context"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/polarsource/polar-go/models/components"
	"go.uber.org/zap"
)

func StartSyncThread() {
	logger := zap.L()
	for {
		logger.Info("Syncing internal state with Polar...")
		if err := syncDotfilesPurchases(); err != nil {
			logger.Error("Failed to sync dotfiles purchases", zap.Error(err))
		}
		syncSubscriptions()
		time.Sleep(24 * time.Hour)
	}
}

func syncDotfilesPurchases() error {
	after := time.Now().Add(-24 * time.Hour)

	events, err := EventList(components.SystemEventTypeOrderPaid, &after)
	if err != nil {
		return err
	}

	stored, err := repository.DotfilesPurchases(after)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	tx, err := repository.StartTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	new := 0
	for _, event := range events {
		data := event.OrderPaidEvent
		if data == nil {
			continue
		}

		strProductID := data.Metadata.GetProductID()
		strUserID := data.Customer.GetExternalID()
		if strProductID == nil || strUserID == nil {
			continue
		}
		productID, _ := uuid.Parse(*strProductID)
		userID, _ := uuid.Parse(*strUserID)

		if !slices.ContainsFunc(stored, func(s models.DotfilesPurchase) bool {
			return s.ProductID == productID && s.UserID == userID
		}) {
			df, err := repository.FindDotfilesByProductID(productID)
			if err != nil {
				return err
			}

			paidAmount := centsToPrice(data.Metadata.GetAmount())
			err = repository.InsertDotfilesPurchaseTx(tx, userID, df.RiceID, paidAmount, data.Timestamp)
			if err != nil {
				return err
			}

			new++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	zap.L().Sugar().Infof("Synchronized %d new 'order.paid' events", new)
	return nil
}

func syncSubscriptions() {

}

func centsToPrice(cents int64) float32 {
	return float32(cents) / 100.0
}
