package repository

import (
	"context"
	"time"

	"github.com/ricehub-io/api/internal/models"

	"github.com/google/uuid"
)

type UserBanRepository struct {
	db DBExecutor
}

func NewUserBanRepository(db DBExecutor) *UserBanRepository {
	return &UserBanRepository{db}
}

func (r *UserBanRepository) IsUserBanned(
	ctx context.Context,
	userID uuid.UUID,
) (models.UserState, error) {
	const query = `
	SELECT
		EXISTS(
			SELECT 1 FROM users WHERE id = $1
		) AS user_exists,
		EXISTS(
			SELECT 1
			FROM user_bans
			WHERE
				user_id = $1 AND
				(expires_at > NOW() OR expires_at IS NULL) AND
				is_revoked = false
		) AS user_banned
	`
	return rowToStruct[models.UserState](ctx, r.db, query, userID)
}

func (r *UserBanRepository) InsertBan(
	ctx context.Context,
	userID, adminID uuid.UUID,
	reason string,
	expiresAt *time.Time,
) (models.UserBan, error) {
	const query = `
	INSERT INTO user_bans (user_id, admin_id, reason, expires_at)
	VALUES ($1, $2, $3, $4)
	RETURNING *
	`
	return rowToStruct[models.UserBan](ctx, r.db, query, userID, adminID, reason, expiresAt)
}

func (r *UserBanRepository) FindUserBan(ctx context.Context, userID uuid.UUID) (models.UserBan, error) {
	const query = `
	SELECT *
	FROM user_bans
	WHERE 
		user_id = $1 AND 
		(expires_at > NOW() OR expires_at IS NULL) AND
		is_revoked = false
	`
	return rowToStruct[models.UserBan](ctx, r.db, query, userID)
}

// Revoke is an irreversible action therefore no need for generalized 'set is_revoked'
// function as it can only be updated to one state.
func (r *UserBanRepository) RevokeBan(ctx context.Context, userID uuid.UUID) error {
	const query = `
	UPDATE user_bans
	SET is_revoked = true
	WHERE user_id = $1
	`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
