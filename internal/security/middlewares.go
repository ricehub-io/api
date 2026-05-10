package security

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ricehub-io/api/internal/cache"
	"github.com/ricehub-io/api/internal/config"
	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func ValidateToken(tokenStr string) (*AccessToken, errs.AppError) {
	if len(tokenStr) == 0 {
		return nil, errs.MissingAuthToken
	}

	tokenStr, found := strings.CutPrefix(tokenStr, "Bearer ")
	if !found {
		return nil, errs.UserError("Invalid authorization header format. It must begin with 'Bearer'", http.StatusUnauthorized)
	}

	token, err := DecodeAccessToken(tokenStr)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errs.UserError("Access token is expired! Please refresh it.", http.StatusUnauthorized)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, errs.UserError("Access token has an invalid signature! Please authenticate again.", http.StatusUnauthorized)
		}
		return nil, errs.UserError(err.Error(), http.StatusUnauthorized)
	}

	return token, nil
}

func AuthMiddleware(c *gin.Context) {
	tokenStr := c.Request.Header.Get("Authorization")
	tokenStr = strings.TrimSpace(tokenStr)

	token, err := ValidateToken(tokenStr)
	if err != nil {
		// reading the request so Firefox doesn't throw NS_ERROR_NET_RESET
		_, _ = c.GetRawData()

		c.Error(err)
		c.Abort()
		return
	}

	c.Set("token", token)
	c.Next()
}

func AdminMiddleware(
	userRepo *repository.UserRepository,
	banRepo *repository.UserBanRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, ok := c.Get("token")
		if !ok {
			c.Error(errs.MissingAuthToken)
			c.Abort()
			return
		}
		token := rawToken.(*AccessToken)

		if !token.IsAdmin {
			c.Error(errs.NoAccess)
			c.Abort()
			return
		}

		// check if admin is banned
		if _, err := VerifyUserID(c.Request.Context(), userRepo, banRepo, token.Subject); err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		c.Next()
	}
}

func getClientID(c *gin.Context) string {
	clientID := c.RemoteIP()

	// try to extract access token from headers
	tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
	token, err := ValidateToken(tokenStr)
	if err == nil {
		clientID = token.Subject
	}

	return clientID
}

// Checks whether request caller is an admin without throwing any errors
func isAdmin(c *gin.Context) bool {
	tokenStr := c.Request.Header.Get("Authorization")
	tokenStr = strings.TrimSpace(tokenStr)
	token, err := ValidateToken(tokenStr)
	return err == nil && token.IsAdmin
}

func RateLimitMiddleware(maxRequests int64, resetAfter time.Duration) gin.HandlerFunc {
	logger := zap.L()
	logger.Info("Creating a rate limit middleware",
		zap.Int64("max_requests", maxRequests),
		zap.Duration("reset_after", resetAfter),
	)

	return func(c *gin.Context) {
		if config.Config.App.DisableRateLimits || isAdmin(c) {
			c.Next()
			return
		}

		clientID := getClientID(c)

		count, err := cache.IncrementRateLimit(clientID, resetAfter)
		if err != nil {
			logger.Error("Failed to increment rate limit for client",
				zap.String("client_id", clientID),
				zap.Error(err),
			)
		}

		if count > maxRequests {
			c.Error(errs.UserError("You are sending too many requests to the server!", http.StatusTooManyRequests))
			c.Abort()
			return
		}

		c.Next()
	}
}

func PathRateLimitMiddleware(maxRequests int64, resetAfter time.Duration) gin.HandlerFunc {
	logger := zap.L()

	return func(c *gin.Context) {
		if config.Config.App.DisableRateLimits || isAdmin(c) {
			c.Next()
			return
		}

		clientID := getClientID(c)
		path := c.Request.URL.Path

		count, err := cache.IncrementPathRateLimit(path, clientID, resetAfter)
		if err != nil {
			logger.Error("Failed to increment path rate limit for client",
				zap.String("path", path),
				zap.String("client_id", clientID),
				zap.Error(err),
			)
		}

		if count > maxRequests {
			c.Error(errs.UserError("You are sending too many requests to this path!", http.StatusTooManyRequests))
			c.Abort()
			return
		}

		c.Next()
	}
}

func FileSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// try to parse the request form
		if err := c.Request.ParseMultipartForm(1024); err != nil {
			if _, ok := err.(*http.MaxBytesError); ok {
				c.Error(errs.UserError(fmt.Sprintf("Uploaded file(s) exceed(s) the size limit of %v bytes!", maxBytes), http.StatusRequestEntityTooLarge))
				c.Abort()
				return
			}

			c.Error(errs.UserError(fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest))
			c.Abort()
			return
		}

		c.Next()
	}
}

// middleware that automatically responds with appropriate error if maintenance is toggled in config
func MaintenanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Config.App.Maintenance {
			_, _ = c.GetRawData()

			c.Error(errs.UserError(
				"API is in read-only mode for a maintenance. Please retry later.",
				http.StatusServiceUnavailable,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
