package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/storage"
	"ricehub/internal/validation"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type UserService struct {
	users *repository.UserRepository
	bans  *repository.UserBanRepository
	rices *repository.RiceRepository
}

func NewUserService(
	users *repository.UserRepository,
	bans *repository.UserBanRepository,
	rices *repository.RiceRepository,
) *UserService {
	return &UserService{users, bans, rices}
}

// GetUserByUsername returns a user by their username.
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (models.User, errs.AppError) {
	user, err := s.users.FindUserByUsername(ctx, username)
	if err != nil {
		return user, errs.FromDBError(err, errs.UserNotFound)
	}
	return user, nil
}

// ListBannedUsers returns all currently banned users with their ban info.
func (s *UserService) ListBannedUsers(ctx context.Context) ([]models.UserWithBan, errs.AppError) {
	users, err := s.users.FetchBannedUsers(ctx)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return users, nil
}

// ListRecentUsers returns the most recently registered users up to the given limit.
func (s *UserService) ListRecentUsers(ctx context.Context, limit int) ([]models.User, errs.AppError) {
	users, err := s.users.FetchRecentUsers(ctx, limit)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return users, nil
}

// GetUserByID fetches a user by ID, enforcing that only the owner or an admin can access it.
func (s *UserService) GetUserByID(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
) (models.User, errs.AppError) {
	var zero models.User
	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return zero, err
	}

	user, err := s.users.FindUserByID(ctx, targetID)
	if err != nil {
		return zero, errs.FromDBError(err, errs.UserNotFound)
	}

	return user, nil
}

