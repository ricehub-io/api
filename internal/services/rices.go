package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/storage"
	"ricehub/internal/validation"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type RiceService struct{}

func NewRiceService() *RiceService {
	return &RiceService{}
}

type ListRicesResult struct {
	Rices models.PartialRices
	// unaimeds: why is it f32??? why did i do that??
	PageCount float32
}

// CreateRice validates and saves a new rice with its screenshots, dotfiles, and tags.
// autoAccept skips the waiting state and immediately publishes the rice.
// If the dotfiles type is paid, a Polar product is created for the purchase flow.
// TODO: remove uploaded files if tx failed
func (s *RiceService) CreateRice(
	userID uuid.UUID,
	dto models.CreateRiceDTO,
	screenshots []*multipart.FileHeader,
	dotfiles *multipart.FileHeader,
	autoAccept bool,
	tags []int,
) errs.AppError {
	var err error

	if len(screenshots) <= 0 {
		return errs.NotEnoughScreenshots
	}

	maxScreenshots := config.Config.Limits.MaxScreenshotsPerRice
	if int64(len(screenshots)) > maxScreenshots {
		return errs.TooManyScreenshots(maxScreenshots)
	}

	bl := config.Config.Blacklist.Words
	if validation.ContainsBlacklistedWord(dto.Title, bl) {
		return errs.BlacklistedRiceTitle
	}
	if validation.ContainsBlacklistedWord(dto.Description, bl) {
		return errs.BlacklistedRiceDescription
	}

	validScreenshots := make(map[string]*multipart.FileHeader, len(screenshots))
	for _, scr := range screenshots {
		ext, err := validation.ValidateFileAsImage(scr)
		if err != nil {
			return err
		}

		scrPath := fmt.Sprintf("/screenshots/%v%v", uuid.New(), ext)
		validScreenshots[scrPath] = scr
	}

	dfPath, err := storage.HandleDotfilesUpload(dotfiles)
	if err != nil {
		return err.(errs.AppError)
	}

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		return errs.InternalError(err)
	}
	defer tx.Rollback(ctx)

	rice, err := repository.InsertRice(tx, userID, dto.Title, slug.Make(dto.Title), dto.Description, autoAccept)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return errs.RiceTitleTaken
		}
		return errs.InternalError(err)
	}

	for path, file := range validScreenshots {
		filename := filepath.Base(path)
		if err := storage.SaveScreenshotFile(file, filename); err != nil {
			return errs.InternalError(err)
		}
		if err := repository.InsertRiceScreenshotTx(tx, rice.ID, path); err != nil {
			return errs.InternalError(err)
		}
	}

	var productID *string

	if dto.DotfilesType != nil && *dto.DotfilesType != models.Free {
		res, err := polar.CreateProduct(dto.Title, *dto.DotfilesPrice)
		if err != nil {
			return errs.InternalError(err)
		}

		productID = &res.Product.ID
	}

	if err = repository.InsertRiceDotfiles(
		tx, rice.ID, dfPath, dotfiles.Size,
		dto.DotfilesType, dto.DotfilesPrice, productID,
	); err != nil {
		return errs.InternalError(err)
	}

	if len(tags) > 0 {
		if err := repository.InsertRiceTagsTx(tx, rice.ID, tags); err != nil {
			return errs.InternalError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// ListRices fetches a paginated list of accepted rices for the given sort method.
// userID is optional and used to populate IsStarred/IsOwned fields per rice.
// Results are reversed in memory when pag.Reverse is set.
func (s *RiceService) ListRices(sort models.SortBy, pag repository.Pagination, userID *uuid.UUID) (ListRicesResult, errs.AppError) {
	var res ListRicesResult

	var rices models.PartialRices
	var err error

	switch sort {
	case models.Trending:
		rices, err = repository.FetchTrendingRices(&pag, userID)
	case models.Recent:
		rices, err = repository.FetchRecentRices(&pag, userID)
	case models.MostDownloads:
		rices, err = repository.FetchMostDownloadedRices(&pag, userID)
	case models.MostStars:
		rices, err = repository.FetchMostStarredRices(&pag, userID)
	default:
		return res, errs.InvalidSortBy
	}

	if err != nil {
		return res, errs.InternalError(err)
	}

	pageCount, err := repository.FetchPageCount()
	if err != nil {
		return res, errs.InternalError(err)
	}

	if pag.Reverse {
		for i, j := 0, len(rices)-1; i < j; i, j = i+1, j-1 {
			rices[i], rices[j] = rices[j], rices[i]
		}
	}

	res.Rices = rices
	res.PageCount = pageCount
	return res, nil
}

// ListWaitingRices returns all rices pending admin review.
func (s *RiceService) ListWaitingRices() (models.PartialRices, errs.AppError) {
	rices, err := repository.FetchWaitingRices()
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}

// GetRiceByID fetches a rice by ID. Waiting rices are only visible to admins.
func (s *RiceService) GetRiceByID(userID *uuid.UUID, riceID uuid.UUID, isAdmin bool) (models.RiceWithRelations, errs.AppError) {
	rice, err := repository.FindRiceByID(userID, riceID)
	if err != nil {
		return rice, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Waiting && !isAdmin {
		return rice, errs.RiceNotFound
	}

	return rice, nil
}

// ListRiceComments returns all comments for a given rice.
func (s *RiceService) ListRiceComments(riceID uuid.UUID) ([]models.CommentWithUser, errs.AppError) {
	comments, err := repository.FetchCommentsByRiceID(riceID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return comments, nil
}

// UpdateRiceMetadata updates the title and/or description of a rice.
// Enforces ownership and blacklist checks.
func (s *RiceService) UpdateRiceMetadata(riceID, userID uuid.UUID, isAdmin bool, dto models.UpdateRiceDTO) errs.AppError {
	if dto.Title == nil && dto.Description == nil {
		return errs.NoRiceFieldsToUpdate
	}

	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return err
	}

	bl := config.Config.Blacklist.Words
	if dto.Title != nil && validation.ContainsBlacklistedWord(*dto.Title, bl) {
		return errs.BlacklistedRiceTitle
	}
	if dto.Description != nil && validation.ContainsBlacklistedWord(*dto.Description, bl) {
		return errs.BlacklistedRiceDescription
	}

	if err := repository.UpdateRice(riceID, dto.Title, dto.Description); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdateRiceState updates a rice's state to accepted or rejected (deleted).
// Returns true if the rice was rejected.
func (s *RiceService) UpdateRiceState(riceID uuid.UUID, dto models.UpdateRiceStateDTO) (rejected bool, _ errs.AppError) {
	rice, err := repository.FindRiceByID(nil, riceID)
	if err != nil {
		return false, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Accepted {
		return false, errs.RiceAlreadyAccepted
	}

	switch dto.NewState {
	case "accepted":
		if err := repository.UpdateRiceState(riceID, models.Accepted); err != nil {
			return false, errs.InternalError(err)
		}
	case "rejected":
		if _, err := repository.DeleteRice(riceID); err != nil {
			return false, errs.InternalError(err)
		}
		return true, nil
	}

	return false, nil
}

// DeleteRice deletes a rice and archives its Polar product if one exists.
// Enforces ownership check before proceeding.
func (s *RiceService) DeleteRice(riceID, userID uuid.UUID, isAdmin bool) errs.AppError {
	if err := canModifyRice(riceID, userID, isAdmin); err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		return errs.InternalError(err)
	}
	defer tx.Rollback(ctx)

	productID, err := repository.FindDotfilesProductID(tx, riceID)
	if err != nil {
		return errs.InternalError(err)
	}

	deleted, err := repository.DeleteRiceTx(tx, riceID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !deleted {
		return errs.RiceNotFound
	}

	if productID != nil {
		if _, err := polar.ArchiveProduct(productID.String()); err != nil {
			return errs.InternalError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// canModifyRice checks whether the user is allowed to modify the given rice.
// Admins bypass ownership checks.
func canModifyRice(riceID, userID uuid.UUID, isAdmin bool) errs.AppError {
	if isAdmin {
		return nil
	}

	owns, err := repository.UserOwnsRice(riceID, userID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !owns {
		return errs.NoAccess
	}

	return nil
}
