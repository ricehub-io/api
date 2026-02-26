package security

import (
	"errors"
	"fmt"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"time"

	"github.com/jackc/pgx/v5"
)

// checks whether user can access the API - i.e. is not banned
func VerifyUser(user *models.User) error {
	if !user.IsBanned {
		return nil
	}

	// fetch ban information
	ban, err := repository.FetchUserBan(user.ID)
	if err != nil {
		return errs.InternalError(err)
	}

	// construct response message
	msg := fmt.Sprintf("Your account has been restricted permanently. Reason: %v.", ban.Reason)
	if ban.ExpiresAt != nil {
		dur := time.Until(*ban.ExpiresAt).Truncate(time.Second)
		msg = fmt.Sprintf("Your account has been restricted for %v. Reason: %v.", dur.String(), ban.Reason)
	}

	return errs.UserError(msg, http.StatusForbidden)
}

// checks whether user (from provided ID) can access the API - i.e. is not banned
func VerifyUserID(userID string) error {
	user, err := repository.FindUserById(userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.UserNotFound
		}

		return errs.InternalError(err)
	}

	return VerifyUser(user)
}
