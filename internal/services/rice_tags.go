package services

import (
	"context"
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

type RiceTagService struct {
	rices    *repository.RiceRepository
	riceTags *repository.RiceTagRepository
}

func NewRiceTagService(
	rices *repository.RiceRepository,
	riceTags *repository.RiceTagRepository,
) *RiceTagService {
	return &RiceTagService{rices, riceTags}
}

// AddRiceTags attaches the given tag IDs to a rice.
// Enforces ownership check before proceeding.
func (s *RiceTagService) AddRiceTags(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	tags []int,
) errs.AppError {
	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}
	if err := s.riceTags.InsertRiceTags(ctx, riceID, tags); err != nil {
		return errs.FromDBError(err, errs.RiceNotFound)
	}
	return nil
}

// RemoveRiceTags detaches the given tag IDs from a rice.
// Enforces ownership check before proceeding.
func (s *RiceTagService) RemoveRiceTags(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	tags []int,
) errs.AppError {
	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}
	if err := s.riceTags.DeleteRiceTags(ctx, riceID, tags); err != nil {
		return errs.InternalError(err)
	}
	return nil
}
