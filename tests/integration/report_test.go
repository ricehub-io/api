package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

func createReport(t *testing.T, tok, riceID string) string {
	t.Helper()
	body := fmt.Sprintf(`{"riceId":%q,"reason":"this is a valid report reason"}`, riceID)
	w := testutil.DoRequest(testApp, http.MethodPost, "/reports", body, testutil.AuthHeader(tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("createReport failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("createReport JSON: %v", err)
	}
	id, _ := resp["reportId"].(string)
	return id
}

// ---------------------------------------------------------------------------
// POST /reports
// ---------------------------------------------------------------------------
func TestCreateReport_RequiresAuth(t *testing.T) {
	body := `{"riceId":"00000000-0000-0000-0000-000000000000","reason":"test report reason"}`
	w := testutil.DoRequest(testApp, http.MethodPost, "/reports", body, nil)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/403, got %d", w.Code)
	}
}

func TestCreateReport_Success(t *testing.T) {
	userID, tok := registerUser(t, "reportcreator", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Reported Rice")

	reportID := createReport(t, tok, riceID)
	if reportID == "" {
		t.Fatal("createReport did not return ID")
	}
}

// ---------------------------------------------------------------------------
// GET /reports (admin)
// ---------------------------------------------------------------------------
func TestListReports_AsAdmin(t *testing.T) {
	adminID, _ := registerUser(t, "rptlistadmin", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	w := testutil.DoRequest(testApp, http.MethodGet, "/reports", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin reports list, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListReports_RequiresAdmin(t *testing.T) {
	_, tok := registerUser(t, "rptlistnonadm", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/reports", "", testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin reports list, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /reports/:id (admin)
// ---------------------------------------------------------------------------
func TestGetReportByID_AsAdmin(t *testing.T) {
	userID, tok := registerUser(t, "reportgetowner", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Rice For Report Get")
	reportID := createReport(t, tok, riceID)
	if reportID == "" {
		t.Fatal("createReport did not return ID")
	}

	adminTok := makeAdminToken(t, userID)
	w := testutil.DoRequest(testApp, http.MethodGet, "/reports/"+reportID, "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get report by ID, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["id"] != reportID {
		t.Fatalf("report ID mismatch: want %s, got %v", reportID, resp["id"])
	}
}

// ---------------------------------------------------------------------------
// POST /reports/:id/close (admin)
// ---------------------------------------------------------------------------
func TestCloseReport_AsAdmin(t *testing.T) {
	userID, tok := registerUser(t, "rptcloseown", "Password123!")
	riceID, _ := createRiceAsAdmin(t, userID, "Rice For Report Close")
	reportID := createReport(t, tok, riceID)
	if reportID == "" {
		t.Fatal("createReport did not return ID")
	}

	adminTok := makeAdminToken(t, userID)
	w := testutil.DoRequest(testApp, http.MethodPost, "/reports/"+reportID+"/close", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("expected 200/204 for close report, got %d: %s", w.Code, w.Body.String())
	}
}
