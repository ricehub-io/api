package repository

import "ricehub/src/models"

const fetchLinkSql = `
SELECT *
FROM links
WHERE name = $1
`

func FetchLink(name string) (link models.Link, err error) {
	link, err = rowToStruct[models.Link](fetchLinkSql, name)
	return
}
