package services

import (
	"errors"
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateReport inserts a new report for one given resource.
// Returns an error if no resource with given id exists or user has already reported it.
func CreateReport(userID uuid.UUID, riceID, commentID *string, reason string) (uuid.UUID, errs.AppError) {
	reportID, err := repository.InsertReport(userID, reason, riceID, commentID)
	if err != nil {
		var zero uuid.UUID

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.ForeignKeyViolation:
				return zero, errs.ResourceNotFound
			case pgerrcode.UniqueViolation:
				return zero, errs.AlreadyReported
			}
		}

		return zero, errs.InternalError(err)
	}

	return reportID, nil
}
