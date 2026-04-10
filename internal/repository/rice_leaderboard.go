package repository

import (
	"context"
	"fmt"
	"ricehub/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UpsertRiceLeaderboard(tx pgx.Tx, period models.LeaderboardPeriod) error {
	const query = `
	INSERT INTO rice_leaderboard (rice_id, period, score, position, snapshot_at)
	WITH scores AS (
		SELECT
			r.id AS rice_id,
			(
				COUNT(DISTINCT COALESCE(d.user_id, d.id)) +
				COUNT(DISTINCT s.user_id) +
				COUNT(DISTINCT dp.user_id)
			) AS score,
			r.created_at
		FROM rices r
		LEFT JOIN rice_downloads d ON d.rice_id = r.id
			AND d.downloaded_at >= now() - $2::interval
		LEFT JOIN rice_stars s ON s.rice_id = r.id
			AND s.starred_at >= now() - $2::interval
		LEFT JOIN dotfiles_purchases dp ON dp.rice_id = r.id
			AND dp.purchased_at >= now() - $2::interval
		WHERE r.created_at <= now() - interval '24 hours'
		GROUP BY r.id
	)
	SELECT
		rice_id,
		$1 as period,
		score,
		ROW_NUMBER() OVER (ORDER BY score DESC, created_at DESC) position,
		now() AS snapshot_at
	FROM scores
	LIMIT 20
	ON CONFLICT (rice_id, period)
	DO UPDATE SET
		score = excluded.score,
		position = excluded.position,
		snapshot_at = excluded.snapshot_at
	`
	_, err := tx.Exec(context.Background(), query, period, period.Interval())
	return err
}

func FetchLeaderboard(period models.LeaderboardPeriod, user *uuid.UUID) (models.LeaderboardRices, error) {
	isStarred := "false AS is_starred"
	if user != nil {
		isStarred = `EXISTS (
			SELECT 1 FROM rice_stars rs
			WHERE rs.rice_id = rice.id AND rs.user_id = $2
		) AS is_starred`
	}

	query := fmt.Sprintf(`
	SELECT
		leaderboard.position,
		rice.id, rice.title, rice.slug, rice.created_at, rice.state,
		author.display_name, author.username,
		screenshot.file_path AS thumbnail,
		dotfiles.download_count, dotfiles.type AS dotfiles_type,
		COUNT(DISTINCT stars.user_id) AS star_count,
		COUNT(DISTINCT comments.id) AS comment_count,
		0 AS score,
		array_remove(array_agg(DISTINCT t.name), NULL) AS tags,
		%s
	FROM rice_leaderboard leaderboard
	JOIN rices rice ON rice.id = leaderboard.rice_id
	JOIN users author ON author.id = rice.author_id
	JOIN LATERAL (
		SELECT file_path
		FROM rice_screenshots
		WHERE rice_id = rice.id
		ORDER BY created_at
		LIMIT 1
	) screenshot ON TRUE
	JOIN rice_dotfiles dotfiles ON dotfiles.rice_id = rice.id
	LEFT JOIN rice_stars stars ON stars.rice_id = rice.id
	LEFT JOIN rice_comments comments ON comments.rice_id = rice.id
	LEFT JOIN rice_tag rt ON rt.rice_id = rice.id
	LEFT JOIN tags t ON t.id = rt.tag_id
	WHERE period = $1
	GROUP BY
		leaderboard.position,
		rice.id, rice.state, rice.slug, rice.title, rice.created_at,
		author.display_name, author.username,
		screenshot.file_path,
		dotfiles.download_count, dotfiles.type
	`, isStarred)

	args := []any{period}
	if user != nil {
		args = append(args, user)
	}

	return rowsToStruct[models.LeaderboardRice](query, args...)
}

func CleanupRiceLeaderboard(tx pgx.Tx, period models.LeaderboardPeriod) error {
	const query = `
	DELETE FROM rice_leaderboard
	WHERE
		period = $1 AND
		(snapshot_at < now() - interval '1 hour' OR position > 20)
	`
	_, err := tx.Exec(context.Background(), query, period)
	return err
}
