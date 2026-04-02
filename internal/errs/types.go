package errs

import (
	"net/http"
	"strings"
)

type AppError struct {
	Code     int
	Messages []string
	Err      error
}

func (e *AppError) Error() string {
	return strings.Join(e.Messages, ", ")
}

func InternalError(err error) *AppError {
	return &AppError{
		Code:     http.StatusInternalServerError,
		Messages: []string{"Unexpected internal server error occurred"},
		Err:      err,
	}
}

func UserError(message string, code int) *AppError {
	return &AppError{
		Code:     code,
		Messages: []string{message},
	}
}

func UserErrors(messages []string, code int) *AppError {
	return &AppError{
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
var BlacklistedDisplayName = UserError(
	"Username contains blacklisted phrases",
	http.StatusUnprocessableEntity,
)

// Auth
var InvalidCredentials = UserError(
	"Invalid credentials provided",
	http.StatusUnauthorized,
)

// Comments
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
