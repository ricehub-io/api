package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
)

// GetWebVarByKey fetches a website variable by its key.
// Returns WebsiteVariableNotFound if no variable with that key exists.
func GetWebVarByKey(key string) (models.WebsiteVariable, errs.AppError) {
	v, err := repository.FindWebsiteVariable(key)
	if err != nil {
		return v, errs.FromDBError(err, errs.WebsiteVariableNotFound)
	}

	return v, nil
}
