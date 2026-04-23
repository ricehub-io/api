package integration

import (
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /profiles/:username
// ---------------------------------------------------------------------------
func TestGetProfile_Success(t *testing.T) {
	_, _ = registerUser(t, "profileuser", "Password123!")

	w := testutil.DoRequest(testApp, http.MethodGet, "/profiles/profileuser", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing profile, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/profiles/doesnotexist", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown profile, got %d: %s", w.Code, w.Body.String())
	}
}
