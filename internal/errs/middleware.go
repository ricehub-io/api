package errs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func logInternalError(logger *zap.Logger, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		logger.Error("Row not found error", zap.Error(err))
		return
	}

	logger.Error("Unexpected error occurred", zap.Error(err))
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		errs := c.Errors
		if len(errs) == 0 {
			return
		}

		err := errs.Last().Err

		var appErr *AppError
		if ok := errors.As(err, &appErr); ok {
			if appErr.Err != nil {
				logInternalError(logger, appErr.Err)
			}

			c.JSON(appErr.Code, gin.H{"errors": appErr.Messages})
		} else {
			logger.Error("Unhandled non-app error occurred", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Unhandled server error! Please report to administrator.",
			})
		}
	}
}
