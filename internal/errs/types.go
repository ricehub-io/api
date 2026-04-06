package errs

import (
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

// Rices
var RiceNotFound = UserError("Rice not found", http.StatusNotFound)
var InvalidRiceID = UserError(
	"Invalid rice ID path parameter. It must be a valid UUID.",
	http.StatusBadRequest,
)
var BlacklistedRiceTitle = UserError(
	"Title contains blacklisted words!",
	http.StatusUnprocessableEntity,
)
var BlacklistedRiceDescription = UserError(
	"Description contains blacklisted words!",
	http.StatusUnprocessableEntity,
)

// Tags
var InvalidTagID = UserError(
	"Failed to parse tag ID! It must be an integer.",
	http.StatusBadRequest,
)
var TagNotFound = UserError(
	"Tag with provided ID not found",
	http.StatusNotFound,
)

// Users
var UserNotFound = UserError("User not found", http.StatusNotFound)
var InvalidUserID = UserError(
	"Invalid user ID provided. It must be a valid UUID.",
	http.StatusBadRequest,
)
var QueryRequired = UserError(
	"At least one query parameter is required",
	http.StatusBadRequest,
)
