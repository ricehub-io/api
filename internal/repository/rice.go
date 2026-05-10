package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ricehub-io/api/internal/config"
	"github.com/ricehub-io/api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RiceRepository struct {
	db DBExecutor
}

func NewRiceRepository(db DBExecutor) *RiceRepository {
	return &RiceRepository{db}
}

func (r *RiceRepository) WithTx(tx pgx.Tx) *RiceRepository {
	return &RiceRepository{tx}
}

func buildFetchRicesSql(sortBy string, subsequent bool, withUser bool, reverse bool) string {
	argIdx := 1

	isStarred := "false AS is_starred"
	if withUser {
		isStarred = fmt.Sprintf(`EXISTS (
				SELECT 1 FROM rice_stars rs
				WHERE rs.rice_id = r.id AND rs.user_id = $%d
			) AS is_starred`, argIdx)
		argIdx++
	}

	scoreCol := "0 AS score"
	downloadsJoin := ""
	if sortBy == "trending" {
		scoreCol = `(
				(count(DISTINCT coalesce(rd.user_id, r.id)) + count(DISTINCT s.user_id))
				/ pow(extract(EPOCH FROM (date_trunc('hour', current_timestamp) - r.created_at)) / 3600 + 2, 1.5)
			) AS score`
		downloadsJoin = "LEFT JOIN rice_downloads rd ON rd.rice_id = r.id"
	}

	ord, sign := "DESC", "<"
	if reverse {
		ord, sign = "ASC", ">"
	}

	var where, order string
	switch sortBy {
	case "trending":
		if subsequent {
			where = fmt.Sprintf("WHERE (score, id) %s ($%d, $%d)", sign, argIdx, argIdx+1)
		}
		order = fmt.Sprintf("ORDER BY score %s, id %s", ord, ord)
	case "recent":
		if subsequent {
			where = fmt.Sprintf("WHERE (created_at, id) %s ($%d, $%d)", sign, argIdx, argIdx+1)
		}
		order = fmt.Sprintf("ORDER BY created_at %s, id %s", ord, ord)
	case "downloads":
		if subsequent {
			where = fmt.Sprintf("WHERE (download_count, id) %s ($%d, $%d)", sign, argIdx, argIdx+1)
		}
		order = fmt.Sprintf("ORDER BY download_count %s, id %s", ord, ord)
	case "stars":
		if subsequent {
			where = fmt.Sprintf("WHERE (star_count, id) %s ($%d, $%d)", sign, argIdx, argIdx+1)
		}
		order = fmt.Sprintf("ORDER BY star_count %s, id %s", ord, ord)
	}

	return fmt.Sprintf(`
		WITH ranked AS (
			SELECT
				r.id, r.title, r.slug, r.created_at, r.state,
				u.display_name, u.username,
				p.file_path AS thumbnail,
				count(DISTINCT s.user_id) AS star_count,
				count(DISTINCT c.id) AS comment_count,
				df.download_count, df.type AS dotfiles_type,
				%s,
				array_remove(array_agg(DISTINCT t.name), NULL) AS tags,
				%s
			FROM rices r
			JOIN users u ON u.id = r.author_id
			LEFT JOIN rice_stars s ON s.rice_id = r.id
			LEFT JOIN rice_comments c ON c.rice_id = r.id
			JOIN rice_dotfiles df ON df.rice_id = r.id
			JOIN LATERAL (
				SELECT p.file_path
				FROM rice_screenshots p
				WHERE p.rice_id = r.id
				ORDER BY p.created_at
				LIMIT 1
			) p ON TRUE
			%s
			LEFT JOIN rice_tag rt ON rt.rice_id = r.id
			LEFT JOIN tags t ON t.id = rt.tag_id
			WHERE r.state != 'waiting'
			GROUP BY
				r.id, r.state, r.slug, r.title, r.created_at,
				df.download_count, df.type, u.display_name,
				u.username, p.file_path
		)
		SELECT * FROM ranked %s %s
		LIMIT %d`,
		scoreCol, isStarred, downloadsJoin, where, order, config.Config.App.PaginationLimit)
}

type FindRiceBy uint8

const (
	RiceID FindRiceBy = iota
	SlugAndUsername
)

