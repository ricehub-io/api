package repository

import (
	"context"
	"ricehub/src/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func FetchProductID(tx pgx.Tx, riceID string) (productID *uuid.UUID, err error) {
	err = tx.QueryRow(
		context.Background(),
		"SELECT product_id FROM rice_dotfiles WHERE rice_id = $1",
		riceID,
	).Scan(&productID)
	return
}

func UpdateDotfilesType(tx pgx.Tx, riceID string, newType models.DotfilesType, productID *string) (bool, error) {
	cmd, err := tx.Exec(
		context.Background(),
		"UPDATE rice_dotfiles SET type = $2, product_id = $3 WHERE rice_id = $1",
		riceID, newType, productID,
	)
	return cmd.RowsAffected() > 0, err
}

func UpdateDotfilesPrice(tx pgx.Tx, riceID string, newPrice float32) (productID uuid.UUID, err error) {
	err = tx.QueryRow(
		context.Background(),
		"UPDATE rice_dotfiles SET price = $2 WHERE rice_id = $1 RETURNING product_id",
		riceID, newPrice,
	).Scan(&productID)
	return
}
