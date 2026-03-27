package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func _insertRiceTags(exec DBExecutor, riceID uuid.UUID, tagIDs []int) error {
	args := make([]any, 0, len(tagIDs)+1)
	args = append(args, riceID)

	query := "INSERT INTO rice_tag (rice_id, tag_id) VALUES "
	for i, tagID := range tagIDs {
		if i > 0 {
			query += ","
		}

		// i+2 because $0 is invalid and $1 is riceID
		query += fmt.Sprintf("($1, $%d)", i+2)
		args = append(args, tagID)
	}

	_, err := exec.Exec(context.Background(), query, args...)
	return err
}

func InsertRiceTags(riceID uuid.UUID, tagIDs []int) error {
	return _insertRiceTags(db, riceID, tagIDs)
}

func InsertRiceTagsTx(tx pgx.Tx, riceID uuid.UUID, tagIDs []int) error {
	return _insertRiceTags(tx, riceID, tagIDs)
}
