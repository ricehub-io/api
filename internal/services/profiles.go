package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

type ProfileService struct{}

func NewProfileService() *ProfileService {
	return &ProfileService{}
}

type GetProfileResult struct {
	User  models.User
	Rices models.PartialRices
}

// GetProfileByUsername fetches given user data and rices.
// The caller's userID is optional and used to include user-specific data in result.
// Returns an error if no user with the given username exists.
func (s *ProfileService) GetProfileByUsername(username string, callerID *uuid.UUID) (GetProfileResult, errs.AppError) {
	var res GetProfileResult

	user, err := repository.FindUserByUsername(username)
	if err != nil {
		return res, errs.FromDBError(err, errs.UserNotFound)
	}

	rices, err := repository.FetchUserRices(user.ID, callerID)
	if err != nil {
		return res, errs.InternalError(err)
	}

	res.User = user
	res.Rices = rices
	return res, nil
}
