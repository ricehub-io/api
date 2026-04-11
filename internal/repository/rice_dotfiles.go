package repository

import (
	"context"
	"ricehub/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RiceDotfilesRepository struct {
	db DBExecutor
}

func NewRiceDotfilesRepository(db DBExecutor) *RiceDotfilesRepository {
	return &RiceDotfilesRepository{db}
}

func (r *RiceDotfilesRepository) WithTx(tx pgx.Tx) *RiceDotfilesRepository {
	return &RiceDotfilesRepository{tx}
}

func (r *RiceDotfilesRepository) InsertRiceDotfiles(
	ctx context.Context,
	riceID uuid.UUID,
	filePath string,
	fileSize int64,
	dfType *models.DotfilesType,
	price *float64,
	productID *string,
) error {
	const query = `
	INSERT INTO rice_dotfiles (rice_id, file_path, file_size, type, price, product_id)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	if dfType == nil {
		temp := models.Free
		dfType = &temp
	}
	if price == nil {
		temp := 1.0
		price = &temp
	}

	_, err := r.db.Exec(ctx, query, riceID, filePath, fileSize, dfType, price, productID)
	return err
}

func (r *RiceDotfilesRepository) FindDotfilesByProductID(
	ctx context.Context,
	productID uuid.UUID,
) (models.RiceDotfiles, error) {
	const query = "SELECT * FROM rice_dotfiles WHERE product_id = $1"
	return rowToStruct[models.RiceDotfiles](ctx, r.db, query, productID)
}

func (r *RiceDotfilesRepository) FindDotfilesProductID(
	ctx context.Context,
	riceID uuid.UUID,
) (productID *uuid.UUID, err error) {
	const query = "SELECT product_id FROM rice_dotfiles WHERE rice_id = $1"
	err = r.db.QueryRow(ctx, query, riceID).Scan(&productID)
	return
}

func (r *RiceDotfilesRepository) FetchRiceDotfilesPath(
	ctx context.Context,
	riceID uuid.UUID,
) (filePath *string, err error) {
	const query = "SELECT file_path FROM rice_dotfiles WHERE rice_id = $1"
	err = r.db.QueryRow(ctx, query, riceID).Scan(&filePath)
	return
}

func (r *RiceDotfilesRepository) UpdateRiceDotfiles(
	ctx context.Context,
	riceID uuid.UUID,
	filePath string,
	fileSize int64,
) (models.RiceDotfiles, error) {
	const query = `
	UPDATE rice_dotfiles
	SET file_path = $2, file_size = $3
	WHERE rice_id = $1
	RETURNING *
	`
	return rowToStruct[models.RiceDotfiles](ctx, r.db, query, riceID, filePath, fileSize)
}

func (r *RiceDotfilesRepository) UpdateDotfilesType(
	ctx context.Context,
	riceID uuid.UUID,
	newType models.DotfilesType,
	productID *string,
) (bool, error) {
	const query = `
	UPDATE rice_dotfiles
	SET type = $2, product_id = $3
	WHERE rice_id = $1
	`

	cmd, err := r.db.Exec(ctx, query, riceID, newType, productID)
	return cmd.RowsAffected() > 0, err
}

func (r *RiceDotfilesRepository) UpdateDotfilesPrice(
	ctx context.Context,
	riceID uuid.UUID,
	newPrice float64,
) (productID *uuid.UUID, err error) {
	const query = `
	UPDATE rice_dotfiles
	SET price = $2
	WHERE rice_id = $1
	RETURNING product_id
	`

	err = r.db.QueryRow(ctx, query, riceID, newPrice).Scan(&productID)
	return
}

func (r *RiceDotfilesRepository) IncrementDownloadCount(
	ctx context.Context,
	riceID uuid.UUID,
) (filePath string, err error) {
	const query = `
	UPDATE rice_dotfiles df
	SET download_count = download_count + 1
	FROM rices r
	WHERE r.id = $1 AND r.id = df.rice_id
	RETURNING df.file_path
	`

	err = r.db.QueryRow(ctx, query, riceID).Scan(&filePath)
	return
}
