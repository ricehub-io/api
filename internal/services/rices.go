package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/ricehub-io/api/internal/config"
	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/polar"
	"github.com/ricehub-io/api/internal/repository"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/storage"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type RiceService struct {
	dbPool   *pgxpool.Pool
	rices    *repository.RiceRepository
	dotfiles *repository.RiceDotfilesRepository
	riceTags *repository.RiceTagRepository
	comments *repository.CommentRepository
	users    *repository.UserRepository
	bans     *repository.UserBanRepository
}

func NewRiceService(
	dbPool *pgxpool.Pool,
	rices *repository.RiceRepository,
	dotfiles *repository.RiceDotfilesRepository,
	riceTags *repository.RiceTagRepository,
	comments *repository.CommentRepository,
	users *repository.UserRepository,
	bans *repository.UserBanRepository,
) *RiceService {
	return &RiceService{dbPool, rices, dotfiles, riceTags, comments, users, bans}
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
	ctx context.Context,
	userID uuid.UUID,
	dto models.CreateRiceDTO,
	screenshots []*multipart.FileHeader,
	dotfiles *multipart.FileHeader,
	autoAccept bool,
	tags []int,
) errs.AppError {
	var err error

	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return err
	}

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
		_, err := validation.ValidateFileAsImage(scr)
		if err != nil {
			return err
		}

		scrPath := fmt.Sprintf("/screenshots/%v.webp", uuid.New())
		validScreenshots[scrPath] = scr
	}

	dfPath, err := storage.HandleDotfilesUpload(dotfiles)
	if err != nil {
		return err.(errs.AppError)
	}

	tx, err := s.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return errs.InternalError(err)
	}
	defer tx.Rollback(ctx)
	txRices := s.rices.WithTx(tx)

	rice, err := txRices.InsertRice(ctx, userID, dto.Title, slug.Make(dto.Title), dto.Description, autoAccept)
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
		if err := txRices.InsertRiceScreenshotTx(ctx, rice.ID, path); err != nil {
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

	txDotfles := s.dotfiles.WithTx(tx)
	if err = txDotfles.InsertRiceDotfiles(
		ctx, rice.ID, dfPath, dotfiles.Size,
		dto.DotfilesType, dto.DotfilesPrice, productID,
	); err != nil {
		return errs.InternalError(err)
	}

	if len(tags) > 0 {
		txTags := s.riceTags.WithTx(tx)
		if err := txTags.InsertRiceTags(ctx, rice.ID, tags); err != nil {
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
func (s *RiceService) ListRices(
	ctx context.Context,
	sort models.SortBy,
	pag repository.Pagination,
	userID *uuid.UUID,
) (ListRicesResult, errs.AppError) {
	var res ListRicesResult

	var rices models.PartialRices
	var err error

	switch sort {
	case models.Trending:
		rices, err = s.rices.FetchTrendingRices(ctx, &pag, userID)
	case models.Recent:
		rices, err = s.rices.FetchRecentRices(ctx, &pag, userID)
	case models.MostDownloads:
		rices, err = s.rices.FetchMostDownloadedRices(ctx, &pag, userID)
	case models.MostStars:
		rices, err = s.rices.FetchMostStarredRices(ctx, &pag, userID)
	default:
		return res, errs.InvalidSortBy
	}

	if err != nil {
		return res, errs.InternalError(err)
	}

	pageCount, err := s.rices.FetchPageCount(ctx)
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
func (s *RiceService) ListWaitingRices(ctx context.Context) (models.PartialRices, errs.AppError) {
	rices, err := s.rices.FetchWaitingRices(ctx)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}

// GetRiceByID fetches a rice by ID. Waiting rices are only visible to admins.
func (s *RiceService) GetRiceByID(
	ctx context.Context,
	userID *uuid.UUID,
	riceID uuid.UUID,
	isAdmin bool,
) (models.RiceWithRelations, errs.AppError) {
	rice, err := s.rices.FindRiceByID(ctx, userID, riceID)
	if err != nil {
		return rice, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Waiting && !isAdmin {
		return rice, errs.RiceNotFound
	}

	return rice, nil
}

// ListRiceComments returns all comments for a given rice.
func (s *RiceService) ListRiceComments(ctx context.Context, riceID uuid.UUID) ([]models.CommentWithUser, errs.AppError) {
	comments, err := s.comments.FetchCommentsByRiceID(ctx, riceID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return comments, nil
}

// UpdateRiceMetadata updates the title and/or description of a rice.
// Enforces ownership and blacklist checks.
func (s *RiceService) UpdateRiceMetadata(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
	dto models.UpdateRiceDTO,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return err
	}

	if dto.Title == nil && dto.Description == nil {
		return errs.NoRiceFieldsToUpdate
	}

	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}

	bl := config.Config.Blacklist.Words
	if dto.Title != nil && validation.ContainsBlacklistedWord(*dto.Title, bl) {
		return errs.BlacklistedRiceTitle
	}
	if dto.Description != nil && validation.ContainsBlacklistedWord(*dto.Description, bl) {
		return errs.BlacklistedRiceDescription
	}

	if err := s.rices.UpdateRice(ctx, riceID, dto.Title, dto.Description); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdateRiceState updates a rice's state to accepted or rejected (deleted).
// Returns true if the rice was rejected.
func (s *RiceService) UpdateRiceState(ctx context.Context, riceID uuid.UUID, dto models.UpdateRiceStateDTO) (rejected bool, _ errs.AppError) {
	rice, err := s.rices.FindRiceByID(ctx, nil, riceID)
	if err != nil {
		return false, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Accepted {
		return false, errs.RiceAlreadyAccepted
	}

	switch dto.NewState {
	case "accepted":
		if err := s.rices.UpdateRiceState(ctx, riceID, models.Accepted); err != nil {
			return false, errs.InternalError(err)
		}
	case "rejected":
		if _, err := s.rices.DeleteRice(ctx, riceID); err != nil {
			return false, errs.InternalError(err)
		}
		return true, nil
	}

	return false, nil
}

// DeleteRice deletes a rice and archives its Polar product if one exists.
// Enforces ownership check before proceeding.
func (s *RiceService) DeleteRice(
	ctx context.Context,
	riceID, userID uuid.UUID,
	isAdmin bool,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
		return err
	}

	if err := canModifyRice(ctx, s.rices, riceID, userID, isAdmin); err != nil {
		return err
	}

	productID, err := s.dotfiles.FindDotfilesProductID(ctx, riceID)
	if err != nil {
		return errs.FromDBError(err, errs.RiceNotFound)
	}

	deleted, err := s.rices.DeleteRice(ctx, riceID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !deleted {
		return errs.RiceNotFound
	}

	if productID != nil {
		if _, err := polar.ArchiveProduct(productID.String()); err != nil {
			zap.L().Error("Could not archive rice dotfiles product in Polar",
				zap.Error(err),
				zap.String("product_id", productID.String()),
				zap.String("rice_id", riceID.String()),
			)
		}
	}

	return nil
}

// canModifyRice checks whether the user is allowed to modify the given rice.
// Admins bypass ownership checks.
func canModifyRice(
	ctx context.Context,
	rices *repository.RiceRepository,
	riceID, userID uuid.UUID,
	isAdmin bool,
) errs.AppError {
	if isAdmin {
		return nil
	}

	owns, err := rices.UserOwnsRice(ctx, riceID, userID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !owns {
		return errs.NoAccess
	}

	return nil
}
