package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/storage"
	"ricehub/internal/validation"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// GetUserByUsername returns a user by their username.
func GetUserByUsername(username string) (models.User, errs.AppError) {
	user, err := repository.FindUserByUsername(username)
	if err != nil {
		return user, errs.FromDBError(err, errs.UserNotFound)
	}
	return user, nil
}

// ListBannedUsers returns all currently banned users with their ban info.
func ListBannedUsers() ([]models.UserWithBan, errs.AppError) {
	users, err := repository.FetchBannedUsers()
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return users, nil
}

// ListRecentUsers returns the most recently registered users up to the given limit.
func ListRecentUsers(limit int) ([]models.User, errs.AppError) {
	users, err := repository.FetchRecentUsers(limit)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return users, nil
}

// GetUserByID fetches a user by ID, enforcing that only the owner or an admin can access it.
func GetUserByID(targetID, callerID uuid.UUID, isAdmin bool) (models.User, errs.AppError) {
	var zero models.User
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return zero, err
	}

	user, err := repository.FindUserByID(targetID)
	if err != nil {
		return zero, errs.FromDBError(err, errs.UserNotFound)
	}

	return user, nil
}

// GetUserRiceBySlug fetches a rice by the author's username and rice slug.
// Waiting rices are only visible to admins.
func GetUserRiceBySlug(callerID *uuid.UUID, slug, username string, isAdmin bool) (models.RiceWithRelations, errs.AppError) {
	var zero models.RiceWithRelations

	taken, err := repository.UsernameExists(username)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	if !taken {
		return zero, errs.UserNotFound
	}

	rice, err := repository.FindRiceBySlug(callerID, slug, username)
	if err != nil {
		return zero, errs.FromDBError(err, errs.RiceNotFound)
	}
	if rice.Rice.State == models.Waiting && !isAdmin {
		return zero, errs.RiceNotFound
	}

	return rice, nil
}

// ListUserRices returns all accepted rices for the given user.
func ListUserRices(userID uuid.UUID, callerID *uuid.UUID) (models.PartialRices, errs.AppError) {
	rices, err := repository.FetchUserRices(userID, callerID)
	if err != nil {
		return nil, errs.InternalError(err)
	}
	return rices, nil
}

// ListPurchasedRices returns all rices the target user has purchased.
// Enforces that only the owner or an admin can fetch purchases.
func ListPurchasedRices(targetID, callerID uuid.UUID, isAdmin bool) (models.PartialRices, errs.AppError) {
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return nil, err
	}

	rices, err := repository.FetchUserPurchasedRices(targetID)
	if err != nil {
		return nil, errs.InternalError(err)
	}

	return rices, nil
}

// UpdateDisplayName changes a user's display name after blacklist validation.
// Enforces that only the owner or an admin can update.
func UpdateDisplayName(targetID, callerID uuid.UUID, isAdmin bool, dto models.UpdateDisplayNameDTO) errs.AppError {
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if validation.IsDisplayNameBlacklisted(dto.DisplayName) {
		return errs.BlacklistedDisplayName
	}

	if err := repository.UpdateUserDisplayName(targetID, dto.DisplayName); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdatePassword changes a user's password after verifying the current one.
// Admins bypass the current password check.
// Enforces that only the owner or an admin can update.
func UpdatePassword(targetID, callerID uuid.UUID, isAdmin bool, dto models.UpdatePasswordDTO) errs.AppError {
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if !isAdmin {
		user, err := repository.FindUserByID(targetID)
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

	if err := repository.UpdateUserPassword(targetID, hash); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// UpdateAvatar saves a new avatar, removes the old one from disk, and updates the DB.
// Returns the CDN URL of the uploaded avatar.
// Enforces that only the owner or an admin can upload.
func UpdateAvatar(targetID, callerID uuid.UUID, isAdmin bool, file *multipart.FileHeader) (string, errs.AppError) {
	var err error

	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return "", err
	}

	ext, err := validation.ValidateFileAsImage(file)
	if err != nil {
		return "", err.(errs.AppError)
	}

	oldAvatar, err := repository.FetchUserAvatarPath(targetID)
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

	if err := repository.UpdateUserAvatarPath(targetID, &avatarPath); err != nil {
		return "", errs.InternalError(err)
	}

	return config.Config.App.CDNUrl + avatarPath, nil
}

// DeleteAvatar removes the user's custom avatar from the DB.
// Enforces that only the owner or an admin can delete.
func DeleteAvatar(targetID, callerID uuid.UUID, isAdmin bool) errs.AppError {
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	if err := repository.UpdateUserAvatarPath(targetID, nil); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// DeleteUser deletes an account after verifying the user's password.
// Enforces that only the owner or an admin can delete.
func DeleteUser(targetID, callerID uuid.UUID, isAdmin bool, dto models.DeleteUserDTO) errs.AppError {
	if err := canAccessUser(targetID, callerID, isAdmin); err != nil {
		return err
	}

	user, err := repository.FindUserByID(targetID)
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

	if err := repository.DeleteUser(targetID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// BanUser creates a ban record for a user and revokes their admin role if applicable.
func BanUser(targetID, adminID uuid.UUID, dto models.BanUserDTO) (models.UserBan, errs.AppError) {
	var zero models.UserBan
	var err error

	expiresAt, err := computeExpiration(dto.Duration)
	if err != nil {
		return zero, err.(errs.AppError)
	}

	state, err := repository.IsUserBanned(targetID)
	if err != nil {
		return zero, errs.InternalError(err)
	}
	if !state.UserExists {
		return zero, errs.UserNotFound
	}
	if state.UserBanned {
		return zero, errs.UserAlreadyBanned
	}

	ban, err := repository.InsertBan(targetID, adminID, dto.Reason, expiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation {
			return zero, errs.CannotBanSelf
		}
		return zero, errs.InternalError(err)
	}

	if err := repository.RevokeAdmin(targetID); err != nil {
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
func UnbanUser(targetID uuid.UUID) errs.AppError {
	state, err := repository.IsUserBanned(targetID)
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
	if err := repository.RevokeBan(targetID); err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// canAccessUser checks whether the caller is allowed to access the target user's data.
// Admins bypass the ownership check.
func canAccessUser(targetID, callerID uuid.UUID, isAdmin bool) errs.AppError {
	if !isAdmin && callerID != targetID {
		return errs.NoAccess
	}
	return nil
}

// computeExpiration parses an optional duration string into an absolute expiry time.
func computeExpiration(duration *string) (*time.Time, errs.AppError) {
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
