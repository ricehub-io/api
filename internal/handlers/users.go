package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ricehub-io/api/internal/config"
	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/security"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc *services.UserService
}

func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc}
}

type usersPath struct {
	UserID string `uri:"id" binding:"required,uuid"`
}

// @Summary List users or look up a user by username
// @Description Public: use ?username= to find a specific user. Admin-only: ?status=banned to list banned users, or no query for recent users
// @Tags users
// @Produce json
// @Param username query string false "Filter by exact username"
// @Param status query string false "Filter by status (admin only, value: banned)"
// @Param limit query int false "Max results for admin recent-users query (default 20)"
// @Success 200 {object} object "UserDTO or array of UserDTO / UserWithBanDTO"
// @Failure 400 {object} models.ErrorDTO "Bad query"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var query struct {
		Status   string `form:"status"`
		Username string `form:"username"`
		Limit    int    `form:"limit,default=-1"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(errs.UserError(
			fmt.Sprintf("Failed to parse query parameters: %v", err.Error()),
			http.StatusBadRequest,
		))
		return
	}

	ctx := c.Request.Context()
	// public
	if query.Username != "" {
		user, err := h.svc.GetUserByUsername(ctx, query.Username)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, user.ToDTO())
		return
	}

	// all remaining operations are admin-only
	token := GetTokenFromRequest(c)
	if token == nil || !token.IsAdmin {
		c.Error(errs.QueryRequired)
		return
	}

	if query.Status != "" {
		if query.Status != "banned" {
			c.Error(errs.UserError(
				"Only filtering by status = `banned` is supported",
				http.StatusBadRequest,
			))
			return
		}

		users, err := h.svc.ListBannedUsers(ctx)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, models.UsersWithBanToDTO(users))
		return
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	users, err := h.svc.ListRecentUsers(ctx, limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, models.UsersToDTO(users))
}

// @Summary Get a user by ID
// @Tags users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} models.UserDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Failure 404 {object} models.ErrorDTO "User not found"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	user, err := h.svc.GetUserByID(c.Request.Context(), targetID, callerID, token.IsAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user.ToDTO())
}

// @Summary Get a specific rice owned by a user by slug
// @Tags users
// @Produce json
// @Param id path string true "Username"
// @Param slug path string true "Rice slug"
// @Success 200 {object} models.RiceWithRelationsDTO
// @Failure 404 {object} models.ErrorDTO "Rice not found"
// @Router /users/{id}/rices/{slug} [get]
func (h *UserHandler) GetUserRiceBySlug(c *gin.Context) {
	// gin requires the param name to match the route definition, which uses :id here
	username := c.Param("id")
	slug := c.Param("slug")

	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin
	callerID := GetUserIDFromRequest(c)

	rice, err := h.svc.GetUserRiceBySlug(c.Request.Context(), callerID, slug, username, isAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

// @Summary List all rices belonging to a user
// @Tags users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {array} models.PartialRiceDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Router /users/{id}/rices [get]
func (h *UserHandler) ListUserRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	userID, _ := uuid.Parse(path.UserID)
	callerID := GetUserIDFromRequest(c)

	rices, err := h.svc.ListUserRices(c.Request.Context(), userID, callerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices.ToDTO())
}

// @Summary List dotfiles the authenticated user has purchased
// @Tags users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {array} models.PartialRiceDTO
// @Failure 400 {object} models.ErrorDTO "Invalid UUID"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /users/{id}/purchased [get]
func (h *UserHandler) ListPurchasedRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	rices, err := h.svc.ListPurchasedRices(c.Request.Context(), targetID, callerID, token.IsAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices.ToDTO())
}

// @Summary Update a user's display name
// @Tags users
// @Accept json
// @Param id path string true "User ID (UUID)"
// @Param body body models.UpdateDisplayNameDTO true "New display name"
// @Success 204 "Updated"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /users/{id}/displayName [patch]
func (h *UserHandler) UpdateDisplayName(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	var body models.UpdateDisplayNameDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdateDisplayName(c.Request.Context(), targetID, callerID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Update a user's password
// @Tags users
// @Accept json
// @Param id path string true "User ID (UUID)"
// @Param body body models.UpdatePasswordDTO true "Old and new passwords"
// @Success 204 "Updated"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Invalid current password"
// @Security BearerAuth
// @Router /users/{id}/password [patch]
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	var body models.UpdatePasswordDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdatePassword(c.Request.Context(), targetID, callerID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Upload a new avatar for a user
// @Tags users
// @Accept mpfd
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param file formData file true "Avatar image file"
// @Success 201 {object} object "Returns avatarUrl (string)"
// @Failure 400 {object} models.ErrorDTO "Missing file"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /users/{id}/avatar [post]
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	var err error

	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	avatarURL, err := h.svc.UpdateAvatar(c.Request.Context(), targetID, callerID, token.IsAdmin, file)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"avatarUrl": avatarURL})
}

// @Summary Delete a user's avatar (resets to default)
// @Tags users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} object "Returns default avatarUrl (string)"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /users/{id}/avatar [delete]
func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	if err := h.svc.DeleteAvatar(c.Request.Context(), targetID, callerID, token.IsAdmin); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatarUrl": config.Config.App.CDNUrl + config.Config.App.DefaultAvatar,
	})
}

// @Summary Ban a user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "Target user ID (UUID)"
// @Param body body models.BanUserDTO true "Ban details"
// @Success 201 {object} models.UserBanDTO
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 409 {object} models.ErrorDTO "User already banned"
// @Security BearerAuth
// @Router /users/{id}/ban [post]
func (h *UserHandler) BanUser(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	adminID, _ := uuid.Parse(token.Subject)

	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	var ban models.BanUserDTO
	if err := validation.ValidateJSON(c, &ban); err != nil {
		c.Error(err)
		return
	}

	userBan, err := h.svc.BanUser(c.Request.Context(), targetID, adminID, ban)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userBan.ToDTO())
}

// @Summary Unban a user (admin only)
// @Tags users
// @Param id path string true "Target user ID (UUID)"
// @Success 204 "Unbanned"
// @Failure 403 {object} models.ErrorDTO "Admin access required"
// @Failure 409 {object} models.ErrorDTO "User is not banned"
// @Security BearerAuth
// @Router /users/{id}/ban [delete]
func (h *UserHandler) UnbanUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	if err := h.svc.UnbanUser(c.Request.Context(), targetID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Delete a user account
// @Tags users
// @Accept json
// @Param id path string true "User ID (UUID)"
// @Param body body models.DeleteUserDTO true "Password confirmation"
// @Success 204 "Deleted"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 403 {object} models.ErrorDTO "Forbidden"
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	var body models.DeleteUserDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), targetID, callerID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetTokenFromRequest tries to extract and validate the access token from the
// Authorization header. Returns nil if the token is missing or invalid.
func GetTokenFromRequest(c *gin.Context) *security.AccessToken {
	tokenStr := strings.TrimSpace(c.Request.Header.Get("Authorization"))
	token, err := security.ValidateToken(tokenStr)
	if err == nil {
		return token
	}
	return nil
}

// GetUserIDFromRequest extracts the caller's UUID from the access token, if present.
func GetUserIDFromRequest(c *gin.Context) *uuid.UUID {
	if token := GetTokenFromRequest(c); token != nil {
		id, _ := uuid.Parse(token.Subject)
		return &id
	}
	return nil
}
