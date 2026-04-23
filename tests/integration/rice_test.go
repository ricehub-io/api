package integration

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

func createRice(t *testing.T, userID, tok, title string) string {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	_ = mw.WriteField("title", title)
	_ = mw.WriteField("description", "integration test rice description")

	fw, _ := mw.CreateFormFile("screenshots[]", "shot.png")
	_, _ = fw.Write([]byte("fake-png-data"))

	fw, _ = mw.CreateFormFile("dotfiles", "dotfiles.zip")
	zw := zip.NewWriter(fw)
	_, _ = zw.Create("placeholder.txt")
	zw.Close()

	mw.Close()

	w := testutil.DoRawRequest(testApp, http.MethodPost, "/rices", &buf, mw.FormDataContentType(), testutil.AuthHeader(tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("createRice failed: %d %s", w.Code, w.Body.String())
	}

	listW := testutil.DoRawRequest(testApp, http.MethodGet, fmt.Sprintf("/users/%s/rices", userID), nil, "", testutil.AuthHeader(tok))
	if listW.Code != http.StatusOK {
		t.Fatalf("list user rices failed: %d %s", listW.Code, listW.Body.String())
	}

	var rices []map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &rices); err != nil {
		t.Fatalf("parse rices JSON: %v", err)
	}
	for _, r := range rices {
		if r["title"] == title {
			id, _ := r["id"].(string)
			return id
		}
	}

	t.Fatalf("rice %q not found in user's rice list after creation", title)
	return ""
}

// ---------------------------------------------------------------------------
// GET /rices (public)
// ---------------------------------------------------------------------------
func TestListRices_Public(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/rices", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /rices/:id
// ---------------------------------------------------------------------------
func TestGetRice_NotFound(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/rices/00000000-0000-0000-0000-000000000000", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown rice, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /rices (requires auth)
// ---------------------------------------------------------------------------
func TestCreateRice_RequiresAuth(t *testing.T) {
	body := `{"title":"My Rice","description":"test"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/rices", body, nil)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403 without auth, got %d", w.Code)
	}
}

func TestCreateAndGetRice(t *testing.T) {
	userID, tok := registerUser(t, "riceauthor", "Password123!")
	id := createRice(t, userID, tok, "My Awesome Rice")
	if id == "" {
		t.Skip("create rice did not return an ID")
	}

	listW := testutil.DoRawRequest(testApp, http.MethodGet, "/users/"+userID+"/rices", nil, "", testutil.AuthHeader(tok))
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 from user rice list, got %d: %s", listW.Code, listW.Body.String())
	}
}

func TestUpdateRiceMetadata(t *testing.T) {
	userID, tok := registerUser(t, "riceupdater", "Password123!")
	id := createRice(t, userID, tok, "Rice To Update")
	if id == "" {
		t.Skip("create rice did not return an ID")
	}

	body := `{"title":"Updated Title","description":"updated desc"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/rices/"+id, body, testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 on update, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRice(t *testing.T) {
	userID, tok := registerUser(t, "ricedeleter", "Password123!")
	id := createRice(t, userID, tok, "Rice To Delete")
	if id == "" {
		t.Skip("create rice did not return an ID")
	}

	w := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+id, "", testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 on delete, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRice_OtherUser(t *testing.T) {
	ownerID, ownerTok := registerUser(t, "riceowner", "Password123!")
	_, otherTok := registerUser(t, "ricestealer", "Password123!")

	id := createRice(t, ownerID, ownerTok, "Protected Rice")
	if id == "" {
		t.Skip("create rice did not return an ID")
	}

	w := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+id, "", testutil.AuthHeader(otherTok))
	if w.Code == http.StatusOK || w.Code == http.StatusNoContent {
		t.Fatal("other user should not be able to delete rice")
	}
}
