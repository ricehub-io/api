package errs

import (
	"errors"
	"net/http"
	"testing"
)

// #################################################
// ########### AppError (constructors) #############
// #################################################
func TestAppError_Error_SingleMessage(t *testing.T) {
	err := &appError{Messages: []string{"something went wrong"}}
	if err.Error() != "something went wrong" {
		t.Errorf("unexpected: %q", err.Error())
	}
}

func TestAppError_Error_MultipleMessages(t *testing.T) {
	err := &appError{
		Messages: []string{
			"field a is required",
			"field b is required",
		},
	}

	got := err.Error()
	want := "field a is required, field b is required"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestUserError_SetsCodeAndMessage(t *testing.T) {
	err := UserError("not found", http.StatusNotFound).(*appError)

	if err.Code != http.StatusNotFound {
		t.Errorf("want code 404, got %d", err.Code)
	}

	if len(err.Messages) != 1 || err.Messages[0] != "not found" {
		t.Errorf("unexpected messages: %v", err.Messages)
	}

	if err.Err != nil {
		t.Error("UserError should not wrap an internal error")
	}
}

func TestUserErrors_SetsMultipleMessages(t *testing.T) {
	msgs := []string{"name too short", "email invalid"}

	err := UserErrors(msgs, http.StatusBadRequest).(*appError)
	if err.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", err.Code)
	}

	if len(err.Messages) != 2 {
		t.Errorf("want 2 messages, got %d", len(err.Messages))
	}
}

func TestInternalError_WrapsOriginalError(t *testing.T) {
	original := errors.New("db connection lost")
	err := InternalError(original)

	if err.StatusCode() != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", err.StatusCode())
	}

	if !errors.Is(err, original) {
		t.Error("expected wrapped error to be the original")
	}
}

func TestInternalError_MessageIsGeneric(t *testing.T) {
	err := InternalError(errors.New("some db error")).(*appError)

	// must not leak internal details to the caller
	if len(err.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(err.Messages))
	}

	if err.Messages[0] == "some db error" {
		t.Error("internal error message should not leak the raw error string")
	}
}
