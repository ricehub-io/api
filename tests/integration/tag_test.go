package integration

import (
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /tags
// ---------------------------------------------------------------------------
func TestListTags_Public(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/tags", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /tags (admin only)
// ---------------------------------------------------------------------------
func TestCreateTag_RequiresAdmin(t *testing.T) {
	_, tok := registerUser(t, "tagnonAdmin", "Password123!")
	body := `{"name":"mytag"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/tags", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403 for non-admin, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTag_RequiresAuth(t *testing.T) {
	body := `{"name":"unauthedtag"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/tags", body, nil)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403 without auth, got %d", w.Code)
	}
}
