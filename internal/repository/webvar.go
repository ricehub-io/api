package repository

import (
	"context"
	"ricehub/internal/models"
)

type WebVarRepository struct {
	db DBExecutor
}

func NewWebVarRepository(db DBExecutor) *WebVarRepository {
	return &WebVarRepository{db}
}

func (r *WebVarRepository) FindWebsiteVariable(ctx context.Context, key string) (models.WebsiteVariable, error) {
	const query = "SELECT * FROM website_variables WHERE key = $1"
	return rowToStruct[models.WebsiteVariable](ctx, r.db, query, key)
}
