package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/repository"
	"ricehub/internal/storage"
	"ricehub/internal/validation"

	"github.com/google/uuid"
)

// CreateScreenshot validates and saves new screenshot files for a rice, then
// inserts them into the database. Returns the CDN URLs of the created screenshots.
// Enforces ownership and screenshot limit checks.
func CreateScreenshot(userID, riceID uuid.UUID, files []*multipart.FileHeader, isAdmin bool) ([]string, errs.AppError) {
	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return nil, err
	}

	count, err := repository.FetchRiceScreenshotCount(riceID)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	maxScreenshots := config.Config.Limits.MaxPreviewsPerRice
	if int64(count+len(files)) > maxScreenshots {
		return nil, errs.TooManyScreenshots(maxScreenshots)
	}

	type validFile struct {
		path   string
		header *multipart.FileHeader
	}

	validFiles := make([]validFile, 0, len(files))
	for _, file := range files {
		ext, err := validation.ValidateFileAsImage(file)
		if err != nil {
			return nil, err
		}
		validFiles = append(validFiles, validFile{
			path:   fmt.Sprintf("/previews/%v%v", uuid.New(), ext),
			header: file,
		})
	}

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	defer tx.Rollback(ctx)

	screenshots := make([]string, 0, len(validFiles))
	for _, vf := range validFiles {
		filename := filepath.Base(vf.path)
		if err := storage.SaveScreenshotFile(vf.header, filename); err != nil {
			return nil, errs.InternalError(err)
		}
		if err := repository.InsertRiceScreenshotTx(tx, riceID, vf.path); err != nil {
			return nil, errs.InternalError(err)
		}
		screenshots = append(screenshots, config.Config.App.CDNUrl+vf.path)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.InternalError(err)
	}

	return screenshots, nil
}

// DeleteScreenshot removes a screenshot from a rice, enforcing a minimum of one
// screenshot per rice. Enforces ownership check before proceeding.
func DeleteScreenshot(riceID, screenshotID, userID uuid.UUID, isAdmin bool) errs.AppError {
	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return err
	}

	count, err := repository.FetchRiceScreenshotCount(riceID)
	if err != nil {
		return errs.InternalError(err)
	}
	if count <= 1 {
		return errs.MinimumScreenshotRequired
	}

	deleted, err := repository.DeleteRiceScreenshot(riceID, screenshotID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !deleted {
		return errs.ScreenshotNotFound
	}

	return nil
}
