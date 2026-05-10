package services

import (
	"context"
	"errors"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type TagService struct {
	tags *repository.TagRepository
}

func NewTagService(tags *repository.TagRepository) *TagService {
	return &TagService{tags}
}

// CreateTag inserts a new tag with the given name.
// Returns an error if a tag with that name already exists.
func (s *TagService) CreateTag(ctx context.Context, name string) (models.Tag, errs.AppError) {
	tag, err := s.tags.InsertTag(ctx, name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return tag, errs.TagExists
		}
		return tag, errs.InternalError(err)
	}

	return tag, nil
}

// ListTags returns all tags ordered by ID.
func (s *TagService) ListTags(ctx context.Context) (models.Tags, errs.AppError) {
	tags, err := s.tags.FetchTags(ctx)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return tags, nil
}

// UpdateTag updates the name of the tag with the given ID.
// Returns TagNotFound if no tag with that ID exists.
func (s *TagService) UpdateTag(ctx context.Context, id int, name string) (models.Tag, errs.AppError) {
	tag, err := s.tags.UpdateTag(ctx, id, name)
	if err != nil {
		return tag, errs.FromDBError(err, errs.TagNotFound)
	}

	return tag, nil
}

// DeleteTag deletes the tag with the given ID.
// Returns TagNotFound if no tag with that ID exists.
func (s *TagService) DeleteTag(ctx context.Context, id int) errs.AppError {
	deleted, err := s.tags.DeleteTag(ctx, id)
	if err != nil {
		return errs.InternalError(err)
	}
	if !deleted {
		return errs.TagNotFound
	}

	return nil
}
