package repository

import (
	"context"
	"ricehub/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func InsertDotfilesPurchase(userID uuid.UUID, riceID uuid.UUID, paidAmount float32) error {
	const query = `
	INSERT INTO dotfiles_purchases (user_id, rice_id, price_paid)
	VALUES ($1, $2, $3)
	`
	_, err := db.Exec(context.Background(), query, userID, riceID, paidAmount)
	return err
}

func InsertDotfilesPurchaseTx(tx pgx.Tx, userID uuid.UUID, riceID uuid.UUID, paidAmount float32, purchasedAt time.Time) error {
	const query = `
	INSERT INTO dotfiles_purchases (user_id, rice_id, price_paid, purchased_at)
	VALUES ($1, $2, $3, $4)
	`
	_, err := tx.Exec(context.Background(), query, userID, riceID, paidAmount, purchasedAt)
	return err
}

func DotfilesPurchases(purchasedAfter time.Time) ([]models.DotfilesPurchase, error) {
	const query = `
	SELECT dp.user_id, df.product_id
	FROM dotfiles_purchases dp
	JOIN rice_dotfiles df ON df.rice_id = dp.rice_id
	WHERE dp.purchased_at >= $1
	`
	return rowsToStruct[models.DotfilesPurchase](query, purchasedAfter)
}
