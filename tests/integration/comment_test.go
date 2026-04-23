package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

func createComment(t *testing.T, tok, riceID, content string) string {
	t.Helper()

	body := fmt.Sprintf(`{"riceId":%q,"content":%q}`, riceID, content)
	w := testutil.DoRequest(testApp, http.MethodPost, "/comments", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("createComment failed: %d %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("createComment JSON: %v", err)
	}
	id, _ := resp["id"].(string)
	return id
}

// ---------------------------------------------------------------------------
// POST /comments
// ---------------------------------------------------------------------------
func TestCreateComment_RequiresAuth(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodPost, "/comments",
		`{"riceId":"some-id","content":"hello"}`, nil)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403, got %d", w.Code)
	}
}

func TestCreateAndDeleteComment(t *testing.T) {
	userID, tok := registerUser(t, "commenter", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Comments")
	if riceID == "" {
		t.Fatal("createRice did not return ID")
	}

	commentID := createComment(t, tok, riceID, "great rice here!")
	if commentID == "" {
		t.Fatal("createComment did not return ID")
	}

	w := testutil.DoRequest(testApp, http.MethodDelete, "/comments/"+commentID, "",
		testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 on delete, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateComment(t *testing.T) {
	userID, tok := registerUser(t, "commentupdater", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Comment Update")
	if riceID == "" {
		t.Fatal("createRice did not return ID")
	}

	commentID := createComment(t, tok, riceID, "original comment text")
	if commentID == "" {
		t.Fatal("createComment did not return ID")
	}

	body := `{"content":"updated comment text"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/comments/"+commentID, body,
		testutil.AuthHeader(tok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 on update, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /comments/:id
// ---------------------------------------------------------------------------
func TestGetCommentByID(t *testing.T) {
	userID, tok := registerUser(t, "commentgetter", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Get Comment")
	if riceID == "" {
		t.Fatal("createRice did not return ID")
	}
	commentID := createComment(t, tok, riceID, "fetchable comment text")
	if commentID == "" {
		t.Fatal("createComment did not return ID")
	}

	w := testutil.DoRequest(testApp, http.MethodGet, "/comments/"+commentID, "", testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get comment by ID, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["id"] != commentID {
		t.Fatalf("comment ID mismatch: want %s, got %v", commentID, resp["id"])
	}
}

// ---------------------------------------------------------------------------
// GET /comments (admin)
// ---------------------------------------------------------------------------
func TestListComments_AsAdmin(t *testing.T) {
	adminID, _ := registerUser(t, "cmntlistadmin", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	w := testutil.DoRequest(testApp, http.MethodGet, "/comments", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin comment list, got %d: %s", w.Code, w.Body.String())
	}
}
