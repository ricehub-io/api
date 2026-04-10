package services

import (
	"errors"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type CommentService struct{}

func NewCommentService() *CommentService {
	return &CommentService{}
}

// CreateComment inserts a new comment under the given rice post.
// Returns RiceNotFound if the rice doesn't exist.
func (s *CommentService) CreateComment(userID uuid.UUID, dto models.CreateCommentDTO) (models.RiceComment, errs.AppError) {
	riceID, _ := uuid.Parse(dto.RiceID)

	comment, err := repository.InsertComment(riceID, userID, dto.Content)
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
func (s *CommentService) ListComments(limit int) ([]models.CommentWithUser, errs.AppError) {
	comments, err := repository.FetchRecentComments(limit)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return comments, nil
}

// GetCommentByID fetches given comment and returns CommentNotFound if not found.
func (s *CommentService) GetCommentByID(commentID uuid.UUID) (models.RiceCommentWithSlug, errs.AppError) {
	comment, err := repository.FindCommentByID(commentID)
	if err != nil {
		return comment, errs.FromDBError(err, errs.CommentNotFound)
	}
	return comment, nil
}

// UpdateComment checks if user can modify the comment and updates it with given content.
func (s *CommentService) UpdateComment(isAdmin bool, userID, commentID uuid.UUID, content string) (models.RiceComment, errs.AppError) {
	if err := s.canModifyComment(isAdmin, userID, commentID); err != nil {
		return models.RiceComment{}, err
	}

	comment, err := repository.UpdateComment(commentID, content)
	if err != nil {
		return comment, errs.InternalError(err)
	}

	return comment, nil
}

// DeleteComment checks if user can modify the comment and deletes it if so.
func (s *CommentService) DeleteComment(isAdmin bool, userID, commentID uuid.UUID) errs.AppError {
	if err := s.canModifyComment(isAdmin, userID, commentID); err != nil {
		return err
	}

	if err := repository.DeleteComment(commentID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// canModifyComment checks whether user is either an admin or author of the given comment.
// Returns NoAccess if user is not allowed to modify it.
func (s *CommentService) canModifyComment(isAdmin bool, userID, commentID uuid.UUID) errs.AppError {
	if isAdmin {
		return nil
	}

	isAuthor, err := repository.UserOwnsComment(commentID, userID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !isAuthor {
		return errs.NoAccess
	}

	return nil
}
