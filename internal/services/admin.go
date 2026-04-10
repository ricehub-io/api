package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
)

// ServiceStatistics fetches latest service statistics from database.
func ServiceStatistics() (models.ServiceStatistics, errs.AppError) {
	stats, err := repository.FetchServiceStatistics()
	if err != nil {
		return models.ServiceStatistics{}, errs.InternalError(err)
	}

	return stats, nil
}
