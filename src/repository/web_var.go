package repository

import "ricehub/src/models"

const fetchWebVarSql = `
SELECT *
FROM website_variables
WHERE key = $1
`

func FetchWebsiteVariable(key string) (v models.WebsiteVariable, err error) {
	v, err = rowToStruct[models.WebsiteVariable](fetchWebVarSql, key)
	return
}
