package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

type LeaderboardService struct{}

func NewLeaderboardService() *LeaderboardService {
	return &LeaderboardService{}
}

// FetchLeaderboard fetches leaderboard entries for the given period.
// The caller's userID is optional and used to include user-specific data in result.
func (s *LeaderboardService) FetchLeaderboard(period models.LeaderboardPeriod, callerID *uuid.UUID) (models.LeaderboardRices, errs.AppError) {
	rices, err := repository.FetchLeaderboard(period, callerID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}
