package repository

import (
	"context"
	"time"
)

func InsertUserSubscription(userID string, currentPeriodEnd time.Time) (bool, error) {
	const query = `
	INSERT INTO user_subscriptions (user_id, current_period_end)
	VALUES ($1, $2)
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_end = excluded.current_period_end
	`

	cmd, err := db.Exec(context.Background(), query, userID, currentPeriodEnd)
	return cmd.RowsAffected() > 0, err
}
