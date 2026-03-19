package repository

import (
	"context"
	"ricehub/src/models"
)

func dbUpdate(query string, args ...any) (bool, error) {
	tag, err := db.Exec(context.Background(), query, args...)
	return tag.RowsAffected() > 0, err
}

func UpdateDotfilesType(riceID string, newType models.DotfilesType) (bool, error) {
	return dbUpdate(
		"UPDATE rice_dotfiles SET type = $2 WHERE rice_id = $1",
		riceID, newType,
	)
}

func UpdateDotfilesPrice(riceID string, newPrice float32) (bool, error) {
	return dbUpdate(
		"UPDATE rice_dotfiles SET price = $2 WHERE rice_id = $1",
		riceID, newPrice,
	)
}
