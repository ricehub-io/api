package repository

import (
	"context"
	"ricehub/internal/models"
)

type TagRepository struct {
	db DBExecutor
}

func NewTagRepository(db DBExecutor) *TagRepository {
	return &TagRepository{db}
}

func (r *TagRepository) FetchTags(ctx context.Context) (models.Tags, error) {
	const query = "SELECT * FROM tags ORDER BY id"
	return rowsToStruct[models.Tag](ctx, r.db, query)
}

func (r *TagRepository) InsertTag(ctx context.Context, name string) (models.Tag, error) {
	const query = "INSERT INTO tags (name) VALUES ($1) RETURNING *"
	return rowToStruct[models.Tag](ctx, r.db, query, name)
}

func (r *TagRepository) UpdateTag(ctx context.Context, id int, name string) (models.Tag, error) {
	const query = "UPDATE tags SET name = $1 WHERE id = $2 RETURNING *"
	return rowToStruct[models.Tag](ctx, r.db, query, name, id)
}

func (r *TagRepository) DeleteTag(ctx context.Context, id int) (bool, error) {
	const query = "DELETE FROM tags WHERE id = $1"
	cmd, err := r.db.Exec(ctx, query, id)
	return cmd.RowsAffected() == 1, err
}
