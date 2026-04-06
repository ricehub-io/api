package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
)

func ServiceStatistics() (models.ServiceStatistics, errs.AppError) {
	stats, err := repository.FetchServiceStatistics()
	if err != nil {
		return stats, errs.InternalError(err)
	}

	return stats, nil
}
