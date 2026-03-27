package repository

import (
	"context"
	"ricehub/src/models"
	"time"
)

func InsertUserSubscription(userID string, currentPeriodEnd time.Time) (models.UserSubscription, error) {
	const query = `
	INSERT INTO user_subscriptions (user_id, current_period_end)
	VALUES ($1, $2)
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_end = excluded.current_period_end
	RETURNING *
	`
	return rowToStruct[models.UserSubscription](query, userID, currentPeriodEnd)
}

func SubscriptionActive(userID string) (active bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM user_subscriptions
		WHERE user_id = $1 AND current_period_end > now()
	)
	`
	err = db.QueryRow(context.Background(), query, userID).Scan(&active)
	return
}
