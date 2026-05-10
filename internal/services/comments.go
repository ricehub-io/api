package services

import (
	"context"
	"errors"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"
	"github.com/ricehub-io/api/internal/security"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type CommentService struct {
	comments *repository.CommentRepository
	users    *repository.UserRepository
	bans     *repository.UserBanRepository
}

func NewCommentService(
	comments *repository.CommentRepository,
	users *repository.UserRepository,
	bans *repository.UserBanRepository,
) *CommentService {
	return &CommentService{comments, users, bans}
}

// CreateComment inserts a new comment under the given rice post.
// Returns RiceNotFound if the rice doesn't exist.
func (s *CommentService) CreateComment(
	ctx context.Context,
	userID uuid.UUID,
	dto models.CreateCommentDTO,
) (models.RiceComment, errs.AppError) {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return models.RiceComment{}, err
	}

	riceID, _ := uuid.Parse(dto.RiceID)

	comment, err := s.comments.InsertComment(ctx, riceID, userID, dto.Content)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return comment, errs.RiceNotFound
		}
		return comment, errs.InternalError(err)
	}

	return comment, nil
}

// ListComments fetches limited amount of comments sorted by creation date.
func (s *CommentService) ListComments(ctx context.Context, limit int) ([]models.CommentWithUser, errs.AppError) {
	comments, err := s.comments.FetchRecentComments(ctx, limit)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return comments, nil
}

// GetCommentByID fetches given comment and returns CommentNotFound if not found.
func (s *CommentService) GetCommentByID(ctx context.Context, commentID uuid.UUID) (models.RiceCommentWithSlug, errs.AppError) {
	comment, err := s.comments.FindCommentByID(ctx, commentID)
	if err != nil {
		return comment, errs.FromDBError(err, errs.CommentNotFound)
	}
	return comment, nil
}

// UpdateComment checks if user can modify the comment and updates it with given content.
func (s *CommentService) UpdateComment(
	ctx context.Context,
	isAdmin bool,
	userID, commentID uuid.UUID,
	content string,
) (models.RiceComment, errs.AppError) {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return models.RiceComment{}, err
	}

	if err := s.canModifyComment(ctx, isAdmin, userID, commentID); err != nil {
		return models.RiceComment{}, err
	}

	comment, err := s.comments.UpdateComment(ctx, commentID, content)
	if err != nil {
		return comment, errs.InternalError(err)
	}

	return comment, nil
}

// DeleteComment checks if user can modify the comment and deletes it if so.
func (s *CommentService) DeleteComment(
	ctx context.Context,
	isAdmin bool,
	userID, commentID uuid.UUID,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return err
	}

	if err := s.canModifyComment(ctx, isAdmin, userID, commentID); err != nil {
		return err
	}

	if err := s.comments.DeleteComment(ctx, commentID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// canModifyComment checks whether user is either an admin or author of the given comment.
// Returns NoAccess if user is not allowed to modify it.
func (s *CommentService) canModifyComment(
	ctx context.Context,
	isAdmin bool,
	userID, commentID uuid.UUID,
) errs.AppError {
	if isAdmin {
		return nil
	}

	isAuthor, err := s.comments.UserOwnsComment(ctx, commentID, userID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !isAuthor {
		return errs.NoAccess
	}

	return nil
}
