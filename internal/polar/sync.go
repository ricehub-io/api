package polar

// import (
// 	"context"
// 	"errors"
// 	"slices"
// 	"time"

// 	"github.com/polarsource/polar-go/models/components"
// 	"github.com/ricehub-io/api/internal/models"
// 	"github.com/ricehub-io/api/internal/repository"

// 	"github.com/google/uuid"
// 	"github.com/jackc/pgx/v5"
// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"go.uber.org/zap"
// )

// func StartSyncThread(
// 	dbPool *pgxpool.Pool,
// 	rdfRepo *repository.RiceDotfilesRepository,
// 	dfpRepo *repository.DotfilesPurchaseRepository,
// 	subRepo *repository.UserSubscriptionRepository,
// ) {
// 	l := zap.L()
// 	for {
// 		l.Info("Syncing internal state with Polar...")
// 		sync(dbPool, rdfRepo, dfpRepo, subRepo)
// 		time.Sleep(24 * time.Hour)
// 	}
// }

// func sync(
// 	dbPool *pgxpool.Pool,
// 	rdfRepo *repository.RiceDotfilesRepository,
// 	dfpRepo *repository.DotfilesPurchaseRepository,
// 	subRepo *repository.UserSubscriptionRepository,
// ) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
// 	defer cancel()

// 	l := zap.L()
// 	if err := syncDotfilesPurchases(ctx, dbPool, rdfRepo, dfpRepo); err != nil {
// 		l.Error("Failed to sync dotfiles purchases", zap.Error(err))
// 	}

// 	if err := syncSubscriptions(ctx, dbPool, subRepo); err != nil {
// 		l.Error("Failed to sync subscriptions", zap.Error(err))
// 	}
// }

// func syncDotfilesPurchases(
// 	ctx context.Context,
// 	dbPool *pgxpool.Pool,
// 	rdfRepo *repository.RiceDotfilesRepository,
// 	dfpRepo *repository.DotfilesPurchaseRepository,
// ) error {
// 	after := time.Now().Add(-24 * time.Hour)

// 	events, err := EventList(components.SystemEventTypeOrderPaid, &after)
// 	if err != nil {
// 		return err
// 	}

// 	stored, err := dfpRepo.DotfilesPurchases(ctx, after)
// 	if err != nil {
// 		return err
// 	}

// 	tx, err := dbPool.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(ctx)

// 	new := 0
// 	for _, event := range events {
// 		order := event.OrderPaidEvent

// 		meta := order.GetMetadata()
// 		strProductID := meta.ProductID
// 		strCustomerID := order.GetCustomer().ExternalID
// 		if strProductID == nil || strCustomerID == nil {
// 			continue
// 		}
// 		productID, _ := uuid.Parse(*strProductID)
// 		customerID, _ := uuid.Parse(*strCustomerID)

// 		if !slices.ContainsFunc(stored, func(s models.DotfilesPurchase) bool {
// 			return s.ProductID == productID && s.UserID == customerID
// 		}) {
// 			df, err := rdfRepo.FindDotfilesByProductID(ctx, productID)
// 			if err != nil {
// 				if errors.Is(err, pgx.ErrNoRows) {
// 					continue
// 				}
// 				return err
// 			}

// 			paidAmount := centsToPrice(meta.Amount)
// 			err = dfpRepo.InsertDotfilesPurchaseTx(ctx, customerID, df.RiceID, paidAmount, order.Timestamp)
// 			if err != nil {
// 				return err
// 			}

// 			new++
// 		}
// 	}

// 	if err := tx.Commit(ctx); err != nil {
// 		return err
// 	}

// 	zap.L().Sugar().Infof("Synchronized %d new dotfiles purchases", new)
// 	return nil
// }

// func syncSubscriptions(
// 	ctx context.Context,
// 	dbPool *pgxpool.Pool,
// 	repo *repository.UserSubscriptionRepository,
// ) error {
// 	subs, err := SubscriptionList()
// 	if err != nil {
// 		return err
// 	}

// 	tx, err := dbPool.BeginTx(ctx, pgx.TxOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(ctx)

// 	// upsert active subscriptions
// 	seen := []uuid.UUID{}
// 	for _, sub := range subs {
// 		strCustomerID := sub.Customer.ExternalID
// 		if strCustomerID == nil {
// 			continue
// 		}
// 		customerID, _ := uuid.Parse(*strCustomerID)
// 		seen = append(seen, customerID)

// 		err := repo.InsertUserSubscriptionTx(ctx, customerID, sub.CurrentPeriodEnd)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// delete unseen
// 	cancelled, err := repo.CancelUserSubscriptionsExcept(ctx, seen)
// 	if err != nil {
// 		return err
// 	}

// 	if err := tx.Commit(ctx); err != nil {
// 		return err
// 	}

// 	l := zap.L().Sugar()
// 	l.Infof("Upserted %d subscriptions", len(seen))
// 	l.Infof("Canceled %d subscriptions", cancelled)
// 	return nil
// }

// func centsToPrice(cents int64) float32 {
// 	return float32(cents) / 100.0
// }
