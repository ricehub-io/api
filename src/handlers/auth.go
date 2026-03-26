package handlers

import (
	"errors"
	"math"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/security"
	"ricehub/src/utils"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func Register(c *gin.Context) {
	var body models.RegisterDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if utils.IsUsernameBlacklisted(body.Username) {
		c.Error(errs.UserError(
			"You can't use this username! Please try again with a different one.",
			http.StatusUnprocessableEntity,
		))
		return
	}

	if utils.IsDisplayNameBlacklisted(body.DisplayName) {
		c.Error(errs.BlacklistedDisplayName)
		return
	}

	taken, err := repository.UsernameExists(body.Username)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if taken {
		c.Error(errs.UserError("Username is already taken", http.StatusConflict))
		return
	}

	hashedPassword, err := argon2id.CreateHash(body.Password, argon2id.DefaultParams)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	err = repository.InsertUser(body.Username, body.DisplayName, hashedPassword)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func Login(c *gin.Context) {
	var body models.LoginDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	user, err := repository.FindUserByUsername(body.Username)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.InvalidCredentials))
		return
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.Password)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !match {
		c.Error(errs.InvalidCredentials)
		return
	}

	if err := security.VerifyUser(user); err != nil {
		c.Error(err)
		return
	}

	subActive, err := repository.SubscriptionActive(user.ID.String())
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	access, refresh, err := issueTokenPair(user.ID, user.IsAdmin, subActive)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, gin.H{
		"accessToken": access,
		"user":        user.ToDTO(),
	})
}

func RefreshToken(c *gin.Context) {
	tokenStr, err := c.Cookie("refresh_token")
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

	refresh, err := security.DecodeRefreshToken(tokenStr)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			c.Error(errs.UserError(
				"Refresh token is expired! Please authenticate again.",
				http.StatusForbidden,
			))
			return
		}

		c.Error(errs.UserError(err.Error(), http.StatusForbidden))
		return
	}

	user, err := repository.FindUserByID(refresh.Subject)
	if err != nil {
		c.Error(errs.FromDBError(err, errs.UserError(
			"Invalid refresh token! Log out and try again.",
			http.StatusForbidden,
		)))
		return
	}

	if err := security.VerifyUser(user); err != nil {
		c.Error(err)
		return
	}

	subActive, err := repository.SubscriptionActive(user.ID.String())
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	access, err := security.NewAccessToken(user.ID, user.IsAdmin, subActive)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessToken": access})
}

func LogOut(c *gin.Context) {
	clearRefreshCookie(c)
}

// issueTokenPair generates access and refresh token for given parameters.
func issueTokenPair(userID uuid.UUID, isAdmin, hasSubscription bool) (access, refresh string, err error) {
	refresh, err = security.NewRefreshToken(userID)
	if err != nil {
		return
	}

	access, err = security.NewAccessToken(userID, isAdmin, hasSubscription)
	return
}

// setRefreshCookie writes refresh token to secure and http-only cookie header.
func setRefreshCookie(c *gin.Context, token string) {
	maxAge := int(math.Round(utils.Config.JWT.AccessExpiration.Seconds()))
	c.SetCookie("refresh_token", token, maxAge, "/", utils.Config.Server.CookiesDomain, true, true)
}

// clearRefreshCookie writes empty refresh token that's expired into response headers.
func clearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", utils.Config.Server.CookiesDomain, true, true)
}
