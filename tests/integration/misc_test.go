package integration

import (
	"net/http"
	"ricehub/internal/testutil"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /leaderboard/[week|month|year]
// ---------------------------------------------------------------------------
func TestLeaderboard_Week(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/leaderboard/week", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for weekly leaderboard, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLeaderboard_Month(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/leaderboard/month", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for monthly leaderboard, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLeaderboard_Year(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/leaderboard/year", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for yearly leaderboard, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /vars/:key
// ---------------------------------------------------------------------------
func TestGetWebVar_Success(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/vars/terms_of_service_text", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for seeded webvar, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWebVar_NotFound(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/vars/no_such_key", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing webvar, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /links/:name
// ---------------------------------------------------------------------------
func TestGetLink_Success(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/links/discord", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for seeded link, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetLink_NotFound(t *testing.T) {
	w := testutil.DoRequest(testApp, http.MethodGet, "/links/nosuchlink", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing link, got %d: %s", w.Code, w.Body.String())
	}
}
