package repository

import (
	"context"

	"github.com/ricehub-io/api/internal/models"

	"github.com/google/uuid"
)

type UserRepository struct {
	db DBExecutor
}

func NewUserRepository(db DBExecutor) *UserRepository {
	return &UserRepository{db}
}

func (r *UserRepository) UsernameExists(ctx context.Context, username string) (exists bool, err error) {
	const query = "SELECT EXISTS ( SELECT 1 FROM users WHERE username = $1 )"
	err = r.db.QueryRow(ctx, query, username).Scan(&exists)
	return
}

func (r *UserRepository) InsertUser(
	ctx context.Context,
	username, displayName, password string,
) error {
	const query = `
	INSERT INTO users (username, display_name, password)
	VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, username, displayName, password)
	return err
}

func (r *UserRepository) FetchRecentUsers(ctx context.Context, limit int) ([]models.User, error) {
	const query = `
	SELECT *
	FROM users_with_ban_status
	ORDER BY created_at DESC
	LIMIT $1
	`
	return rowsToStruct[models.User](ctx, r.db, query, limit)
}

// Fetches all banned users with ban data and orders it from recent ban
func (r *UserRepository) FetchBannedUsers(ctx context.Context) ([]models.UserWithBan, error) {
	const query = `
	SELECT DISTINCT ON (u.id)
		to_jsonb(u) AS "user",
		to_jsonb(b) AS "ban"
	FROM users_with_ban_status u
	JOIN user_bans b ON b.user_id = u.id
	WHERE
		u.is_banned = true
	ORDER BY u.id, b.banned_at DESC
	`
	return rowsToStruct[models.UserWithBan](ctx, r.db, query)
}

func (r *UserRepository) FindUserByUsername(ctx context.Context, username string) (models.User, error) {
	const query = "SELECT * FROM users_with_ban_status WHERE username = $1 LIMIT 1"
	return rowToStruct[models.User](ctx, r.db, query, username)
}

func (r *UserRepository) FindUserByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	const query = "SELECT * FROM users_with_ban_status WHERE id = $1 LIMIT 1"
	return rowToStruct[models.User](ctx, r.db, query, userID)
}

func (r *UserRepository) FetchUserAvatarPath(ctx context.Context, userID uuid.UUID) (avatarPath *string, err error) {
	const query = "SELECT avatar_path FROM users WHERE id = $1"
	err = r.db.QueryRow(ctx, query, userID).Scan(&avatarPath)
	return
}

func (r *UserRepository) UpdateUserDisplayName(ctx context.Context, userID uuid.UUID, displayName string) error {
	const query = "UPDATE users SET display_name = $1 WHERE id = $2"
	_, err := r.db.Exec(ctx, query, displayName, userID)
	return err
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	const query = "UPDATE users SET password = $1 WHERE id = $2"
	_, err := r.db.Exec(ctx, query, password, userID)
	return err
}

func (r *UserRepository) UpdateUserAvatarPath(ctx context.Context, userID uuid.UUID, avatarPath *string) error {
	const query = "UPDATE users SET avatar_path = $1 WHERE id = $2"
	_, err := r.db.Exec(ctx, query, avatarPath, userID)
	return err
}

func (r *UserRepository) RevokeAdmin(ctx context.Context, userID uuid.UUID) error {
	const query = "UPDATE users SET is_admin = false WHERE id = $1"
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	const query = "DELETE FROM users WHERE id = $1"
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
