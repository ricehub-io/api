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
	_ = zw.Close()

	_ = mw.Close()

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

func createRiceAsAdmin(t *testing.T, userID, title string) (riceID, adminTok string) {
	t.Helper()
	adminTok = makeAdminToken(t, userID)
	riceID = createRice(t, userID, adminTok, title)
	return riceID, adminTok
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

func TestGetRice_Success(t *testing.T) {
	userID, _ := registerUser(t, "ricegetowner", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Accepted Rice For Get")

	w := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID, "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for accepted rice, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["id"] != riceID {
		t.Fatalf("rice ID mismatch: want %s, got %v", riceID, resp["id"])
	}
}

func TestGetRice_WaitingNotVisibleToPublic(t *testing.T) {
	userID, tok := registerUser(t, "waitingowner", "Password123!")
	riceID := createRice(t, userID, tok, "Waiting Rice")

	w := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID, "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for waiting rice to public, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /rices/:id/comments
// ---------------------------------------------------------------------------
func TestListRiceComments(t *testing.T) {
	userID, tok := registerUser(t, "cmntlistown", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Rice For Comment List")

	commentID := createComment(t, tok, riceID, "a comment for list test")
	if commentID == "" {
		t.Fatal("createComment did not return ID")
	}

	w := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID+"/comments", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for rice comments, got %d: %s", w.Code, w.Body.String())
	}
	var comments []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &comments); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(comments) == 0 {
		t.Fatal("expected at least one comment in list")
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
		t.Fatal("createRice did not return an ID")
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
		t.Fatal("createRice did not return an ID")
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
		t.Fatal("createRice did not return an ID")
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
		t.Fatal("createRice did not return an ID")
	}

	w := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+id, "", testutil.AuthHeader(otherTok))
	if w.Code == http.StatusOK || w.Code == http.StatusNoContent {
		t.Fatal("other user should not be able to delete rice")
	}
}

