package services

import (
	"context"
	"errors"
	"ricehub/internal/errs"
	"ricehub/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type ReportService struct {
	reports *repository.ReportRepository
}

func NewReportService(reports *repository.ReportRepository) *ReportService {
	return &ReportService{reports}
}

// CreateReport inserts a new report for one given resource.
// Returns an error if no resource with given id exists or user has already reported it.
func (s *ReportService) CreateReport(
	ctx context.Context,
	userID uuid.UUID,
	riceID, commentID *string,
	reason string,
) (uuid.UUID, errs.AppError) {
	repID, err := s.reports.InsertReport(ctx, userID, reason, riceID, commentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.ForeignKeyViolation:
				return repID, errs.ResourceNotFound
			case pgerrcode.UniqueViolation:
				return repID, errs.AlreadyReported
			}
		}
		return repID, errs.InternalError(err)
	}

	return repID, nil
}
