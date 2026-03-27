package repository

import (
	"context"
	"ricehub/src/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func InsertRiceDotfiles(tx pgx.Tx, riceID uuid.UUID, filePath string, fileSize int64, dfType *models.DotfilesType, price *float64, productID *string) (models.RiceDotfiles, error) {
	if dfType == nil {
		temp := models.Free
		dfType = &temp
	}
	if price == nil {
		temp := 1.0
		price = &temp
	}

	const query = `
	INSERT INTO rice_dotfiles (rice_id, file_path, file_size, type, price, product_id)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING *
	`

	return txRowToStruct[models.RiceDotfiles](tx, query, riceID, filePath, fileSize, dfType, price, productID)
}

func FindDotfilesProductID(tx pgx.Tx, riceID string) (productID *uuid.UUID, err error) {
	const query = "SELECT product_id FROM rice_dotfiles WHERE rice_id = $1"
	err = tx.QueryRow(context.Background(), query, riceID).Scan(&productID)
	return
}

func FetchRiceDotfilesPath(riceID string) (filePath *string, err error) {
	const query = "SELECT file_path FROM rice_dotfiles WHERE rice_id = $1"
	err = db.QueryRow(context.Background(), query, riceID).Scan(&filePath)
	return
}

func UpdateRiceDotfiles(riceID string, filePath string, fileSize int64) (models.RiceDotfiles, error) {
	const query = `
	UPDATE rice_dotfiles
	SET file_path = $2, file_size = $3
	WHERE rice_id = $1
	RETURNING *
	`

	return rowToStruct[models.RiceDotfiles](query, riceID, filePath, fileSize)
}

func UpdateDotfilesType(tx pgx.Tx, riceID string, newType models.DotfilesType, productID *string) (bool, error) {
	const query = `
	UPDATE rice_dotfiles
	SET type = $2, product_id = $3
	WHERE rice_id = $1
	`

	cmd, err := tx.Exec(context.Background(), query, riceID, newType, productID)
	return cmd.RowsAffected() > 0, err
}

func UpdateDotfilesPrice(tx pgx.Tx, riceID string, newPrice float64) (productID *uuid.UUID, err error) {
	const query = `
	UPDATE rice_dotfiles
	SET price = $2
	WHERE rice_id = $1
	RETURNING product_id
	`

	err = tx.QueryRow(context.Background(), query, riceID, newPrice).Scan(&productID)
	return
}

func IncrementDownloadCount(riceID string) (filePath string, err error) {
	const query = `
	UPDATE rice_dotfiles df
	SET download_count = download_count + 1
	FROM rices r
	WHERE r.id = $1 AND r.id = df.rice_id
	RETURNING df.file_path
	`

	err = db.QueryRow(context.Background(), query, riceID).Scan(&filePath)
	return
}
