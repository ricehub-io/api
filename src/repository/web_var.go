package repository

import "ricehub/src/models"

func FindWebsiteVariable(key string) (models.WebsiteVariable, error) {
	const query = "SELECT * FROM website_variables WHERE key = $1"
	return rowToStruct[models.WebsiteVariable](query, key)
}
