package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

func registerUser(t *testing.T, username, password string) (id, token string) {
	t.Helper()

	reg := fmt.Sprintf(`{"username":%q,"password":%q,"displayName":%q}`, username, password, username)
	regResp := testutil.DoRequest(testApp, http.MethodPost, "/auth/register", reg, nil)
	if regResp.Code != http.StatusCreated {
		t.Fatalf("register %q failed: %d %s", username, regResp.Code, regResp.Body.String())
	}

	login := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	loginResp := testutil.DoRequest(testApp, http.MethodPost, "/auth/login", login, nil)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login %q failed: %d %s", username, loginResp.Code, loginResp.Body.String())
	}

	var loginData map[string]any
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginData); err != nil {
		t.Fatalf("login response JSON: %v", err)
	}
	tok, _ := loginData["accessToken"].(string)
	if tok == "" {
		t.Fatal("login response missing accessToken")
	}

	// unaimeds: xd
	id = loginData["user"].(map[string]any)["id"].(string)
	return id, "Bearer " + tok
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------
func TestListUsers_NoAccess(t *testing.T) {
	_, tok := registerUser(t, "listnoaccs", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/users", "", testutil.AuthHeader(tok))

	// unaimeds: because non-admin users can only query single user by username
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for listing users, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id
// ---------------------------------------------------------------------------
func TestGetUser_RequiresAuth(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/users/some-uuid", "", nil)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403 without auth, got %d", w.Code)
	}
}

func TestGetUser_NoAccess(t *testing.T) {
	_, tok := registerUser(t, "getnoaccs", "Password123!")
	w := testutil.DoRequest(
		testApp,
		http.MethodGet,
		"/users/00000000-0000-0000-0000-000000000000",
		"",
		testutil.AuthHeader(tok),
	)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for inaccessible user, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id/rices
// ---------------------------------------------------------------------------
func TestListUserRices_Public(t *testing.T) {
	id, _ := registerUser(t, "ricelistuser", "Password123!")
	if id == "" {
		t.Skip("register did not return user ID")
	}

	w := testutil.DoRequest(testApp, http.MethodGet, "/users/"+id+"/rices", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// PATCH /users/:id/displayName
// ---------------------------------------------------------------------------
func TestUpdateDisplayName_Success(t *testing.T) {
	id, tok := registerUser(t, "dnameself", "Password123!")

	body := `{"displayName":"New Display Name"}`
	w := testutil.DoRequest(
		testApp,
		http.MethodPatch,
		"/users/"+id+"/displayName",
		body,
		testutil.AuthHeader(tok),
	)
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}
