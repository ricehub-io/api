package handlers

import (
	"fmt"
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"
	"strings"

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

	// public
	if query.Username != "" {
		user, err := h.svc.GetUserByUsername(query.Username)
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

		users, err := h.svc.ListBannedUsers()
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
	users, err := h.svc.ListRecentUsers(limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, models.UsersToDTO(users))
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, _ := uuid.Parse(token.Subject)

	user, err := h.svc.GetUserByID(targetID, callerID, token.IsAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user.ToDTO())
}

func (h *UserHandler) GetUserRiceBySlug(c *gin.Context) {
	// gin requires the param name to match the route definition, which uses :id here
	username := c.Param("id")
	slug := c.Param("slug")

	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin
	callerID := GetUserIDFromRequest(c)

	rice, err := h.svc.GetUserRiceBySlug(callerID, slug, username, isAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func (h *UserHandler) ListUserRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	userID, _ := uuid.Parse(path.UserID)
	callerID := GetUserIDFromRequest(c)

	rices, err := h.svc.ListUserRices(userID, callerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices.ToDTO())
}

func (h *UserHandler) ListPurchasedRices(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	rices, err := h.svc.ListPurchasedRices(targetID, callerID, token.IsAdmin)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rices.ToDTO())
}

func (h *UserHandler) UpdateDisplayName(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.UpdateDisplayNameDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdateDisplayName(targetID, callerID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.UpdatePasswordDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.UpdatePassword(targetID, callerID, token.IsAdmin, body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	var err error

	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	avatarURL, err := h.svc.UpdateAvatar(targetID, callerID, token.IsAdmin, file)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"avatarUrl": avatarURL})
}

func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.DeleteAvatar(targetID, callerID, token.IsAdmin); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatarUrl": config.Config.App.CDNUrl + config.Config.App.DefaultAvatar,
	})
}

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

	userBan, err := h.svc.BanUser(targetID, adminID, ban)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userBan.ToDTO())
}

func (h *UserHandler) UnbanUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	if err := h.svc.UnbanUser(targetID); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	var path usersPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.InvalidUserID)
		return
	}
	targetID, _ := uuid.Parse(path.UserID)

	token := c.MustGet("token").(*security.AccessToken)
	callerID, err := security.VerifyUserID(token.Subject)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.DeleteUserDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.DeleteUser(targetID, callerID, token.IsAdmin, body); err != nil {
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
