package repository

import (
	"context"

	"github.com/ricehub-io/api/internal/models"

	"github.com/google/uuid"
)

type CommentRepository struct {
	db DBExecutor
}

func NewCommentRepository(db DBExecutor) *CommentRepository {
	return &CommentRepository{db}
}

func (r *CommentRepository) InsertComment(ctx context.Context, riceID, authorID uuid.UUID, content string) (models.RiceComment, error) {
	const query = `
	INSERT INTO rice_comments (rice_id, author_id, content)
	VALUES ($1, $2, $3)
	RETURNING *
	`

	return rowToStruct[models.RiceComment](ctx, r.db, query, riceID, authorID, content)
}

func (r *CommentRepository) UserOwnsComment(ctx context.Context, commentID, userID uuid.UUID) (exists bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM rice_comments
		WHERE id = $1 AND author_id = $2
	)
	`

	err = r.db.QueryRow(ctx, query, commentID, userID).Scan(&exists)
	return
}

func (r *CommentRepository) FetchRecentComments(ctx context.Context, limit int) ([]models.CommentWithUser, error) {
	const query = `
	SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path, u.is_banned
	FROM rice_comments c
	JOIN users_with_ban_status u ON u.id = c.author_id
	ORDER BY c.created_at DESC
	LIMIT $1
	`

	return rowsToStruct[models.CommentWithUser](ctx, r.db, query, limit)
}

func (r *CommentRepository) FetchCommentsByRiceID(ctx context.Context, riceID uuid.UUID) ([]models.CommentWithUser, error) {
	const query = `
	SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path, u.is_banned
	FROM rice_comments c
	JOIN users_with_ban_status u ON u.id = c.author_id
	WHERE rice_id = $1
	ORDER BY created_at DESC
	`

	return rowsToStruct[models.CommentWithUser](ctx, r.db, query, riceID)
}

// deluxe version of find comment because it fetches username and slug too
func (r *CommentRepository) FindCommentByID(ctx context.Context, commentID uuid.UUID) (models.RiceCommentWithSlug, error) {
	const query = `
	SELECT rc.*, r.slug AS rice_slug, u.username AS rice_author_username
	FROM rice_comments rc
	JOIN rices r ON r.id = rc.rice_id
	JOIN users u ON u.id = r.author_id
	WHERE rc.id = $1
	`
	return rowToStruct[models.RiceCommentWithSlug](ctx, r.db, query, commentID)
}

func (r *CommentRepository) UpdateComment(ctx context.Context, commentID uuid.UUID, content string) (models.RiceComment, error) {
	const query = `
	UPDATE rice_comments SET content = $1 WHERE id = $2
	RETURNING *
	`

	return rowToStruct[models.RiceComment](ctx, r.db, query, content, commentID)
}

func (r *CommentRepository) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	const query = `
	DELETE FROM rice_comments
	WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, commentID)
	return err
}
