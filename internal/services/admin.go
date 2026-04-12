package services

import (
	"context"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
)

type AdminService struct {
	repo *repository.AdminRepository
}

func NewAdminService(repo *repository.AdminRepository) *AdminService {
	return &AdminService{repo}
}

// ServiceStatistics fetches latest service statistics from database.
func (s *AdminService) ServiceStatistics(ctx context.Context) (models.ServiceStatistics, errs.AppError) {
	stats, err := s.repo.FetchServiceStatistics(ctx)
	if err != nil {
		return stats, errs.InternalError(err)
	}

	return stats, nil
}
