package integration

import (
	"encoding/json"
	"fmt"
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
// POST /tags (admin)
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

// ---------------------------------------------------------------------------
// Admin CRUD: POST -> PATCH -> DELETE
// ---------------------------------------------------------------------------
func TestTagCRUD_AsAdmin(t *testing.T) {
	adminID, _ := registerUser(t, "tagcrudadmin", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	// create
	tagName := testutil.RandString(8)
	createBody := fmt.Sprintf(`{"name":%q}`, tagName)
	createW := testutil.DoRequest(testApp, http.MethodPost, "/tags", createBody, testutil.AuthHeader(adminTok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201 for tag create, got %d: %s", createW.Code, createW.Body.String())
	}
	var tag map[string]any
	if err := json.Unmarshal(createW.Body.Bytes(), &tag); err != nil {
		t.Fatalf("parse create tag response: %v", err)
	}
	tagID := int(tag["id"].(float64))

	// update
	updateBody := fmt.Sprintf(`{"name":%q}`, tagName+"upd")
	updateW := testutil.DoRequest(testApp, http.MethodPatch, fmt.Sprintf("/tags/%d", tagID), updateBody, testutil.AuthHeader(adminTok))
	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200 for tag update, got %d: %s", updateW.Code, updateW.Body.String())
	}

	// delete
	deleteW := testutil.DoRequest(testApp, http.MethodDelete, fmt.Sprintf("/tags/%d", tagID), "", testutil.AuthHeader(adminTok))
	if deleteW.Code != http.StatusOK && deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for tag delete, got %d: %s", deleteW.Code, deleteW.Body.String())
	}
}
