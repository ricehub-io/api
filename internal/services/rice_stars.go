package services

import (
	"context"
	"errors"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type RiceStarService struct {
	rices *repository.RiceRepository
}

func NewRiceStarService(rices *repository.RiceRepository) *RiceStarService {
	return &RiceStarService{rices}
}

// CreateRiceStar adds a star to a rice for the given user.
// Silently succeeds if the user has already starred the rice.
func (s *RiceStarService) CreateRiceStar(
	ctx context.Context,
	riceID, userID uuid.UUID,
) errs.AppError {
	if err := s.rices.InsertRiceStar(ctx, riceID, userID); err != nil {
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
func (s *RiceStarService) DeleteRiceStar(
	ctx context.Context,
	riceID, userID uuid.UUID,
) errs.AppError {
	if err := s.rices.DeleteRiceStar(ctx, riceID, userID); err != nil {
		return errs.InternalError(err)
	}
	return nil
}
