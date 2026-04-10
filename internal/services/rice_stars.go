package services

import (
	"errors"
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type RiceStarService struct{}

func NewRiceStarService() *RiceStarService {
	return &RiceStarService{}
}

// CreateRiceStar adds a star to a rice for the given user.
// Silently succeeds if the user has already starred the rice.
func (s *RiceStarService) CreateRiceStar(riceID, userID string) errs.AppError {
	if err := repository.InsertRiceStar(riceID, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return nil
			case pgerrcode.ForeignKeyViolation:
				return errs.RiceNotFound
			}
		}
		return errs.InternalError(err)
	}
	return nil
}

// DeleteRiceStar removes a star from a rice for the given user.
func (s *RiceStarService) DeleteRiceStar(riceID, userID string) errs.AppError {
	if err := repository.DeleteRiceStar(riceID, userID); err != nil {
		return errs.InternalError(err)
	}
	return nil
}
