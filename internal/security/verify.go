package security

import (
	"fmt"
	"net/http"
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/repository"
	"time"

	"github.com/google/uuid"
)

// VerifyUser checks if user is banned.
func VerifyUser(user models.User) errs.AppError {
	if !user.IsBanned {
		return nil
	}

	ban, err := repository.FindUserBan(user.ID)
	if err != nil {
		return errs.InternalError(err)
	}

	msg := fmt.Sprintf(
		"Your account has been restricted permanently. Reason: %v.",
		ban.Reason,
	)
	if ban.ExpiresAt != nil {
		dur := time.Until(*ban.ExpiresAt).Truncate(time.Second)
		msg = fmt.Sprintf(
			"Your account has been restricted for %v. Reason: %v.",
			dur.String(), ban.Reason,
		)
	}

	return errs.UserError(msg, http.StatusForbidden)
}

// VerifyUserID checks if the user exists and is not banned. Given user ID must be a valid UUID.
//
// Returns parsed user ID.
func VerifyUserID(strUserID string) (uuid.UUID, errs.AppError) {
	userID, _ := uuid.Parse(strUserID)

	user, err := repository.FindUserByID(userID)
	if err != nil {
		return userID, errs.FromDBError(err, errs.UserNotFound)
	}

	return userID, VerifyUser(user)
}
