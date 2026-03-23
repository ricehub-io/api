package repository

import "ricehub/src/models"

func FindLink(name string) (models.Link, error) {
	const query = "SELECT * FROM links WHERE name = $1"
	return rowToStruct[models.Link](query, name)
}
