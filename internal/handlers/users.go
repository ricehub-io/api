package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/validation"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type usersPath struct {
	UserID string `uri:"id" binding:"required,uuid"`
}

func findUser(userID uuid.UUID) (models.User, errs.AppError) {
	user, err := repository.FindUserByID(userID)
	if err != nil {
		return user, errs.FromDBError(err, errs.UserNotFound)
	}
	return user, nil
}

// Checks if user can modify the resource.
//
// It protects user data from being modified by other non-admin users.
func preCheck(token *security.AccessToken, userID uuid.UUID) (models.User, errs.AppError) {
	subID, _ := uuid.Parse(token.Subject)
	if subID != userID && !token.IsAdmin {
		return models.User{}, errs.UserError(
			"You can't access this resource",
			http.StatusForbidden,
		)
	}

	return findUser(userID)
}

// I chose to not implement custom validator for expiration/duration
// because in the end I need to parse the string again in the handler
// to get time.Duration therefore Imma just write a helper func instead
func computeExpiration(duration *string) (*time.Time, error) {
	if duration != nil {
		parsed, err := time.ParseDuration(*duration)
		if err != nil {
			return nil, errs.UserError("Failed to parse duration", http.StatusBadRequest)
		}

		if parsed.Seconds() < 0 {
			return nil, errs.UserError("Duration must be a non-negative value", http.StatusBadRequest)
		}

		temp := time.Now().Add(parsed)
		return &temp, nil
	}

	return nil, nil
}

// Tries to safely extract access token from request. Nothing happens if its unable to do so.
func GetTokenFromRequest(c *gin.Context) *security.AccessToken {
	tokenStr := c.Request.Header.Get("Authorization")
	tokenStr = strings.TrimSpace(tokenStr)

	token, err := security.ValidateToken(tokenStr)
	if err == nil {
		return token
	}

	return nil
}

func GetUserIDFromRequest(c *gin.Context) (userID *uuid.UUID) {
	if token := GetTokenFromRequest(c); token != nil {
		parsedID, _ := uuid.Parse(token.Subject)
		userID = &parsedID
	}
	return userID
}

func fetchRecentUsers(c *gin.Context, limit int) {
	users, err := repository.FetchRecentUsers(limit)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.UsersToDTO(users))
}

func fetchBannedUsers(c *gin.Context) {
	users, err := repository.FetchBannedUsers()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.UsersWithBanToDTO(users))
}

func findUserByUsername(c *gin.Context, username string) {
	user, err := repository.FindUserByUsername(username)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserNotFound))
		return
	}

	c.JSON(http.StatusOK, user.ToDTO())
}

func checkIsAdmin(header http.Header) error {
	tokenStr := header.Get("Authorization")
	tokenStr = strings.TrimSpace(tokenStr)

	token, err := security.ValidateToken(tokenStr)
	if err != nil {
		return err
	}

	if !token.IsAdmin {
		return errs.QueryRequired
	}

	return nil
}

