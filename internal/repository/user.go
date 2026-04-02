package repository

import (
	"context"
	"ricehub/internal/models"
)

func UsernameExists(username string) (exists bool, err error) {
	const query = "SELECT EXISTS ( SELECT 1 FROM users WHERE username = $1 )"
	err = db.QueryRow(context.Background(), query, username).Scan(&exists)
	return
}

func InsertUser(username string, displayName string, password string) error {
	const query = `
	INSERT INTO users (username, display_name, password)
	VALUES ($1, $2, $3)
	`
	_, err := db.Exec(context.Background(), query, username, displayName, password)
	return err
}

func FetchRecentUsers(limit int) ([]models.User, error) {
	const query = `
	SELECT *
	FROM users_with_ban_status
	ORDER BY created_at DESC
	LIMIT $1
	`
	return rowsToStruct[models.User](query, limit)
}

// Fetches all banned users with ban data and orders it from recent ban
func FetchBannedUsers() ([]models.UserWithBan, error) {
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
	return rowsToStruct[models.UserWithBan](query)
}

func FindUserByUsername(username string) (models.User, error) {
	const query = "SELECT * FROM users_with_ban_status WHERE username = $1 LIMIT 1"
	return rowToStruct[models.User](query, username)
}

func FindUserByID(userID string) (models.User, error) {
	const query = "SELECT * FROM users_with_ban_status WHERE id = $1 LIMIT 1"
	return rowToStruct[models.User](query, userID)
}

func FetchUserAvatarPath(userID string) (avatarPath *string, err error) {
	const query = "SELECT avatar_path FROM users WHERE id = $1"
	err = db.QueryRow(context.Background(), query, userID).Scan(&avatarPath)
	return
}

// should I just use single `UpdateUser` function with struct of fields to update and utilize COALESCE?
func UpdateUserDisplayName(userID string, displayName string) error {
	const query = "UPDATE users SET display_name = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, displayName, userID)
	return err
}

func UpdateUserPassword(userID string, password string) error {
	const query = "UPDATE users SET password = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, password, userID)
	return err
}

func UpdateUserAvatarPath(userID string, avatarPath *string) error {
	const query = "UPDATE users SET avatar_path = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, avatarPath, userID)
	return err
}

func RevokeAdmin(userID string) error {
	const query = "UPDATE users SET is_admin = false WHERE id = $1"
	_, err := db.Exec(context.Background(), query, userID)
	return err
}

func DeleteUser(userID string) error {
	const query = "DELETE FROM users WHERE id = $1"
	_, err := db.Exec(context.Background(), query, userID)
	return err
}
