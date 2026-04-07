package errs

import (
	"fmt"
	"net/http"
	"strings"
)

type AppError interface {
	error
	StatusCode() int
}

type appError struct {
	Code     int
	Messages []string
	Err      error
}

func (e *appError) Error() string {
	return strings.Join(e.Messages, ", ")
}

func (e *appError) StatusCode() int {
	return e.Code
}

// unaimeds: required for errors.Is to work
func (e *appError) Unwrap() error {
	return e.Err
}

func InternalError(err error) AppError {
	return &appError{
		Code:     http.StatusInternalServerError,
		Messages: []string{"Unexpected internal server error occurred"},
		Err:      err,
	}
}

func UserError(message string, code int) AppError {
	return &appError{
		Code:     code,
		Messages: []string{message},
	}
}

func UserErrors(messages []string, code int) AppError {
	return &appError{
		Code:     code,
		Messages: messages,
	}
}

// Common errors
var NoAccess = UserError(
	"You don't have access to this resource",
	http.StatusForbidden,
)
var MissingFile = UserError("Required file is missing", http.StatusBadRequest)
var BlacklistedUsername = UserError(
	"Username contains blacklisted words",
	http.StatusUnprocessableEntity,
)
var BlacklistedDisplayName = UserError(
	"Username contains blacklisted phrases",
	http.StatusUnprocessableEntity,
)

// Auth
var UsernameTaken = UserError("This username is not available", http.StatusConflict)
var InvalidCredentials = UserError(
	"Invalid credentials provided",
	http.StatusUnauthorized,
)
var RefreshTokenExpired = UserError(
	"Refresh token is expired, please authenticate again.",
	http.StatusUnauthorized,
)
var InvalidRefreshToken = UserError(
	"Invalid refresh token, please authenticate again.",
	http.StatusUnauthorized,
)

// Comments
var CommentNotFound = UserError("Comment not found", http.StatusNotFound)
var InvalidCommentID = UserError(
	"Invalid comment ID path parameter! It must be a valid UUID.",
	http.StatusBadRequest,
)

// Reports
var ReportNotFound = UserError(
	"Report with provided ID not found!",
	http.StatusNotFound,
)
var InvalidReportID = UserError(
	"Invalid report ID path parameter! It must be a valid UUID.",
	http.StatusBadRequest,
)
var ResourceNotFound = UserError(
	"Resource with given ID not found",
	http.StatusNotFound,
)
var AlreadyReported = UserError(
	"You already submitted similar report",
	http.StatusConflict,
)

// Rices
var NoRiceFieldsToUpdate = UserError("No field to update provided", http.StatusBadRequest)
var RiceAlreadyAccepted = UserError("This rice has been already accepted", http.StatusConflict)
var FreeDotfilesNotPurchasable = UserError("You can't purchase free dotfiles", http.StatusBadRequest)
var DotfilesAlreadyOwned = UserError("You already own these dotfiles", http.StatusConflict)
var DotfilesAccessDenied = UserError("You don't have access to these dotfiles", http.StatusForbidden)
var MinimumScreenshotRequired = UserError(
	"You cannot delete this preview! At least one preview is required for a rice.",
	http.StatusUnprocessableEntity,
)
var ScreenshotNotFound = UserError("Rice preview with provided ID not found", http.StatusNotFound)
var NotEnoughScreenshots = UserError(
	"At least one screenshot is required",
	http.StatusBadRequest,
)

func TooManyScreenshots(max int64) AppError {
	return UserError(
		fmt.Sprintf("You cannot upload more than %v screenshots", max),
		http.StatusRequestEntityTooLarge,
	)
}

var BlacklistedRiceTitle = UserError(
	"Title contains blacklisted words!",
	http.StatusUnprocessableEntity,
)
var BlacklistedRiceDescription = UserError(
	"Description contains blacklisted words!",
	http.StatusUnprocessableEntity,
)
var RiceTitleTaken = UserError("This rice title is already in use", http.StatusConflict)
var RiceNotFound = UserError("Rice not found", http.StatusNotFound)
var InvalidRiceID = UserError(
	"Invalid rice ID path parameter. It must be a valid UUID.",
	http.StatusBadRequest,
)
var InvalidSortBy = UserError("Invalid sorting method", http.StatusBadRequest)

// Tags
var TagExists = UserError("Tag with that name already existis", http.StatusConflict)
var InvalidTagID = UserError(
	"Failed to parse tag ID! It must be an integer.",
	http.StatusBadRequest,
)
var TagNotFound = UserError(
	"Tag with provided ID not found",
	http.StatusNotFound,
)

// Users
var UserAlreadyBanned = UserError("User is already banned", http.StatusConflict)
var UserNotBanned = UserError("User is not banned", http.StatusConflict)
var CannotBanSelf = UserError("You cannot ban yourself", http.StatusBadRequest)
var InvalidCurrentPassword = UserError("Invalid current password provided", http.StatusForbidden)
var InvalidBanDuration = UserError("Failed to parse duration", http.StatusBadRequest)
var NegativeBanDuration = UserError("Duration must be a non-negative value", http.StatusBadRequest)
var UserNotFound = UserError("User not found", http.StatusNotFound)
var InvalidUserID = UserError(
	"Invalid user ID provided. It must be a valid UUID.",
	http.StatusBadRequest,
)
var QueryRequired = UserError(
	"At least one query parameter is required",
	http.StatusBadRequest,
)

// Links
var LinkNotFound = UserError("Link not found", http.StatusNotFound)

// Website variables
var WebsiteVariableNotFound = UserError(
	"Website variable with provided key not found",
	http.StatusNotFound,
)
var ActiveSubscription = UserError(
	"You already have an active subscription",
	http.StatusConflict,
)
