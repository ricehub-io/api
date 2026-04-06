package services

import (
	"errors"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateTag inserts a new tag with the given name.
// Returns an error if a tag with that name already exists.
func CreateTag(name string) (models.Tag, errs.AppError) {
	tag, err := repository.InsertTag(name)
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
func ListTags() (models.Tags, errs.AppError) {
	tags, err := repository.FetchTags()
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return tags, nil
}

// UpdateTag updates the name of the tag with the given ID.
// Returns TagNotFound if no tag with that ID exists.
func UpdateTag(id int, name string) (models.Tag, errs.AppError) {
	tag, err := repository.UpdateTag(id, name)
	if err != nil {
		return tag, errs.FromDBError(err, errs.TagNotFound)
	}

	return tag, nil
}

// DeleteTag deletes the tag with the given ID.
// Returns TagNotFound if no tag with that ID exists.
func DeleteTag(id int) errs.AppError {
	deleted, err := repository.DeleteTag(id)
	if err != nil {
		return errs.InternalError(err)
	}
	if !deleted {
		return errs.TagNotFound
	}

	return nil
}
