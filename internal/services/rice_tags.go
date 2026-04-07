package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

// AddRiceTags attaches the given tag IDs to a rice.
// Enforces ownership check before proceeding.
func AddRiceTags(riceID, userID uuid.UUID, isAdmin bool, tags []int) errs.AppError {
	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return err
	}
	if err := repository.InsertRiceTags(riceID, tags); err != nil {
		return errs.FromDBError(err, errs.RiceNotFound)
	}
	return nil
}

// RemoveRiceTags detaches the given tag IDs from a rice.
// Enforces ownership check before proceeding.
func RemoveRiceTags(riceID, userID uuid.UUID, isAdmin bool, tags []int) errs.AppError {
	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return err
	}
	if err := repository.DeleteRiceTags(riceID, tags); err != nil {
		return errs.InternalError(err)
	}
	return nil
}