// ---------------------------------------------------------------------------
// PATCH /rices/:id/state (admin)
// ---------------------------------------------------------------------------
func TestUpdateRiceState_AsAdmin(t *testing.T) {
	userID, tok := registerUser(t, "stateowner", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For State Change")
	adminTok := makeAdminToken(t, userID)

	body := `{"newState":"accepted"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/rices/"+riceID+"/state", body, testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for admin accept, got %d: %s", w.Code, w.Body.String())
	}

	getW := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID, "", nil)
	if getW.Code != http.StatusOK {
		t.Fatalf("rice not visible after accept, got %d: %s", getW.Code, getW.Body.String())
	}
}

func TestUpdateRiceState_RequiresAdmin(t *testing.T) {
	userID, tok := registerUser(t, "statenoadmin", "Password123!")
	riceID := createRice(t, userID, tok, "Rice State No Admin")
	body := `{"newState":"accepted"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/rices/"+riceID+"/state", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin state change, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /rices?state=waiting (admin)
// ---------------------------------------------------------------------------
func TestListWaitingRices_AsAdmin(t *testing.T) {
	adminID, _ := registerUser(t, "listwaitadm", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	w := testutil.DoRequest(testApp, http.MethodGet, "/rices?state=waiting", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin waiting list, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /rices/:id/star  &  DELETE /rices/:id/star
// ---------------------------------------------------------------------------
func TestRiceStarAndUnstar(t *testing.T) {
	ownerID, ownerTok := registerUser(t, "starowner", "Password123!")
	riceID, _ := createRiceAsAdmin(t, ownerID, "Rice To Star")

	_, starTok := registerUser(t, "starruser", "Password123!")

	starW := testutil.DoRequest(testApp, http.MethodPost, "/rices/"+riceID+"/star", "", testutil.AuthHeader(starTok))
	if starW.Code != http.StatusCreated {
		t.Fatalf("expected 201 for star, got %d: %s", starW.Code, starW.Body.String())
	}

	unstarW := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+riceID+"/star", "", testutil.AuthHeader(starTok))
	if unstarW.Code != http.StatusOK && unstarW.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for unstar, got %d: %s", unstarW.Code, unstarW.Body.String())
	}

	_ = ownerTok
}

// ---------------------------------------------------------------------------
// POST /rices/:id/tags  &  DELETE /rices/:id/tags
// ---------------------------------------------------------------------------
func TestAddAndRemoveRiceTags(t *testing.T) {
	userID, tok := registerUser(t, "tagowner", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Tags")

	// unaimeds: tags automatically seeded from schema.sql
	addBody := `{"tags":[1,2]}`
	addW := testutil.DoRequest(testApp, http.MethodPost, "/rices/"+riceID+"/tags", addBody, testutil.AuthHeader(tok))
	if addW.Code != http.StatusOK {
		t.Fatalf("expected 200 for add tags, got %d: %s", addW.Code, addW.Body.String())
	}

	removeBody := `{"tags":[1]}`
	removeW := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+riceID+"/tags", removeBody, testutil.AuthHeader(tok))
	if removeW.Code != http.StatusOK && removeW.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for remove tags, got %d: %s", removeW.Code, removeW.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /rices/:id/screenshots
// ---------------------------------------------------------------------------
func TestAddScreenshot(t *testing.T) {
	userID, tok := registerUser(t, "scrowner", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Screenshots")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files[]", "extra.png")
	_, _ = fw.Write([]byte("fake-png-data"))
	_ = mw.Close()

	w := testutil.DoRawRequest(testApp, http.MethodPost, "/rices/"+riceID+"/screenshots", &buf, mw.FormDataContentType(), testutil.AuthHeader(tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for add screenshot, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["screenshots"] == nil {
		t.Fatal("screenshots missing from response")
	}
}

// ---------------------------------------------------------------------------
// DELETE /rices/:id/screenshots/:screenshotId
// ---------------------------------------------------------------------------
func TestDeleteScreenshot(t *testing.T) {
	userID, tok := registerUser(t, "screenshotdel", "Password123!")
	riceID := createRice(t, userID, tok, "Rice for screenshot delete")

	// add a second screenshot so we can delete one (min 1 required)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files[]", "extra2.png")
	_, _ = fw.Write([]byte("fake-png-data"))
	_ = mw.Close()
	addW := testutil.DoRawRequest(testApp, http.MethodPost, "/rices/"+riceID+"/screenshots", &buf, mw.FormDataContentType(), testutil.AuthHeader(tok))
	if addW.Code != http.StatusCreated {
		t.Fatalf("add screenshot setup failed: %d %s", addW.Code, addW.Body.String())
	}
	var addResp map[string]any
	if err := json.Unmarshal(addW.Body.Bytes(), &addResp); err != nil {
		t.Fatalf("parse add screenshot response: %v", err)
	}
	screenshots, _ := addResp["screenshots"].([]any)
	if len(screenshots) == 0 {
		t.Fatal("no screenshots in add response")
	}

	// fetch rice to get all screenshot ids
	riceW := testutil.DoRequest(testApp, http.MethodGet, "/users/"+userID+"/rices", "", testutil.AuthHeader(tok))
	if riceW.Code != http.StatusOK {
		t.Fatalf("list rices failed: %d", riceW.Code)
	}
	var rices []map[string]any
	if err := json.Unmarshal(riceW.Body.Bytes(), &rices); err != nil {
		t.Fatalf("parse rices: %v", err)
	}

	// get the first screenshot id from the rice detail
	riceDetailW := testutil.DoRawRequest(testApp, http.MethodGet, "/users/"+userID+"/rices", nil, "", testutil.AuthHeader(tok))
	_ = riceDetailW

	// get screenshot id via the admin full rice
	adminTok := makeAdminToken(t, userID)
	detailW := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID, "", testutil.AuthHeader(adminTok))
	if detailW.Code != http.StatusOK {
		t.Fatalf("get rice detail failed: %d %s", detailW.Code, detailW.Body.String())
	}
	var rice map[string]any
	if err := json.Unmarshal(detailW.Body.Bytes(), &rice); err != nil {
		t.Fatalf("parse rice: %v", err)
	}
	scrs, _ := rice["screenshots"].([]any)
	if len(scrs) < 2 {
		t.Fatalf("expected at least 2 screenshots, got %d", len(scrs))
	}
	firstScr, _ := scrs[0].(map[string]any)
	screenshotID, _ := firstScr["id"].(string)
	if screenshotID == "" {
		t.Fatal("screenshot ID missing from rice detail")
	}

	delW := testutil.DoRequest(testApp, http.MethodDelete, "/rices/"+riceID+"/screenshots/"+screenshotID, "", testutil.AuthHeader(tok))
	if delW.Code != http.StatusOK && delW.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for screenshot delete, got %d: %s", delW.Code, delW.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /rices/:id/dotfiles
// ---------------------------------------------------------------------------
func TestUpdateDotfiles(t *testing.T) {
	userID, tok := registerUser(t, "dfupdater", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Dotfiles Update")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "newdotfiles.zip")
	zw := zip.NewWriter(fw)
	_, _ = zw.Create("newfile.txt")
	_ = zw.Close()
	_ = mw.Close()

	w := testutil.DoRawRequest(testApp, http.MethodPost, "/rices/"+riceID+"/dotfiles", &buf, mw.FormDataContentType(), testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for dotfiles update, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// PATCH /rices/:id/dotfiles/type
// ---------------------------------------------------------------------------
func TestUpdateDotfilesType_FreeNoOp(t *testing.T) {
	userID, tok := registerUser(t, "dotfilestype", "Password123!")
	riceID := createRice(t, userID, tok, "Rice For Dotfiles Type")

	body := `{"newType":"free"}`
	w := testutil.DoRequest(testApp, http.MethodPatch, "/rices/"+riceID+"/dotfiles/type", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for dotfiles type update, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /rices/:id/dotfiles
// ---------------------------------------------------------------------------
func TestDownloadDotfiles_Free(t *testing.T) {
	userID, _ := registerUser(t, "dfdownl", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Rice For Download")

	w := testutil.DoRequest(testApp, http.MethodGet, "/rices/"+riceID+"/dotfiles", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for free dotfiles download, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Disposition") == "" {
		t.Fatal("expected Content-Disposition header for file download")
	}
}
