package integration

import (
	"encoding/json"
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /admin/stats
// ---------------------------------------------------------------------------
func TestAdminStats_AsAdmin(t *testing.T) {
	adminID, _ := registerUser(t, "statsadmin", "Password123!")
	adminTok := makeAdminToken(t, adminID)

	w := testutil.DoRequest(testApp, http.MethodGet, "/admin/stats", "", testutil.AuthHeader(adminTok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin stats, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, field := range []string{"userCount", "riceCount", "commentCount", "reportCount"} {
		if resp[field] == nil {
			t.Fatalf("stats response missing field %q", field)
		}
	}
}

func TestAdminStats_RequiresAdmin(t *testing.T) {
	_, tok := registerUser(t, "statsnonadmin", "Password123!")
	w := testutil.DoRequest(testApp, http.MethodGet, "/admin/stats", "", testutil.AuthHeader(tok))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin stats, got %d: %s", w.Code, w.Body.String())
	}
}
