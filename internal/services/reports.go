package services

import (
	"context"
	"errors"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"

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

// ListReports returns all reports ordered by creation date.
func (s *ReportService) ListReports(ctx context.Context) ([]models.ReportWithUser, errs.AppError) {
	reports, err := s.reports.FetchReports(ctx)
	if err != nil {
		return reports, errs.InternalError(err)
	}
	return reports, nil
}

// GetReportByID fetches a report by ID. Returns an error if not found.
func (s *ReportService) GetReportByID(
	ctx context.Context,
	reportID uuid.UUID,
) (models.ReportWithUser, errs.AppError) {
	report, err := s.reports.FindReportByID(ctx, reportID)
	if err != nil {
		return report, errs.FromDBError(err, errs.ReportNotFound)
	}
	return report, nil
}

// CloseReport updates given report's status to closed.
// Returns error if report doesn't exist.
func (s *ReportService) CloseReport(ctx context.Context, reportID uuid.UUID) errs.AppError {
	updated, err := s.reports.CloseReport(ctx, reportID, true)
	if err != nil {
		return errs.InternalError(err)
	}
	if !updated {
		return errs.ReportNotFound
	}
	return nil
}
