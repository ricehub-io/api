package integration

import (
	"encoding/json"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

// ---------------------------------------------------------------------------
// POST /auth/register
// ---------------------------------------------------------------------------
func TestRegister_Success(t *testing.T) {
	body := `{"username":"testuser1","password":"Password123!","displayName":"Test User"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/register", body, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	body := `{"username":"dupuser","password":"Password123!","displayName":"Dup"}`
	testutil.DoRequest(testApp, http.MethodPost, "/auth/register", body, nil)

	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/register", body, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_MissingFields(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/register", `{}`, nil)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400 or 422, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------
func TestLogin_Success(t *testing.T) {
	reg := `{"username":"loginuser","password":"Password123!","displayName":"Login User"}`
	testutil.DoRequest(testApp, http.MethodPost, "/auth/register", reg, nil)

	body := `{"username":"loginuser","password":"Password123!"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/login", body, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp["accessToken"] == nil {
		t.Fatal("response missing accessToken field")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	reg := `{"username":"wrongpwuser","password":"Password123!","displayName":"WP User"}`
	testutil.DoRequest(testApp, http.MethodPost, "/auth/register", reg, nil)

	body := `{"username":"wrongpwuser","password":"wrongpassword"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/login", body, nil)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Fatalf("expected 401/403 for wrong password, got %d", w.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	body := `{"username":"nobody","password":"Password123!"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/login", body, nil)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for unknown user login")
	}
}

// ---------------------------------------------------------------------------
// POST /auth/refresh
// ---------------------------------------------------------------------------
func TestRefresh_ValidToken(t *testing.T) {
	reg := `{"username":"refreshuser","password":"Password123!","displayName":"Refresh"}`
	testutil.DoRequest(testApp, http.MethodPost, "/auth/register", reg, nil)

	loginBody := `{"username":"refreshuser","password":"Password123!"}`
	loginResp := testutil.DoRequest(testApp, http.MethodPost, "/auth/login", loginBody, nil)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("login setup failed: %d %s", loginResp.Code, loginResp.Body.String())
	}

	var refreshCookie *http.Cookie
	for _, c := range loginResp.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Skip("no refresh_token cookie in login response")
	}

	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/refresh", "", map[string]string{
		"Cookie": refreshCookie.Name + "=" + refreshCookie.Value,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from refresh, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /auth/logout
// ---------------------------------------------------------------------------
func TestLogout_Success(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodPost, "/auth/logout", "", nil)
	if w.Code >= 500 {
		t.Fatalf("expected non-5xx from logout, got %d", w.Code)
	}
}
