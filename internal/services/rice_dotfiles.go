package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/storage"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"go.uber.org/zap"
)

type RiceDotfilesService struct {
	rices    *repository.RiceRepository
	dotfiles *repository.RiceDotfilesRepository
}

func NewRiceDotfilesService(
	rices *repository.RiceRepository,
	dotfiles *repository.RiceDotfilesRepository,
) *RiceDotfilesService {
	return &RiceDotfilesService{rices, dotfiles}
}

type DownloadDotfilesResult struct {
	FilePath string
	FileName string
}

// PurchaseDotfiles creates a Polar checkout session for paid dotfiles.
// Returns the checkout URL to redirect the user to, or create embedded checkout.
func (s *RiceDotfilesService) PurchaseDotfiles(
	ctx context.Context,
	userID, riceID uuid.UUID,
) (string, errs.AppError) {
	rice, err := s.rices.FindRiceByID(ctx, &userID, riceID)
	if err != nil {
		return "", errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Dotfiles.Type == models.Free {
		return "", errs.FreeDotfilesNotPurchasable
	}
	if rice.IsOwned {
		return "", errs.DotfilesAlreadyOwned
	}

	res, err := polar.CreateCheckoutSession(userID, *rice.Dotfiles.ProductID)
	if err != nil {
		return "", errs.InternalError(err)
	}

	return res.Checkout.URL, nil
}

// DownloadDotfiles verifies access, increments the download counter, logs the
// download event, and returns the file path and attachment filename.
func (s *RiceDotfilesService) DownloadDotfiles(
	ctx context.Context,
	riceID uuid.UUID,
	userID *uuid.UUID,
) (DownloadDotfilesResult, errs.AppError) {
	var res DownloadDotfilesResult

	rice, err := s.rices.FindRiceByID(ctx, userID, riceID)
	if err != nil {
		return res, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Dotfiles.Type != models.Free && !rice.IsOwned {
		return res, errs.DotfilesAccessDenied
	}

	filePath, err := s.dotfiles.IncrementDownloadCount(ctx, riceID)
	if err != nil {
		return res, errs.FromDBError(err, errs.RiceNotFound)
	}

	if err := s.rices.InsertRiceDownload(ctx, riceID, userID); err != nil {
		zap.L().Error(
			"Failed to insert download event",
			zap.Error(err),
			zap.String("rice_id", riceID.String()),
		)
	}

	ext := filepath.Ext(filePath)
	timestamp := time.Now().UTC().Format("20060102-150405")

	res.FilePath = "./public" + filePath
	res.FileName = fmt.Sprintf("%s-%s%s", slug.Make(rice.Rice.Title), timestamp, ext)
	return res, nil
}

// UpdateDotfiles replaces the dotfiles archive for a rice, deleting the old file
// from disk first. Enforces ownership check before proceeding.
func (s *RiceDotfilesService) UpdateDotfiles(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	file *multipart.FileHeader,
) (models.RiceDotfiles, errs.AppError) {
	var zero models.RiceDotfiles
	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return zero, err
	}

	oldPath, err := s.dotfiles.FetchRiceDotfilesPath(ctx, riceID)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	if oldPath != nil {
		full := "./public" + *oldPath
		if err := os.Remove(full); err != nil {
			zap.L().Error("Failed to remove old dotfiles from storage", zap.String("path", full))
		}
	}

	filePath, appErr := storage.HandleDotfilesUpload(file)
	if appErr != nil {
		return zero, appErr
	}

	df, err := s.dotfiles.UpdateRiceDotfiles(ctx, riceID, filePath, file.Size)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	return df, nil
}

// UpdateDotfilesType switches dotfiles between free and paid - creating, hiding,
// or unhiding the corresponding Polar product as needed.
// Enforces ownership check before proceeding.
func (s *RiceDotfilesService) UpdateDotfilesType(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	dto models.UpdateDotfilesTypeDTO,
) errs.AppError {
	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}

	var productID *string

	if dto.NewType == models.Free {
		existingProdID, err := s.dotfiles.FindDotfilesProductID(ctx, riceID)
		if err != nil {
			return errs.InternalError(err)
		}
		if existingProdID != nil {
			idStr := existingProdID.String()
			productID = &idStr
			if _, err = polar.HideProduct(idStr); err != nil {
				return errs.InternalError(err)
			}
		}
	} else {
		data, err := s.rices.FindRiceWithDotfilesByID(ctx, riceID)
		if err != nil {
			return errs.InternalError(err)
		}
		if data.Dotfiles.ProductID != nil {
			idStr := data.Dotfiles.ProductID.String()
			if _, err := polar.ShowProduct(idStr); err != nil {
				return errs.InternalError(err)
			}
			productID = &idStr
		} else {
			res, err := polar.CreateProduct(data.Rice.Title, data.Dotfiles.Price)
			if err != nil {
				return errs.InternalError(err)
			}
			productID = &res.Product.ID
		}
	}

	updated, err := s.dotfiles.UpdateDotfilesType(ctx, riceID, dto.NewType, productID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !updated {
		return errs.RiceNotFound
	}

	return nil
}

// UpdateDotfilesPrice updates the price of paid dotfiles and syncs it with Polar.
// Enforces ownership check before proceeding.
func (s *RiceDotfilesService) UpdateDotfilesPrice(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	dto models.UpdateDotfilesPriceDTO,
) errs.AppError {
	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}

	productID, err := s.dotfiles.FindDotfilesProductID(ctx, riceID)
	if err != nil {
		return errs.FromDBError(err, errs.RiceNotFound)
	}

	if _, err = polar.UpdatePrice(productID.String(), dto.NewPrice); err != nil {
		return errs.InternalError(err)
	}

	if _, err := s.dotfiles.UpdateDotfilesPrice(ctx, riceID, dto.NewPrice); err != nil {
		return errs.InternalError(err)
	}

	return nil
}
