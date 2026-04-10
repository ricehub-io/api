package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
)

type AdminService struct{}

func NewAdminService() *AdminService {
	return &AdminService{}
}

// ServiceStatistics fetches latest service statistics from database.
func (s *AdminService) ServiceStatistics() (models.ServiceStatistics, errs.AppError) {
	stats, err := repository.FetchServiceStatistics()
	if err != nil {
		return models.ServiceStatistics{}, errs.InternalError(err)
	}

	return stats, nil
}
