package services

import (
	"context"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

type ProfileService struct {
	users *repository.UserRepository
	rices *repository.RiceRepository
}

func NewProfileService(
	users *repository.UserRepository,
	rices *repository.RiceRepository,
) *ProfileService {
	return &ProfileService{users, rices}
}

type GetProfileResult struct {
	User  models.User
	Rices models.PartialRices
}

// GetProfileByUsername fetches given user data and rices.
// The caller's userID is optional and used to include user-specific data in result.
// Returns an error if no user with the given username exists.
func (s *ProfileService) GetProfileByUsername(
	ctx context.Context,
	username string,
	callerID *uuid.UUID,
) (GetProfileResult, errs.AppError) {
	var res GetProfileResult

	user, err := s.users.FindUserByUsername(ctx, username)
	if err != nil {
		return res, errs.FromDBError(err, errs.UserNotFound)
	}

	rices, err := s.rices.FetchUserRices(ctx, user.ID, callerID)
	if err != nil {
		return res, errs.InternalError(err)
	}

	res.User = user
	res.Rices = rices
	return res, nil
}
