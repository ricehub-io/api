package services

import (
	"context"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"

	"github.com/google/uuid"
)

type LeaderboardService struct {
	repo *repository.RiceLeaderboardRepository
}

func NewLeaderboardService(repo *repository.RiceLeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo}
}

// FetchLeaderboard fetches leaderboard entries for the given period.
// The caller's userID is optional and used to include user-specific data in result.
func (s *LeaderboardService) FetchLeaderboard(
	ctx context.Context,
	period models.LeaderboardPeriod,
	callerID *uuid.UUID,
) (models.LeaderboardRices, errs.AppError) {
	rices, err := s.repo.FetchLeaderboard(ctx, period, callerID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}
