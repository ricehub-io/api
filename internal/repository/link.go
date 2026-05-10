package repository

import (
	"context"

	"github.com/ricehub-io/api/internal/models"
)

type LinkRepository struct {
	db DBExecutor
}

func NewLinkRepository(db DBExecutor) *LinkRepository {
	return &LinkRepository{db}
}

func (r *LinkRepository) FindLink(ctx context.Context, name string) (models.Link, error) {
	const query = "SELECT * FROM links WHERE name = $1"
	return rowToStruct[models.Link](ctx, r.db, query, name)
}
