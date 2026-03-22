// TODO: move all dotfiles-related queries to rice_dotfiles.go file

package repository

import (
	"context"
	"fmt"
	"ricehub/src/models"
	"ricehub/src/utils"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// idk if thats how you're supposed to write golang code but whatever
// let a man be happy after going through Rust horror
const hasUserRiceSql = `
SELECT EXISTS (
	SELECT 1
	FROM rices
	WHERE id = $1 AND author_id = $2 AND state = 'accepted'
)
`

// FIXME: score has to be fetched for all responses even when not needed because PartialRice requires it
func buildFetchRicesSql(sortBy string, subsequent bool, withUser bool, reverse bool) string {
	argCount := 1

	baseSelect := `
		WITH ranked AS (
			SELECT
				r.id, r.title, r.slug, r.created_at, r.state,
				u.display_name, u.username,
				p.file_path AS thumbnail,
				count(DISTINCT s.user_id) AS star_count,
				count(DISTINCT c.id) AS comment_count,
				df.download_count, df.type AS dotfiles_type,
				(
					(df.download_count + count(DISTINCT s.user_id))
					/ pow(extract(EPOCH FROM (date_trunc('hour', current_timestamp) - r.created_at)) / 3600 + 2, 1.5)
				) AS score,
	`

	userSelect := "false AS is_starred"
	if withUser {
		userSelect = fmt.Sprintf(`
				EXISTS (
					SELECT 1
					FROM rice_stars rs
					WHERE rs.rice_id = r.id AND rs.user_id = $%v
				) AS is_starred
		`, argCount)
		argCount += 1
	}

	base := `
			FROM rices r
			JOIN users u ON u.id = r.author_id
			LEFT JOIN rice_stars s ON s.rice_id = r.id
			LEFT JOIN rice_comments c ON c.rice_id = r.id
			JOIN rice_dotfiles df ON df.rice_id = r.id
			JOIN LATERAL (
				SELECT p.file_path
				FROM rice_previews p
				WHERE p.rice_id = r.id
				ORDER BY p.created_at
				LIMIT 1
			) p ON TRUE
			WHERE r.state != 'waiting'
			GROUP BY
				r.id, r.slug, r.title, r.created_at,
				df.download_count, df.type, u.display_name,
				u.username, p.file_path
		)
	`

	mainSelect := " SELECT * FROM ranked"
	where := ""
	order := ""

	ord := "DESC"
	if reverse {
		ord = "ASC"
	}

	sign := "<"
	if reverse {
		sign = ">"
	}

	switch sortBy {
	case "trending":
		if subsequent {
			where = fmt.Sprintf(" WHERE (score, id) %v ($%v, $%v)", sign, argCount, argCount+1)
			argCount += 2
		}

		order = fmt.Sprintf(" ORDER BY score %v, id %v", ord, ord)
	case "recent":
		if subsequent {
			where = fmt.Sprintf(" WHERE (created_at, id) %v ($%v, $%v)", sign, argCount, argCount+1)
			argCount += 2
		}

		order = fmt.Sprintf(" ORDER BY created_at %v, id %v", ord, ord)
	case "downloads":
		if subsequent {
			where = fmt.Sprintf(" WHERE (download_count, id) %v ($%v, $%v)", sign, argCount, argCount+1)
			argCount += 2
		}

		order = fmt.Sprintf(" ORDER BY download_count %v, id %v", ord, ord)
	case "stars":
		if subsequent {
			where = fmt.Sprintf(" WHERE (star_count, id) %v ($%v, $%v)", sign, argCount, argCount+1)
			argCount += 2
		}

		order = fmt.Sprintf(" ORDER BY star_count %v, id %v", ord, ord)
	}

	return baseSelect + userSelect + base + mainSelect + where + order + fmt.Sprintf(" LIMIT %v", utils.Config.PaginationLimit)
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
		jsonb_agg(to_jsonb(p) ORDER BY p.id) AS previews,
		count(DISTINCT s.user_id) AS star_count,
		coalesce(bool_or(s.user_id = $1), false) AS is_starred,
		CASE WHEN df.type != 'free' AND u.id != $1
			THEN (
				SELECT EXISTS(
					SELECT 1
					FROM dotfiles_purchases dp
					WHERE dp.user_id = $1 AND dp.rice_id = base.id
				)
			)
			ELSE true
    	END AS is_owned
	FROM base
	JOIN users_with_ban_status u ON u.id = base.author_id
	JOIN rice_dotfiles df ON df.rice_id = base.id
	JOIN rice_previews p ON p.rice_id = base.id
	LEFT JOIN rice_stars s ON s.rice_id = base.id
	GROUP BY base.*, df.*, u.*, u.id, base.id, df.type
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

const insertRiceSql = `
INSERT INTO rices (author_id, title, slug, description, state)
VALUES ($1, $2, $3, $4, $5)
RETURNING *
`
const insertScreenshotSql = `
INSERT INTO rice_previews (rice_id, file_path)
VALUES ($1, $2)
RETURNING *
`
const insertDotfilesSql = `
INSERT INTO rice_dotfiles (rice_id, file_path, file_size, type, price, product_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *
`
const insertStarSql = `
INSERT INTO rice_stars (rice_id, user_id)
VALUES ($1, $2)
`
const updateDotfilesSql = `
UPDATE rice_dotfiles
SET file_path = $2, file_size = $3
WHERE rice_id = $1
RETURNING *
`
const incrementDownloadsSql = `
UPDATE rice_dotfiles df
SET download_count = download_count + 1
FROM rices r
WHERE r.id = $1 AND r.id = df.rice_id
RETURNING df.file_path
`
const deleteScreenshotSql = `
DELETE FROM rice_previews
WHERE id = $1 AND rice_id = $2
`

type Pagination struct {
	// LastID is stored here as string but its validated in the handler to make sure its a valid UUID
	LastID        *string
	LastScore     float32
	LastCreatedAt time.Time
	LastDownloads int
	LastStars     int
	Reverse       bool
}

func FetchPageCount() (pages float32, err error) {
	query := "SELECT CEIL(COUNT(*) / $1) FROM rices"
	err = db.QueryRow(context.Background(), query, utils.Config.PaginationLimit).Scan(&pages)
	return
}

func HasUserRiceWithId(riceID string, userID string) (exists bool, err error) {
	err = db.QueryRow(context.Background(), hasUserRiceSql, riceID, userID).Scan(&exists)
	return
}

func DoesRiceExist(riceID string) (exists bool, err error) {
	err = db.QueryRow(
		context.Background(),
		"SELECT EXISTS (SELECT 1 FROM rices WHERE id = $1)",
		riceID,
	).Scan(&exists)
	return
}

func FetchTrendingRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	subsequent := pag.LastScore != -1
	query := buildFetchRicesSql("trending", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, pag.LastScore, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchRecentRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	subsequent := !pag.LastCreatedAt.IsZero()

	query := buildFetchRicesSql("recent", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, pag.LastCreatedAt, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchMostDownloadedRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	subsequent := pag.LastDownloads != -1

	query := buildFetchRicesSql("downloads", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, pag.LastDownloads, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchMostStarredRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	subsequent := pag.LastStars != -1
	query := buildFetchRicesSql("stars", subsequent, userID != nil, pag.Reverse)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if subsequent {
		args = append(args, pag.LastStars, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchWaitingRices() ([]models.PartialRice, error) {
	const query = `
	SELECT
    	r.id, r.title, r.slug, r.created_at, r.state,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		0 AS star_count,
		0 AS comment_count,
		0 AS download_count, df.type AS dotfiles_type,
		0 AS score,
		false AS is_starred
	FROM rices r
	JOIN users u ON u.id = r.author_id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_previews p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	WHERE r.state = 'waiting'
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		u.display_name, u.username, p.file_path,
		df.type
	ORDER BY r.created_at DESC
	`

	return rowsToStruct[models.PartialRice](query)
}

func FetchRiceScreenshotCount(riceID string) (count int, err error) {
	err = db.QueryRow(
		context.Background(),
		"SELECT count(*) FROM rice_previews WHERE rice_id = $1",
		riceID,
	).Scan(&count)
	return
}

func FetchRiceDotfilesPath(riceID string) (*string, error) {
	var filePath *string
	query := "SELECT file_path FROM rice_dotfiles WHERE rice_id = $1"
	err := db.QueryRow(context.Background(), query, riceID).Scan(&filePath)
	return filePath, err
}

func FindRiceByID(userID *string, riceID string) (r models.RiceWithRelations, err error) {
	r, err = rowToStruct[models.RiceWithRelations](findRiceSql, userID, riceID)
	return
}

func FindRiceBySlug(userID *string, slug string, username string) (r models.RiceWithRelations, err error) {
	r, err = rowToStruct[models.RiceWithRelations](findRiceBySlugSql, userID, slug, username)
	return
}

func FetchUserRices(userID string, callerID *string) (r []models.PartialRice, err error) {
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
		) AS is_starred
	FROM rices r
	JOIN users u ON u.id = r.author_id
	LEFT JOIN rice_stars s ON s.rice_id = r.id
	LEFT JOIN rice_comments c ON c.rice_id = r.id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_previews p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	` + where + `
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		df.download_count, df.type, u.display_name,
		u.username, p.file_path
	ORDER BY r.created_at DESC, r.id DESC
	`

	r, err = rowsToStruct[models.PartialRice](query, userID, callerID)
	return
}

func FetchUserPurchasedRices(userID string) (r []models.PartialRice, err error) {
	const query = `
	SELECT
		r.id, r.title, r.slug, r.created_at, r.state,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		count(DISTINCT s.user_id) AS star_count,
		count(DISTINCT c.id) AS comment_count,
		df.download_count, df.type AS dotfiles_type,
		0 AS score,
		false AS is_starred
	FROM dotfiles_purchases dp
	JOIN rices r ON r.id = dp.rice_id
	JOIN users u ON u.id = r.author_id
	LEFT JOIN rice_stars s ON s.rice_id = r.id
	LEFT JOIN rice_comments c ON c.rice_id = r.id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_previews p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	WHERE dp.user_id = $1
	GROUP BY
		r.id, r.slug, r.title, r.created_at,
		df.download_count, df.type, u.display_name,
		u.username, p.file_path
	ORDER BY r.created_at DESC, r.id DESC
	`

	r, err = rowsToStruct[models.PartialRice](query, userID)
	return
}

func FindRiceWithDotfilesByID(tx pgx.Tx, riceID string) (r models.RiceWithDotfiles, err error) {
	const query = `
	SELECT to_jsonb(rices) AS rice, to_jsonb(dotfiles) AS dotfiles
	FROM rices
	JOIN rice_dotfiles dotfiles ON dotfiles.rice_id = rices.id
	WHERE rices.id = $1
	LIMIT 1
	`
	r, err = txRowToStruct[models.RiceWithDotfiles](tx, query, riceID)
	return
}

func InsertRice(tx pgx.Tx, authorID string, title string, slug string, description string, autoAccept bool) (rice models.Rice, err error) {
	state := models.Waiting
	if autoAccept {
		state = models.Accepted
	}

	rice, err = txRowToStruct[models.Rice](tx, insertRiceSql, authorID, title, slug, description, state)

	return
}

func InsertRiceScreenshot(riceID string, scrPath string) (p models.RiceScreenshot, err error) {
	p, err = rowToStruct[models.RiceScreenshot](insertScreenshotSql, riceID, scrPath)
	return
}

func InsertRiceScreenshotTx(tx pgx.Tx, riceID uuid.UUID, scrPath string) error {
	_, err := tx.Exec(context.Background(), insertScreenshotSql, riceID, scrPath)
	return err
}

func InsertRiceDotfiles(tx pgx.Tx, riceID uuid.UUID, filePath string, fileSize int64, dfType models.DotfilesType, price float32, productID *string) (df models.RiceDotfiles, err error) {
	df, err = txRowToStruct[models.RiceDotfiles](tx, insertDotfilesSql, riceID, filePath, fileSize, dfType, price, productID)
	return
}

func InsertRiceStar(riceID string, userID string) error {
	_, err := db.Exec(context.Background(), insertStarSql, riceID, userID)
	return err
}

func UpdateRice(riceID string, title *string, description *string) error {
	query := "UPDATE rices SET"
	args := []any{riceID}

	if title != nil {
		query += " title = $2"
		args = append(args, *title)
	}

	if description != nil {
		if len(args) > 1 {
			query += ","
		}
		query += " description = $3"
		args = append(args, *description)
	}

	query += " WHERE id = $1"

	_, err := db.Exec(context.Background(), query, args...)

	return err
}

func UpdateRiceDotfiles(riceID string, filePath string, fileSize int64) (df models.RiceDotfiles, err error) {
	df, err = rowToStruct[models.RiceDotfiles](updateDotfilesSql, riceID, filePath, fileSize)
	return
}

func UpdateRiceState(riceID string, newState models.RiceState) error {
	query := "UPDATE rices SET state = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, newState, riceID)
	return err
}

func IncrementDotfilesDownloads(riceID string) (string, error) {
	var filePath string
	err := db.QueryRow(context.Background(), incrementDownloadsSql, riceID).Scan(&filePath)
	return filePath, err
}

func DeleteRiceScreenshot(riceID string, screenshotID string) (bool, error) {
	cmd, err := db.Exec(context.Background(), deleteScreenshotSql, screenshotID, riceID)
	return cmd.RowsAffected() == 1, err
}

// star deletion is the only query where i dont see the need to check if any row was affected
func DeleteRiceStar(riceID string, userID string) error {
	_, err := db.Exec(
		context.Background(),
		"DELETE FROM rice_stars WHERE rice_id = $1 AND user_id = $2",
		riceID, userID,
	)
	return err
}

func deleteRice(exec DbExecutor, riceID string) (bool, error) {
	cmd, err := exec.Exec(
		context.Background(),
		"DELETE FROM rices WHERE id = $1",
		riceID,
	)
	return cmd.RowsAffected() == 1, err
}

func DeleteRice(riceID string) (bool, error) {
	return deleteRice(db, riceID)
}

func DeleteRiceTx(tx pgx.Tx, riceID string) (bool, error) {
	return deleteRice(tx, riceID)
}
