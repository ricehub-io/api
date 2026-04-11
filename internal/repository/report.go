package repository

import (
	"context"
	"ricehub/internal/models"

	"github.com/google/uuid"
)

type ReportRepository struct {
	db DBExecutor
}

func NewReportRepository(db DBExecutor) *ReportRepository {
	return &ReportRepository{db}
}

func (r *ReportRepository) InsertReport(
	ctx context.Context,
	reporterID uuid.UUID,
	reason string,
	riceID, commentID *string,
) (id uuid.UUID, err error) {
	const query = `
	INSERT INTO reports (reporter_id, reason, rice_id, comment_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`
	err = r.db.QueryRow(ctx, query, reporterID, reason, riceID, commentID).Scan(&id)
	return
}

func (r *ReportRepository) FetchReports(ctx context.Context) ([]models.ReportWithUser, error) {
	const query = `
	SELECT r.*, u.display_name, u.username
	FROM reports r
	JOIN users u ON u.id = r.reporter_id
	ORDER BY r.created_at DESC
	`

	return rowsToStruct[models.ReportWithUser](ctx, r.db, query)
}

func (r *ReportRepository) FindReportByID(ctx context.Context, reportID uuid.UUID) (models.ReportWithUser, error) {
	const query = `
	SELECT r.*, u.display_name, u.username
	FROM reports r
	JOIN users u ON u.id = r.reporter_id
	WHERE r.id = $1
	`

	return rowToStruct[models.ReportWithUser](ctx, r.db, query, reportID)
}

func (r *ReportRepository) CloseReport(ctx context.Context, reportID uuid.UUID, newState bool) (bool, error) {
	const query = "UPDATE reports SET is_closed = $1 WHERE id = $2"
	cmd, err := r.db.Exec(ctx, query, newState, reportID)
	return cmd.RowsAffected() == 1, err
}
