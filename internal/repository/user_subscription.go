package repository

import (
	"context"
	"ricehub/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserSubscriptionRepository struct {
	db DBExecutor
}

func NewUserSubscriptionRepository(db DBExecutor) *UserSubscriptionRepository {
	return &UserSubscriptionRepository{db}
}
func (r *UserSubscriptionRepository) WithTx(tx pgx.Tx) *UserSubscriptionRepository {
	return &UserSubscriptionRepository{tx}
}

func (r *UserSubscriptionRepository) InsertUserSubscription(
	ctx context.Context,
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
	return rowToStruct[models.UserSubscription](ctx, r.db, query, userID, currentPeriodEnd)
}

func (r *UserSubscriptionRepository) InsertUserSubscriptionTx(
	ctx context.Context,
	userID uuid.UUID,
	currentPeriodEnd time.Time,
) error {
	const query = `
	INSERT INTO user_subscriptions (user_id, current_period_end, status)
	VALUES ($1, $2, 'active')
	ON CONFLICT (user_id) DO UPDATE
		SET current_period_end = excluded.current_period_end, status = excluded.status
	`
	_, err := r.db.Exec(ctx, query, userID, currentPeriodEnd)
	return err
}

func (r *UserSubscriptionRepository) SubscriptionActive(
	ctx context.Context,
	userID uuid.UUID,
) (active bool, err error) {
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
	err = r.db.QueryRow(ctx, query, userID).Scan(&active)
	return
}

// CancelUserSubscriptionsExcept cancels all user subscriptions except for
// rows with user_id contained inside given userIDs.
func (r *UserSubscriptionRepository) CancelUserSubscriptionsExcept(
	ctx context.Context,
	userIDs []uuid.UUID,
) (int64, error) {
	const query = `
	UPDATE user_subscriptions
	SET status = 'canceled'
	WHERE user_id <> ALL($1)
	`
	cmd, err := r.db.Exec(ctx, query, userIDs)
	return cmd.RowsAffected(), err
}
