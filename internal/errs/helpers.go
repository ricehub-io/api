package errs

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// FromDBError checks if error is of type pgx.ErrNoRows and returns
// provided notFoundErr, otherwise wraps the err into internal error
func FromDBError(err error, notFoundErr AppError) AppError {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundErr
	}

	return InternalError(err)
}