func buildFindRiceSql(findBy FindRiceBy) string {
	suffix := `
	SELECT
		to_jsonb(base) AS rice,
		to_jsonb(u) AS "user",
		to_jsonb(df) AS dotfiles,
		jsonb_agg(p ORDER BY p.id) AS screenshots,
		count(DISTINCT s.user_id) AS star_count,
		coalesce(bool_or(s.user_id = $1), false) AS is_starred,
		CASE WHEN df.type != 'free'
			THEN (
				CASE WHEN $1 IS NOT NULL
					THEN (
						SELECT EXISTS(
							SELECT 1
							FROM dotfiles_purchases dp
							WHERE dp.user_id = $1 AND dp.rice_id = base.id
						)
					)
					ELSE false
				END
			)
			ELSE true
    	END AS is_owned,
		t.tags
	FROM base
	JOIN users_with_ban_status u ON u.id = base.author_id
	JOIN rice_dotfiles df ON df.rice_id = base.id
	JOIN rice_screenshots p ON p.rice_id = base.id
	LEFT JOIN rice_stars s ON s.rice_id = base.id
	JOIN LATERAL (
		SELECT coalesce(nullif(jsonb_agg(t), '[null]'), '[]') AS tags
		FROM rice_tag rt
		JOIN tags t ON t.id = rt.tag_id
		WHERE rt.rice_id = base.id
	) t on true
	GROUP BY base.*, df.*, u.*, u.id, base.id, df.type, t.tags
	`

	switch findBy {
	case SlugAndUsername:
		return `
		WITH base AS (
			SELECT r.*
			FROM rices r
			JOIN users u ON u.id = r.author_id
			WHERE r.slug = $2 AND u.username = $3
		)
		` + suffix
	default: // fallback to find by rice id
		return `
		WITH base AS (
			SELECT r.*
			FROM rices r
			WHERE r.id = $2
		)
		` + suffix
	}
}

var findRiceSql = buildFindRiceSql(RiceID)
var findRiceBySlugSql = buildFindRiceSql(SlugAndUsername)

const insertScreenshotSql = `
INSERT INTO rice_screenshots (rice_id, file_path)
VALUES ($1, $2)
RETURNING *
`

type Pagination struct {
	// LastID is stored here as string but its validated in the handler to make sure its a valid UUID
	LastID        *string
	LastScore     *float32
	LastCreatedAt *time.Time
	LastDownloads *int
	LastStars     *int
	Reverse       bool
}

func (r *RiceRepository) FetchPageCount(ctx context.Context) (pages float32, err error) {
	const query = "SELECT CEIL(COUNT(*) / $1) FROM rices"
	err = r.db.QueryRow(ctx, query, config.Config.App.PaginationLimit).Scan(&pages)
	return
}

func (r *RiceRepository) UserOwnsRice(ctx context.Context, riceID, userID uuid.UUID) (exists bool, err error) {
	const query = `
	SELECT EXISTS (
		SELECT 1
		FROM rices
		WHERE id = $1 AND author_id = $2
	)
	`
	err = r.db.QueryRow(ctx, query, riceID, userID).Scan(&exists)
	return
}

func (r *RiceRepository) RiceExists(ctx context.Context, riceID string) (exists bool, err error) {
	const query = "SELECT EXISTS (SELECT 1 FROM rices WHERE id = $1)"
	err = r.db.QueryRow(ctx, query, riceID).Scan(&exists)
	return
}

func (r *RiceRepository) FetchTrendingRices(ctx context.Context, pag *Pagination, userID *uuid.UUID) ([]models.PartialRice, error) {
	subsequent := pag.LastScore != nil
	query := buildFetchRicesSql("trending", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, *pag.LastScore, pag.LastID)
	}

	return rowsToStruct[models.PartialRice](ctx, r.db, query, args...)
}

func (r *RiceRepository) FetchRecentRices(ctx context.Context, pag *Pagination, userID *uuid.UUID) ([]models.PartialRice, error) {
	subsequent := pag.LastCreatedAt != nil
	query := buildFetchRicesSql("recent", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, *pag.LastCreatedAt, pag.LastID)
	}

	return rowsToStruct[models.PartialRice](ctx, r.db, query, args...)
}

func (r *RiceRepository) FetchMostDownloadedRices(ctx context.Context, pag *Pagination, userID *uuid.UUID) ([]models.PartialRice, error) {
	subsequent := pag.LastDownloads != nil
	query := buildFetchRicesSql("downloads", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, *pag.LastDownloads, pag.LastID)
	}

	return rowsToStruct[models.PartialRice](ctx, r.db, query, args...)
}

