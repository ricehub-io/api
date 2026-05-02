package services

import (
	"context"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"
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