func FetchUsers(c *gin.Context) {
	var query struct {
		Status   string `form:"status"`
		Username string `form:"username"`
		Limit    int    `form:"limit,default=-1"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(
			errs.UserError(
				fmt.Sprintf("Failed to parse query parameters: %v", err.Error()),
				http.StatusBadRequest,
			),
		)
		return
	}

	byStatus := query.Status != ""
	byUsername := query.Username != ""

	// fetch by username
	if byUsername {
		findUserByUsername(c, query.Username)
		return
	}

	// make sure the caller is an admin
	if err := checkIsAdmin(c.Request.Header); err != nil {
		c.Error(err)
		return
	}

	// fetch by status
	if byStatus {
		if query.Status != "banned" {
			c.Error(errs.UserError("Only filtering by status = `banned` is supported", http.StatusBadRequest))
			return
		}

		fetchBannedUsers(c)
		return
	}

	// fetch list of recent users
	// use default value of 20 for incorrect limit
	if query.Limit <= 0 {
		query.Limit = 20
	}

	fetchRecentUsers(c, query.Limit)
}

func GetUserByID(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	userID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)

	user, err := preCheck(token, userID)
	if err != nil {
		c.Error(err)
		return
	}

	dto := user.ToDTO()
	c.JSON(http.StatusOK, dto)
}

func GetUserRiceBySlug(c *gin.Context) {
	// path param has to be called 'id' because gin is upset otherwise
	username := c.Param("id")
	slug := c.Param("slug")

	// check if rice's author exists
	taken, err := repository.UsernameExists(username)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !taken {
		c.Error(errs.UserNotFound)
		return
	}

	// try to extract user id from caller
	token := GetTokenFromRequest(c)
	var userID *string = nil
	if token != nil {
		userID = &token.Subject
	}

	rice, err := repository.FindRiceBySlug(userID, slug, username)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.RiceNotFound))
		return
	}

	// check if rice is pending approval and if so, is the user permitted to see it
	if rice.Rice.State == models.Waiting && (token == nil || !token.IsAdmin) {
		c.Error(errs.RiceNotFound)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func FetchUserRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	userID, _ := uuid.Parse(path.UserID)
	callerUserID := GetUserIDFromRequest(c)

	rices, err := repository.FetchUserRices(userID, callerUserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTO(rices))
}

func FetchPurchasedRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	user, err := preCheck(token, userID)
	if err != nil {
		c.Error(err)
		return
	}

	rices, err := repository.FetchUserPurchasedRices(user.ID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTO(rices))
}

func UpdateDisplayName(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	token := c.MustGet("token").(*security.AccessToken)

	// check if caller is banned
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	if _, err := preCheck(token, userID); err != nil {
		c.Error(err)
		return
	}

	var body models.UpdateDisplayNameDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	// check if display name is blacklisted
	if validation.IsDisplayNameBlacklisted(body.DisplayName) {
		c.Error(errs.BlacklistedDisplayName)
		return
	}

	if err := repository.UpdateUserDisplayName(userID, body.DisplayName); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdatePassword(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	user, err := preCheck(token, userID)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.UpdatePasswordDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if !token.IsAdmin {
		match, err := argon2id.ComparePasswordAndHash(body.OldPassword, user.Password)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		if !match {
			c.Error(errs.UserError("Invalid current password provided", http.StatusForbidden))
			return
		}
	}

	hash, err := argon2id.CreateHash(body.NewPassword, argon2id.DefaultParams)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	if err := repository.UpdateUserPassword(path.UserID, hash); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UploadAvatar(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	if _, err := preCheck(token, userID); err != nil {
		c.Error(err)
		return
	}

	// check if user uploaded any file
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	ext, err := validation.ValidateFileAsImage(file)
	if err != nil {
		c.Error(err)
		return
	}

	// delete old avatar file (if exists)
	oldAvatar, err := repository.FetchUserAvatarPath(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if oldAvatar != nil {
		path := "./public" + *oldAvatar
		if err := os.Remove(path); err != nil {
			zap.L().Warn("Failed to remove old user avatar from CDN", zap.String("path", path))
		}
	}

	// save file to cdn
	avatarPath := fmt.Sprintf("/avatars/%v%v", uuid.New(), ext)
	err = c.SaveUploadedFile(file, "./public"+avatarPath)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// update avatar path in database
	err = repository.UpdateUserAvatarPath(path.UserID, &avatarPath)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"avatarUrl": config.Config.App.CDNUrl + avatarPath})
}

func BanUser(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	// 1. validate DTO
	var ban *models.BanUserDTO
	if err := validation.ValidateJSON(c, &ban); err != nil {
		c.Error(err)
		return
	}

	// 2. compute expiration
	expiresAt, err := computeExpiration(ban.Duration)
	if err != nil {
		c.Error(err)
		return
	}

	// 3. check if user exists AND is not already banned
	state, err := repository.IsUserBanned(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !state.UserExists {
		c.Error(errs.UserNotFound)
		return
	}
	if state.UserBanned {
		c.Error(errs.UserError("User is already banned", http.StatusConflict))
		return
	}

	// 4. insert ban into the database
	userBan, err := repository.InsertBan(path.UserID, token.Subject, ban.Reason, expiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation {
			c.Error(errs.UserError("You cannot ban yourself, dummy.", http.StatusBadRequest))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	// 5. remove user permissions (if has any)
	if err := repository.RevokeAdmin(path.UserID); err != nil {
		c.Error(errs.InternalError(err))
		zap.L().Error(
			"Failed to remove admin role after user ban",
			zap.String("userID", path.UserID),
			zap.Error(err),
		)
		return
	}

	// 6. return 201 with ban id in json
	c.JSON(http.StatusCreated, userBan.ToDTO())
}

func UnbanUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	// 1. check if user is banned
	state, err := repository.IsUserBanned(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !state.UserExists {
		c.Error(errs.UserNotFound)
		return
	}
	if !state.UserBanned {
		c.Error(errs.UserError("User is not banned", http.StatusConflict))
		return
	}

	// 2. revoke ban in the database
	if err := repository.RevokeBan(path.UserID); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// 3. return 204
	c.Status(http.StatusNoContent)
}

func DeleteAvatar(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	if _, err := preCheck(token, userID); err != nil {
		c.Error(err)
		return
	}

	if err := repository.UpdateUserAvatarPath(path.UserID, nil); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"avatarUrl": config.Config.App.CDNUrl + config.Config.App.DefaultAvatar})
}

func DeleteUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}

	var err error

	token := c.MustGet("token").(*security.AccessToken)
	userID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	user, err := preCheck(token, userID)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.DeleteUserDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.Password)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !match {
		c.Error(errs.UserError("Invalid current password provided", http.StatusForbidden))
		return
	}

	err = repository.DeleteUser(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
