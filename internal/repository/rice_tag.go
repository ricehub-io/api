package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RiceTagRepository struct {
	db DBExecutor
}

func NewRiceTagRepository(db DBExecutor) *RiceTagRepository {
	return &RiceTagRepository{db}
}
func (r *RiceTagRepository) WithTx(tx pgx.Tx) *RiceTagRepository {
	return &RiceTagRepository{tx}
}

func (r *RiceTagRepository) InsertRiceTags(ctx context.Context, riceID uuid.UUID, tagIDs []int) error {
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

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *RiceTagRepository) DeleteRiceTags(ctx context.Context, riceID uuid.UUID, tagIDs []int) error {
	const query = "DELETE FROM rice_tag WHERE rice_id = $1 AND tag_id = ANY($2)"
	_, err := r.db.Exec(ctx, query, riceID, tagIDs)
	return err
}
