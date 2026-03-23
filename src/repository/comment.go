package repository

import (
	"context"
	"ricehub/src/models"
)

func InsertComment(riceID string, authorID string, content string) (models.RiceComment, error) {
	const query = `
	INSERT INTO rice_comments (rice_id, author_id, content)
	VALUES ($1, $2, $3)
	RETURNING *
	`

	return rowToStruct[models.RiceComment](query, riceID, authorID, content)
}

func UserOwnsComment(commentID string, userID string) (exists bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM rice_comments
		WHERE id = $1 AND author_id = $2
	)
	`

	err = db.QueryRow(context.Background(), query, commentID, userID).Scan(&exists)
	return
}

func FetchRecentComments(limit int) ([]models.CommentWithUser, error) {
	const query = `
	SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path, u.is_banned
	FROM rice_comments c
	JOIN users_with_ban_status u ON u.id = c.author_id
	ORDER BY c.created_at DESC
	LIMIT $1
	`

	return rowsToStruct[models.CommentWithUser](query, limit)
}

func FetchCommentsByRiceID(riceID string) ([]models.CommentWithUser, error) {
	const query = `
	SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path, u.is_banned
	FROM rice_comments c
	JOIN users_with_ban_status u ON u.id = c.author_id
	WHERE rice_id = $1
	ORDER BY created_at DESC
	`

	return rowsToStruct[models.CommentWithUser](query, riceID)
}

// deluxe version of find comment because it fetches username and slug too
func FindCommentByID(commentID string) (models.RiceCommentWithSlug, error) {
	const query = `
	SELECT rc.*, r.slug AS rice_slug, u.username AS rice_author_username
	FROM rice_comments rc
	JOIN rices r ON r.id = rc.rice_id
	JOIN users u ON u.id = r.author_id
	WHERE rc.id = $1
	`

	return rowToStruct[models.RiceCommentWithSlug](query, commentID)
}

func UpdateComment(commentID string, content string) (models.RiceComment, error) {
	const query = `
	UPDATE rice_comments SET content = $1 WHERE id = $2
	RETURNING *
	`

	return rowToStruct[models.RiceComment](query, content, commentID)
}

func DeleteComment(commentID string) error {
	const query = `
	DELETE FROM rice_comments
	WHERE id = $1
	`

	_, err := db.Exec(context.Background(), query, commentID)
	return err
}