// GetUserRiceBySlug fetches a rice by the author's username and rice slug.
// Waiting rices are only visible to admins.
func (s *UserService) GetUserRiceBySlug(
	ctx context.Context,
	callerID *uuid.UUID,
	slug, username string,
	isAdmin bool,
) (models.RiceWithRelations, errs.AppError) {
	var zero models.RiceWithRelations

	taken, err := s.users.UsernameExists(ctx, username)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	if !taken {
		return zero, errs.UserNotFound
	}

	rice, err := s.rices.FindRiceBySlug(ctx, callerID, slug, username)
	if err != nil {
		return zero, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Waiting && !isAdmin {
		return zero, errs.RiceNotFound
	}

	return rice, nil
}

// ListUserRices returns all accepted rices for the given user.
func (s *UserService) ListUserRices(
	ctx context.Context,
	userID uuid.UUID,
	callerID *uuid.UUID,
) (models.PartialRices, errs.AppError) {
	rices, err := s.rices.FetchUserRices(ctx, userID, callerID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}

// ListPurchasedRices returns all rices the target user has purchased.
// Enforces that only the owner or an admin can fetch purchases.
func (s *UserService) ListPurchasedRices(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
) (models.PartialRices, errs.AppError) {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return nil, err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return nil, err
	}

	rices, err := s.rices.FetchUserPurchasedRices(ctx, targetID)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return rices, nil
}

// UpdateDisplayName changes a user's display name after blacklist validation.
// Enforces that only the owner or an admin can update.
func (s *UserService) UpdateDisplayName(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
	dto models.UpdateDisplayNameDTO,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if validation.IsDisplayNameBlacklisted(dto.DisplayName) {
		return errs.BlacklistedDisplayName
	}

	if err := s.users.UpdateUserDisplayName(ctx, targetID, dto.DisplayName); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdatePassword changes a user's password after verifying the current one.
// Admins bypass the current password check.
// Enforces that only the owner or an admin can update.
func (s *UserService) UpdatePassword(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
	dto models.UpdatePasswordDTO,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if !isAdmin {
		user, err := s.users.FindUserByID(ctx, targetID)
		if err != nil {
			return errs.FromDBError(err, errs.UserNotFound)
		}

		match, err := argon2id.ComparePasswordAndHash(dto.OldPassword, user.Password)
		if err != nil {
			return errs.InternalError(err)
		}
		if !match {
			return errs.InvalidCurrentPassword
		}
	}

	hash, err := argon2id.CreateHash(dto.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return errs.InternalError(err)
	}

	if err := s.users.UpdateUserPassword(ctx, targetID, hash); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdateAvatar saves a new avatar, removes the old one from disk, and updates the DB.
// Returns the CDN URL of the uploaded avatar.
// Enforces that only the owner or an admin can upload.
func (s *UserService) UpdateAvatar(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
	file *multipart.FileHeader,
) (string, errs.AppError) {
	var err error

	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return "", err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return "", err
	}

	ext, err := validation.ValidateFileAsImage(file)
	if err != nil {
		return "", err.(errs.AppError)
	}

	oldAvatar, err := s.users.FetchUserAvatarPath(ctx, targetID)
	if err != nil {
		return "", errs.InternalError(err)
	}
	if oldAvatar != nil {
		full := "./public" + *oldAvatar
		if err := os.Remove(full); err != nil {
			zap.L().Warn(
				"Failed to remove old user avatar from storage",
				zap.String("path", full),
			)
		}
	}

	filename := fmt.Sprintf("%v%v", uuid.New(), ext)
	avatarPath := "/avatars/" + filename
	if err := storage.SaveScreenshotFile(file, filename); err != nil {
		return "", errs.InternalError(err)
	}

	if err := s.users.UpdateUserAvatarPath(ctx, targetID, &avatarPath); err != nil {
		return "", errs.InternalError(err)
	}

	return config.Config.App.CDNUrl + avatarPath, nil
}

// DeleteAvatar removes the user's custom avatar from the DB.
// Enforces that only the owner or an admin can delete.
func (s *UserService) DeleteAvatar(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if err := s.users.UpdateUserAvatarPath(ctx, targetID, nil); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// DeleteUser deletes an account after verifying the user's password.
// Enforces that only the owner or an admin can delete.
func (s *UserService) DeleteUser(
	ctx context.Context,
	targetID, callerID uuid.UUID,
	isAdmin bool,
	dto models.DeleteUserDTO,
) errs.AppError {
	if _, err := security.VerifyUserID(ctx, s.users, s.bans, callerID.String()); err != nil {
		return err
	}

	if err := s.canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	user, err := s.users.FindUserByID(ctx, targetID)
	if err != nil {
		return errs.FromDBError(err, errs.UserNotFound)
	}

	match, err := argon2id.ComparePasswordAndHash(dto.Password, user.Password)
	if err != nil {
		return errs.InternalError(err)
	}
	if !match {
		return errs.InvalidCurrentPassword
	}

	if err := s.users.DeleteUser(ctx, targetID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// BanUser creates a ban record for a user and revokes their admin role if applicable.
func (s *UserService) BanUser(
	ctx context.Context,
	targetID, adminID uuid.UUID,
	dto models.BanUserDTO,
) (models.UserBan, errs.AppError) {
	var zero models.UserBan
	var err error

	expiresAt, err := s.computeExpiration(dto.Duration)
	if err != nil {
		return zero, err.(errs.AppError)
	}

	state, err := s.bans.IsUserBanned(ctx, targetID)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	if !state.UserExists {
		return zero, errs.UserNotFound
	}
	if state.UserBanned {
		return zero, errs.UserAlreadyBanned
	}

	ban, err := s.bans.InsertBan(ctx, targetID, adminID, dto.Reason, expiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation {
			return zero, errs.CannotBanSelf
		}
		return zero, errs.InternalError(err)
	}

	if err := s.users.RevokeAdmin(ctx, targetID); err != nil {
		zap.L().Error(
			"Failed to remove admin role after user ban",
			zap.String("user_id", targetID.String()),
			zap.Error(err),
		)
		return zero, errs.InternalError(err)
	}

	return ban, nil
}

// UnbanUser revokes an active ban from a user.
func (s *UserService) UnbanUser(ctx context.Context, targetID uuid.UUID) errs.AppError {
	state, err := s.bans.IsUserBanned(ctx, targetID)
	if err != nil {
		return errs.InternalError(err)
	}
	if !state.UserExists {
		return errs.UserNotFound
	}
	if !state.UserBanned {
		return errs.UserNotBanned
	}

	// TODO: log who revoked the ban
	if err := s.bans.RevokeBan(ctx, targetID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// canAccessUser checks whether the caller is allowed to access the target user's data.
// Admins bypass the ownership check.
func (s *UserService) canAccessUser(targetID, callerID uuid.UUID, isAdmin bool) errs.AppError {
	if !isAdmin && callerID != targetID {
		return errs.NoAccess
	}
	return nil
}

// computeExpiration parses an optional duration string into an absolute expiry time.
func (s *UserService) computeExpiration(duration *string) (*time.Time, errs.AppError) {
	if duration == nil {
		return nil, nil
	}

	parsed, err := time.ParseDuration(*duration)
	if err != nil {
		return nil, errs.InvalidBanDuration
	}

	if parsed.Seconds() < 0 {
		return nil, errs.NegativeBanDuration
	}

	t := time.Now().Add(parsed)
	return &t, nil
}