func (r *RiceRepository) FetchMostStarredRices(
	ctx context.Context,
	pag *Pagination,
	userID *uuid.UUID,
) ([]models.PartialRice, error) {
	subsequent := pag.LastStars != nil
	query := buildFetchRicesSql("stars", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, *pag.LastStars, pag.LastID)
	}

	return rowsToStruct[models.PartialRice](ctx, r.db, query, args...)
}

func (r *RiceRepository) FetchWaitingRices(ctx context.Context) (models.PartialRices, error) {
	const query = `
	SELECT
    	r.id, r.title, r.slug, r.created_at, r.state,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		0 AS star_count,
		0 AS comment_count,
		0 AS download_count, df.type AS dotfiles_type,
		0 AS score,
		false AS is_starred,
		array_remove(array_agg(DISTINCT t.name), NULL) AS tags
	FROM rices r
	JOIN users u ON u.id = r.author_id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_screenshots p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	LEFT JOIN rice_tag rt ON rt.rice_id = r.id
	LEFT JOIN tags t on t.id = rt.tag_id
	WHERE r.state = 'waiting'
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		u.display_name, u.username, p.file_path,
		df.type
	ORDER BY r.created_at DESC
	`

	return rowsToStruct[models.PartialRice](ctx, r.db, query)
}

func (r *RiceRepository) FetchRiceScreenshotCount(ctx context.Context, riceID uuid.UUID) (count int, err error) {
	const query = "SELECT count(*) FROM rice_screenshots WHERE rice_id = $1"
	err = r.db.QueryRow(ctx, query, riceID).Scan(&count)
	return
}

func (r *RiceRepository) FindRiceByID(
	ctx context.Context,
	userID *uuid.UUID,
	riceID uuid.UUID,
) (models.RiceWithRelations, error) {
	return rowToStruct[models.RiceWithRelations](ctx, r.db, findRiceSql, userID, riceID)
}

func (r *RiceRepository) FindRiceBySlug(
	ctx context.Context,
	userID *uuid.UUID,
	slug, username string,
) (models.RiceWithRelations, error) {
	return rowToStruct[models.RiceWithRelations](ctx, r.db, findRiceBySlugSql, userID, slug, username)
}

func (r *RiceRepository) FetchUserRices(
	ctx context.Context,
	userID uuid.UUID,
	callerID *uuid.UUID,
) (models.PartialRices, error) {
	where := "WHERE u.id = $1"
	if callerID == nil || userID != *callerID {
		where += " AND r.state = 'accepted'"
	}

	query := `
	SELECT
		r.id, r.title, r.slug, r.created_at, r.state,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		count(DISTINCT s.user_id) AS star_count,
		count(DISTINCT c.id) AS comment_count,
		df.download_count, df.type AS dotfiles_type,
		0 AS score,
		EXISTS (
			SELECT 1
			FROM rice_stars rs
			WHERE rs.rice_id = r.id AND rs.user_id = $2
		) AS is_starred,
		array_remove(array_agg(DISTINCT t.name), NULL) AS tags
	FROM rices r
	JOIN users u ON u.id = r.author_id
	LEFT JOIN rice_stars s ON s.rice_id = r.id
	LEFT JOIN rice_comments c ON c.rice_id = r.id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_screenshots p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	LEFT JOIN rice_tag rt ON rt.rice_id = r.id
	LEFT JOIN tags t on t.id = rt.tag_id
	` + where + `
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		df.download_count, df.type, u.display_name,
		u.username, p.file_path
	ORDER BY r.created_at DESC, r.id DESC
	`

	return rowsToStruct[models.PartialRice](ctx, r.db, query, userID, callerID)
}

func (r *RiceRepository) FetchUserPurchasedRices(ctx context.Context, userID uuid.UUID) (models.PartialRices, error) {
	const query = `
	SELECT
		r.id, r.title, r.slug, r.created_at, r.state,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		count(DISTINCT s.user_id) AS star_count,
		count(DISTINCT c.id) AS comment_count,
		df.download_count, df.type AS dotfiles_type,
		0 AS score,
		EXISTS (
			SELECT 1
			FROM rice_stars rs
			WHERE rs.rice_id = r.id AND rs.user_id = $1
		) AS is_starred,
		array_remove(array_agg(DISTINCT t.name), NULL) AS tags
	FROM dotfiles_purchases dp
	JOIN rices r ON r.id = dp.rice_id
	JOIN users u ON u.id = r.author_id
	LEFT JOIN rice_stars s ON s.rice_id = r.id
	LEFT JOIN rice_comments c ON c.rice_id = r.id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_screenshots p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	LEFT JOIN rice_tag rt ON rt.rice_id = r.id
	LEFT JOIN tags t on t.id = rt.tag_id
	WHERE dp.user_id = $1
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		df.download_count, df.type, u.display_name,
		u.username, p.file_path
	ORDER BY r.created_at DESC, r.id DESC
	`

	return rowsToStruct[models.PartialRice](ctx, r.db, query, userID)
}

