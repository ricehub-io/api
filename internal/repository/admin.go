package repository

import (
	"context"
	"ricehub/internal/models"
)

type AdminRepository struct {
	db DBExecutor
}

func NewAdminRepository(db DBExecutor) *AdminRepository {
	return &AdminRepository{db}
}

func (r *AdminRepository) FetchServiceStatistics(ctx context.Context) (models.ServiceStatistics, error) {
	const query = `
	WITH user_stats AS (
	    SELECT
	        COUNT(*) AS user_count,
	        COUNT(*) FILTER (
	            WHERE created_at >= NOW() - INTERVAL '24 hours'
	        ) AS user_24h_count
	    FROM users
	),
	rice_stats AS (
	    SELECT
	        COUNT(*) AS rice_count,
	        COUNT(*) FILTER (
	            WHERE created_at >= NOW() - INTERVAL '24 hours'
	        ) AS rice_24h_count
	    FROM rices
	),
	comment_stats AS (
	    SELECT
	        COUNT(*) AS comment_count,
	        COUNT(*) FILTER (
	            WHERE created_at >= NOW() - INTERVAL '24 hours'
	        ) AS comment_24h_count
	    FROM rice_comments
	),
	report_stats AS (
	    SELECT
	        COUNT(*) AS report_count,
	        COUNT(*) FILTER (
	            WHERE is_closed = false
	        ) AS open_report_count
	    FROM reports
	)
	SELECT *
	FROM user_stats, rice_stats, comment_stats, report_stats
	`

	return rowToStruct[models.ServiceStatistics](ctx, r.db, query)
}
