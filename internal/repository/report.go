package repository

import (
	"context"
	"ricehub/internal/models"

	"github.com/google/uuid"
)

func InsertReport(reporterID string, reason string, riceID *string, commentID *string) (id uuid.UUID, err error) {
	const query = `
	INSERT INTO reports (reporter_id, reason, rice_id, comment_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`

	err = db.QueryRow(context.Background(), query, reporterID, reason, riceID, commentID).Scan(&id)
	return
}

func FetchReports() ([]models.ReportWithUser, error) {
	const query = `
	SELECT r.*, u.display_name, u.username
	FROM reports r
	JOIN users u ON u.id = r.reporter_id
	ORDER BY r.created_at DESC
	`

	return rowsToStruct[models.ReportWithUser](query)
}

func FindReportByID(reportID string) (models.ReportWithUser, error) {
	const query = `
	SELECT r.*, u.display_name, u.username
	FROM reports r
	JOIN users u ON u.id = r.reporter_id
	WHERE r.id = $1
	`

	return rowToStruct[models.ReportWithUser](query, reportID)
}

func CloseReport(reportID string, newState bool) (bool, error) {
	const query = "UPDATE reports SET is_closed = $1 WHERE id = $2"
	cmd, err := db.Exec(context.Background(), query, newState, reportID)
	return cmd.RowsAffected() == 1, err
}