func (r *RiceRepository) FindRiceWithDotfilesByID(ctx context.Context, riceID uuid.UUID) (models.RiceWithDotfiles, error) {
	const query = `
	SELECT to_jsonb(rices) AS rice, to_jsonb(dotfiles) AS dotfiles
	FROM rices
	JOIN rice_dotfiles dotfiles ON dotfiles.rice_id = rices.id
	WHERE rices.id = $1
	LIMIT 1
	`
	return rowToStruct[models.RiceWithDotfiles](ctx, r.db, query, riceID)
}

func (r *RiceRepository) InsertRice(
	ctx context.Context,
	authorID uuid.UUID,
	title, slug, description string,
	autoAccept bool,
) (models.Rice, error) {
	const query = `
	INSERT INTO rices (author_id, title, slug, description, state)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING *
	`
	state := models.Waiting
	if autoAccept {
		state = models.Accepted
	}

	return rowToStruct[models.Rice](ctx, r.db, query, authorID, title, slug, description, state)
}

func (r *RiceRepository) InsertRiceScreenshot(
	ctx context.Context,
	riceID uuid.UUID,
	scrPath string,
) (models.RiceScreenshot, error) {
	return rowToStruct[models.RiceScreenshot](ctx, r.db, insertScreenshotSql, riceID, scrPath)
}

func (r *RiceRepository) InsertRiceScreenshotTx(ctx context.Context, riceID uuid.UUID, scrPath string) error {
	_, err := r.db.Exec(ctx, insertScreenshotSql, riceID, scrPath)
	return err
}

func (r *RiceRepository) InsertRiceDownload(ctx context.Context, riceID uuid.UUID, userID *uuid.UUID) error {
	const query = "INSERT INTO rice_downloads (rice_id, user_id) VALUES ($1, $2)"
	_, err := r.db.Exec(ctx, query, riceID, userID)
	return err
}

func (r *RiceRepository) InsertRiceStar(ctx context.Context, riceID, userID uuid.UUID) error {
	const query = "INSERT INTO rice_stars (rice_id, user_id) VALUES ($1, $2)"
	_, err := r.db.Exec(ctx, query, riceID, userID)
	return err
}

func (r *RiceRepository) UpdateRice(
	ctx context.Context,
	riceID uuid.UUID,
	title, description *string,
) error {
	query := "UPDATE rices SET"
	args := []any{riceID}

	argIdx := 2
	if title != nil {
		query += fmt.Sprintf(" title = $%d", argIdx)
		args = append(args, *title)
		argIdx++
	}
	if description != nil {
		if len(args) > 1 {
			query += ","
		}
		query += fmt.Sprintf(" description = $%d", argIdx)
		args = append(args, *description)
	}

	query += " WHERE id = $1"

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *RiceRepository) UpdateRiceState(
	ctx context.Context,
	riceID uuid.UUID,
	newState models.RiceState,
) error {
	const query = "UPDATE rices SET state = $1 WHERE id = $2"
	_, err := r.db.Exec(ctx, query, newState, riceID)
	return err
}

func (r *RiceRepository) DeleteRiceScreenshot(
	ctx context.Context,
	riceID, screenshotID uuid.UUID,
) (bool, error) {
	const query = "DELETE FROM rice_screenshots WHERE id = $1 AND rice_id = $2"
	cmd, err := r.db.Exec(ctx, query, screenshotID, riceID)
	return cmd.RowsAffected() == 1, err
}

func (r *RiceRepository) DeleteRiceStar(
	ctx context.Context,
	riceID, userID uuid.UUID,
) error {
	const query = "DELETE FROM rice_stars WHERE rice_id = $1 AND user_id = $2"
	_, err := r.db.Exec(ctx, query, riceID, userID)
	return err
}

func (r *RiceRepository) DeleteRice(ctx context.Context, riceID uuid.UUID) (bool, error) {
	const query = "DELETE FROM rices WHERE id = $1"
	cmd, err := r.db.Exec(ctx, query, riceID)
	return cmd.RowsAffected() > 0, err
}
