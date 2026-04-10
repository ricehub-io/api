package handlers

import (
	"errors"
	"math"
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/services"
	"ricehub/internal/validation"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var body models.RegisterDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if err := services.Register(body); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func Login(c *gin.Context) {
	var body models.LoginDTO
	if err := validation.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	res, err := services.Login(body)
	if err != nil {
		c.Error(err)
		return
	}

	setRefreshCookie(c, res.RefreshToken)
	c.JSON(http.StatusOK, gin.H{
		"accessToken": res.AccessToken,
		"user":        res.User.ToDTO(),
	})
}

func RefreshToken(c *gin.Context) {
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

	access, err := services.RefreshToken(refreshStr)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessToken": access})
}

func LogOut(c *gin.Context) {
	clearRefreshCookie(c)
}

// setRefreshCookie writes refresh token to secure and http-only cookie header.
func setRefreshCookie(c *gin.Context, token string) {
	maxAge := int(math.Round(config.Config.JWT.AccessExpiration.Seconds()))
	c.SetCookie("refresh_token", token, maxAge, "/", config.Config.Server.CookiesDomain, true, true)
}

// clearRefreshCookie writes empty refresh token that's expired into response headers.
func clearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", config.Config.Server.CookiesDomain, true, true)
}
