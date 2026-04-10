package services

import (
	"errors"
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/validation"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type LoginResult struct {
	User                      models.User
	AccessToken, RefreshToken string
}

// Register creates a new user account with a hashed password.
// Returns an error if the username is blacklisted, already taken, or insert fails.
func (s *AuthService) Register(dto models.RegisterDTO) errs.AppError {
	if validation.IsUsernameBlacklisted(dto.Username) {
		return errs.BlacklistedUsername
	}
	if validation.IsDisplayNameBlacklisted(dto.DisplayName) {
		return errs.BlacklistedDisplayName
	}

	taken, err := repository.UsernameExists(dto.Username)
	if err != nil {
		return errs.InternalError(err)
	}
	if taken {
		return errs.UsernameTaken
	}

	hashed, err := argon2id.CreateHash(dto.Password, argon2id.DefaultParams)
	if err != nil {
		return errs.InternalError(err)
	}

	err = repository.InsertUser(dto.Username, dto.DisplayName, hashed)
	if err != nil {
		return errs.InternalError(err)
	}

	return nil
}

// Login validates credentials, checks if user is banned, and issues an access and refresh token pair.
// Returns InvalidCredentials if username or password is wrong.
func (s *AuthService) Login(dto models.LoginDTO) (LoginResult, errs.AppError) {
	var res LoginResult

	user, err := repository.FindUserByUsername(dto.Username)
	if err != nil {
		return res, errs.FromDBError(err, errs.InvalidCredentials)
	}

	match, err := argon2id.ComparePasswordAndHash(dto.Password, user.Password)
	if err != nil {
		return res, errs.InternalError(err)
	}
	if !match {
		return res, errs.InvalidCredentials
	}

	if err := security.VerifyUser(user); err != nil {
		return res, err
	}

	subActive, err := repository.SubscriptionActive(user.ID)
	if err != nil {
		return res, errs.InternalError(err)
	}

	access, refresh, err := s.issueTokenPair(user.ID, user.IsAdmin, subActive)
	if err != nil {
		return res, errs.InternalError(err)
	}

	res.User = user
	res.AccessToken = access
	res.RefreshToken = refresh

	return res, nil
}

func (s *AuthService) RefreshToken(refreshStr string) (string, errs.AppError) {
	refresh, err := security.DecodeRefreshToken(refreshStr)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", errs.RefreshTokenExpired
		}

		return "", errs.UserError(err.Error(), http.StatusUnauthorized)
	}
	userID, _ := uuid.Parse(refresh.Subject)

	user, err := repository.FindUserByID(userID)
	if err != nil {
		return "", errs.InvalidRefreshToken
	}

	if err := security.VerifyUser(user); err != nil {
		return "", err
	}

	subActive, err := repository.SubscriptionActive(user.ID)
	if err != nil {
		return "", errs.InternalError(err)
	}

	access, err := security.NewAccessToken(user.ID, user.IsAdmin, subActive)
	if err != nil {
		return "", errs.InternalError(err)
	}

	return access, nil
}

// issueTokenPair generates access and refresh token for given parameters.
func (s *AuthService) issueTokenPair(userID uuid.UUID, isAdmin, hasSubscription bool) (access, refresh string, err error) {
	refresh, err = security.NewRefreshToken(userID)
	if err != nil {
		return
	}

	access, err = security.NewAccessToken(userID, isAdmin, hasSubscription)
	return
}
