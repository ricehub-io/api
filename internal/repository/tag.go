package repository

import (
	"context"
	"ricehub/internal/models"
)

func FetchTags() ([]models.Tag, error) {
	const query = "SELECT * FROM tags ORDER BY id"
	return rowsToStruct[models.Tag](query)
}

func InsertTag(name string) (models.Tag, error) {
	const query = "INSERT INTO tags (name) VALUES ($1) RETURNING *"
	return rowToStruct[models.Tag](query, name)
}

func UpdateTag(id int, name string) (models.Tag, error) {
	const query = "UPDATE tags SET name = $1 WHERE id = $2 RETURNING *"
	return rowToStruct[models.Tag](query, name, id)
}

func DeleteTag(id int) (bool, error) {
	const query = "DELETE FROM tags WHERE id = $1"
	cmd, err := db.Exec(context.Background(), query, id)
	return cmd.RowsAffected() == 1, err
}
