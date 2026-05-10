package services

import (
	"context"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"
)

type WebVarService struct {
	webvars *repository.WebVarRepository
}

func NewWebVarService(webvars *repository.WebVarRepository) *WebVarService {
	return &WebVarService{webvars}
}

// GetWebVarByKey fetches a website variable by its key.
// Returns WebsiteVariableNotFound if no variable with that key exists.
func (s *WebVarService) GetWebVarByKey(ctx context.Context, key string) (models.WebsiteVariable, errs.AppError) {
	v, err := s.webvars.FindWebsiteVariable(ctx, key)
	if err != nil {
		return v, errs.FromDBError(err, errs.WebsiteVariableNotFound)
	}
	return v, nil
}
