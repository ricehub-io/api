package handlers

import (
	"errors"
	"math"
	"net/http"

	"github.com/ricehub-io/api/internal/config"
	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/services"
	"github.com/ricehub-io/api/internal/validation"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc}
}

// @Summary Register a new user account
// @Tags auth
// @Accept json
// @Param body body models.RegisterDTO true "Registration data"
// @Success 201 "User created"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 409 {object} models.ErrorDTO "Username taken"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var body models.RegisterDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.Register(c.Request.Context(), body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

// @Summary Authenticate with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.LoginDTO true "Login credentials"
// @Success 200 {object} object "Returns accessToken (string) and user (UserDTO)"
// @Failure 400 {object} models.ErrorDTO "Validation error"
// @Failure 401 {object} models.ErrorDTO "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var body models.LoginDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	res, err := h.svc.Login(c.Request.Context(), body)
	if err != nil {
		c.Error(err)
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	c.JSON(http.StatusOK, gin.H{
		"accessToken": res.AccessToken,
		"user":        res.User.ToDTO(),
	})
}

// @Summary Refresh the access token
// @Description Reads the refresh_token HttpOnly cookie and issues a new access token
// @Tags auth
// @Produce json
// @Success 200 {object} object "Returns accessToken (string)"
// @Failure 400 {object} models.ErrorDTO "Refresh token missing"
// @Failure 401 {object} models.ErrorDTO "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshStr, err := c.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			c.Error(errs.UserError(
				"Refresh token is required to generate a new access token",
				http.StatusBadRequest,
			))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	access, err := h.svc.RefreshToken(c.Request.Context(), refreshStr)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessToken": access})
}

// @Summary Logout and clear the refresh token cookie
// @Tags auth
// @Success 200 "Logged out"
// @Router /auth/logout [post]
func (h *AuthHandler) LogOut(c *gin.Context) {
	h.clearRefreshCookie(c)
}

// setRefreshCookie writes refresh token to secure and http-only cookie header.
func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	maxAge := int(math.Round(config.Config.JWT.RefreshExpiration.Seconds()))
	c.SetCookie("refresh_token", token, maxAge, "/", config.Config.Server.CookiesDomain, true, true)
}

// clearRefreshCookie writes empty refresh token that's expired into response headers.
func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", config.Config.Server.CookiesDomain, true, true)
}
