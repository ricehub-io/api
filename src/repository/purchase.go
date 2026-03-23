package repository

import "context"

func InsertDotfilesPurchase(userID string, riceID string, paidAmount float32) error {
	const query = `
	INSERT INTO dotfiles_purchases (user_id, rice_id, price_paid)
	VALUES ($1, $2, $3)
	`
	_, err := db.Exec(context.Background(), query, userID, riceID, paidAmount)
	return err
}
