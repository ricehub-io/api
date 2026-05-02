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
