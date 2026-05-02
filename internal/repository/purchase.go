package repository

import (
	"context"
	"time"

	"github.com/ricehub-io/api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DotfilesPurchaseRepository struct {
	db DBExecutor
}

func NewDotfilesPurchaseRepository(db DBExecutor) *DotfilesPurchaseRepository {
	return &DotfilesPurchaseRepository{db}
}

func (r *DotfilesPurchaseRepository) WithTx(tx pgx.Tx) *DotfilesPurchaseRepository {
	return &DotfilesPurchaseRepository{tx}
}

func (r *DotfilesPurchaseRepository) InsertDotfilesPurchase(
	ctx context.Context,
	userID, riceID uuid.UUID,
	paidAmount float32,
) error {
	const query = `
	INSERT INTO dotfiles_purchases (user_id, rice_id, price_paid)
	VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, userID, riceID, paidAmount)
	return err
}

func (r *DotfilesPurchaseRepository) InsertDotfilesPurchaseTx(
	ctx context.Context,
	userID, riceID uuid.UUID,
	paidAmount float32,
	purchasedAt time.Time,
) error {
	const query = `
	INSERT INTO dotfiles_purchases (user_id, rice_id, price_paid, purchased_at)
	VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, userID, riceID, paidAmount, purchasedAt)
	return err
}

func (r *DotfilesPurchaseRepository) DotfilesPurchases(
	ctx context.Context,
	purchasedAfter time.Time,
) ([]models.DotfilesPurchase, error) {
	const query = `
	SELECT dp.user_id, df.product_id
	FROM dotfiles_purchases dp
	JOIN rice_dotfiles df ON df.rice_id = dp.rice_id
	WHERE dp.purchased_at >= $1
	`
	return rowsToStruct[models.DotfilesPurchase](ctx, r.db, query, purchasedAfter)
}
