package repository

import (
	"context"
	"ricehub/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func InsertUserSubscription(
	userID uuid.UUID,
	currentPeriodEnd time.Time,
) (models.UserSubscription, error) {
	const query = `
	INSERT INTO user_subscriptions (user_id, current_period_end, status)
	VALUES ($1, $2, 'active')
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_end = excluded.current_period_end
	RETURNING *
	`
	return rowToStruct[models.UserSubscription](query, userID, currentPeriodEnd)
}

func InsertUserSubscriptionTx(
	tx pgx.Tx,
	userID uuid.UUID,
	currentPeriodEnd time.Time,
) error {
	const query = `
	INSERT INTO user_subscriptions (user_id, current_period_end, status)
	VALUES ($1, $2, 'active')
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_end = excluded.current_period_end, status = excluded.status
	`
	_, err := tx.Exec(context.Background(), query, userID, currentPeriodEnd)
	return err
}

func SubscriptionActive(userID uuid.UUID) (active bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM user_subscriptions
		WHERE
			user_id = $1 AND
			status = 'active' OR
			(status = 'canceled' AND current_period_end > now())
	)
	`
	err = db.QueryRow(context.Background(), query, userID).Scan(&active)
	return
}

// CancelUserSubscriptionsExcept cancels all user subscriptions except for
// rows with user_id contained inside given userIDs.
func CancelUserSubscriptionsExcept(tx pgx.Tx, userIDs []uuid.UUID) (int64, error) {
	const query = `
	UPDATE user_subscriptions
	SET status = 'canceled'
	WHERE user_id <> ALL($1)
	`
	cmd, err := tx.Exec(context.Background(), query, userIDs)
	return cmd.RowsAffected(), err
}
