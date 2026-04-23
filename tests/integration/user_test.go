package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"ricehub/internal/testutil"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// registerUser returns user ID and access token in header-ready format: "Bearer <token>".
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

// makeAdminToken returns a signed "Bearer <token>" with IsAdmin=true for an
// existing user. The user must already exist in the DB so AdminMiddleware passes.
func makeAdminToken(t *testing.T, userID string) string {
	t.Helper()
	uid, err := uuid.Parse(userID)
	if err != nil {
		t.Fatalf("makeAdminToken: parse UUID %q: %v", userID, err)
	}
	return testutil.MakeAccessToken(t, uid, true)
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------
func TestListUsers_NoAccess(t *testing.T) {
	_, tok := registerUser(t, "listnoaccs", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/users", "", testutil.AuthHeader(tok))

	// non-admin users can only use username query param
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (QueryRequired) for listing users without query params, got %d", w.Code)
	}
}

func TestGetUserByUsername(t *testing.T) {
	_, _ = registerUser(t, "usernamequery", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/users?username=usernamequery", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for username query, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["username"] != "usernamequery" {
		t.Fatalf("expected username=usernamequery in response, got: %v", resp["username"])
	}
}

func TestListUsers_AsAdmin(t *testing.T) {
	id, _ := registerUser(t, "adminlistuser", "Password123!")
	adminTok := makeAdminToken(t, id)
	w := testutil.DoRequest(testApp, http.MethodGet, "/users", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin listing users, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListBannedUsers_AsAdmin(t *testing.T) {
	id, _ := registerUser(t, "adminbanlist", "Password123!")
	adminTok := makeAdminToken(t, id)
	w := testutil.DoRequest(testApp, http.MethodGet, "/users?status=banned", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin listing banned users, got %d: %s", w.Code, w.Body.String())
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

func TestGetOwnUser(t *testing.T) {
	id, tok := registerUser(t, "getownuser", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/users/"+id, "", testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 fetching own user, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["id"] != id {
		t.Fatalf("response ID mismatch: want %s, got %v", id, resp["id"])
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id/rices
// ---------------------------------------------------------------------------
func TestListUserRices_Public(t *testing.T) {
	id, _ := registerUser(t, "ricelistuser", "Password123!")

	w := testutil.DoRequest(testApp, http.MethodGet, "/users/"+id+"/rices", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id/rices/:slug  	 (practically /users/:username/[...])
// ---------------------------------------------------------------------------
func TestGetUserRiceBySlug(t *testing.T) {
	username := "slugowner"
	userID, tok := registerUser(t, username, "Password123!")
	riceID := createRice(t, userID, tok, "Slug Test Rice")

	// fetch rice to get its slug
	listW := testutil.DoRawRequest(testApp, http.MethodGet, "/users/"+userID+"/rices", nil, "", testutil.AuthHeader(tok))
	if listW.Code != http.StatusOK {
		t.Fatalf("list rices failed: %d %s", listW.Code, listW.Body.String())
	}
	var rices []map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &rices); err != nil {
		t.Fatalf("parse rices: %v", err)
	}
	var slug string
	for _, r := range rices {
		if r["id"] == riceID {
			slug, _ = r["slug"].(string)
			break
		}
	}
	if slug == "" {
		t.Fatal("could not find rice slug in user rices list")
	}

	w := testutil.DoRequest(
		testApp,
		http.MethodGet,
		"/users/"+username+"/rices/"+slug,
		"",
		testutil.AuthHeader(tok),
	)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for rice by slug, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id/purchased
// ---------------------------------------------------------------------------
func TestListPurchasedRices(t *testing.T) {
	id, tok := registerUser(t, "purchasedlist", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/users/"+id+"/purchased", "", testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for own purchased list, got %d: %s", w.Code, w.Body.String())
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

// ---------------------------------------------------------------------------
// PATCH /users/:id/password
// ---------------------------------------------------------------------------
func TestUpdatePassword_Success(t *testing.T) {
	id, tok := registerUser(t, "pwupdater", "OldPass123!")

	body := `{"oldPassword":"OldPass123!","newPassword":"NewPass456!"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/users/"+id+"/password", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePassword_WrongOld(t *testing.T) {
	id, tok := registerUser(t, "pwwrong", "Password123!")
	body := `{"oldPassword":"wrongpassword","newPassword":"NewPass456!"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/users/"+id+"/password", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong old password, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /users/:id/avatar
// ---------------------------------------------------------------------------
func TestUpdateAvatar_Success(t *testing.T) {
	id, tok := registerUser(t, "avatarupload", "Password123!")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "avatar.png")
	_, _ = fw.Write(testutil.TinyPNG(t))
	_ = mw.Close()

	w := testutil.DoRawRequest(testApp, http.MethodPost, "/users/"+id+"/avatar", &buf, mw.FormDataContentType(), testutil.AuthHeader(tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for avatar upload, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	avatarURL, _ := resp["avatarUrl"].(string)
	if avatarURL == "" {
		t.Fatal("avatarUrl missing from response")
	}

	idx := strings.Index(avatarURL, "/avatars/")
	if idx < 0 {
		t.Fatalf("avatarUrl has no /avatars/ segment: %s", avatarURL)
	}
	relPath := avatarURL[idx:]
	filename := filepath.Base(relPath)

	onDisk := filepath.Join(".", "public", "avatars", filename)
	t.Cleanup(func() { _ = os.Remove(onDisk) })

	if _, err := os.Stat(onDisk); err != nil {
		t.Fatalf("expected avatar on disk at %s, stat failed: %v", onDisk, err)
	}

	shouldNotExist := filepath.Join("..", "..", "public", "screenshots", filename)
	if _, err := os.Stat(shouldNotExist); err == nil {
		_ = os.Remove(shouldNotExist)
		t.Fatalf("avatar leaked into screenshots dir: %s", shouldNotExist)
	}
}

// ---------------------------------------------------------------------------
// DELETE /users/:id/avatar
// ---------------------------------------------------------------------------
func TestDeleteAvatar_Success(t *testing.T) {
	id, tok := registerUser(t, "avatardelete", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodDelete, "/users/"+id+"/avatar", "", testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for avatar delete, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["avatarUrl"] == nil {
		t.Fatal("avatarUrl missing from response")
	}
}

// ---------------------------------------------------------------------------
// DELETE /users/:id
// ---------------------------------------------------------------------------
func TestDeleteUser_Success(t *testing.T) {
	id, tok := registerUser(t, "userdelete", "Password123!")
	body := `{"password":"Password123!"}`
	w := testutil.DoRequest(testApp, http.MethodDelete, "/users/"+id, body, testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUser_WrongPassword(t *testing.T) {
	id, tok := registerUser(t, "userdelwrong", "Password123!")
	body := `{"password":"wrongpassword"}`
	w := testutil.DoRequest(testApp, http.MethodDelete, "/users/"+id, body, testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong password on delete, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /users/:id/ban  &  DELETE /users/:id/ban  (admin)
// ---------------------------------------------------------------------------
func TestBanAndUnbanUser(t *testing.T) {
	targetID, _ := registerUser(t, "bantarget", "Password123!")
	adminID, _ := registerUser(t, "banadmin", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	banBody := `{"reason":"Integration test ban reason"}`
	banW := testutil.DoRequest(testApp, http.MethodPost, "/users/"+targetID+"/ban", banBody, testutil.AuthHeader(adminTok))
	if banW.Code != http.StatusCreated {
		t.Fatalf("expected 201 for ban, got %d: %s", banW.Code, banW.Body.String())
	}

	unbanW := testutil.DoRequest(testApp, http.MethodDelete, "/users/"+targetID+"/ban", "", testutil.AuthHeader(adminTok))
	if unbanW.Code != http.StatusOK && unbanW.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for unban, got %d: %s", unbanW.Code, unbanW.Body.String())
	}
}

func TestBanUser_RequiresAdmin(t *testing.T) {
	targetID, _ := registerUser(t, "noadminban", "Password123!")
	_, normalTok := registerUser(t, "noadminbanner", "Password123!")
	banBody := `{"reason":"Should not work"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/users/"+targetID+"/ban", banBody, testutil.AuthHeader(normalTok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin ban attempt, got %d: %s", w.Code, w.Body.String())
	}
}
