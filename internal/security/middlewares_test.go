package security

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func makeValidBearerToken(t *testing.T, isAdmin, hasSubscription bool) string {
	t.Helper()

	tokenStr, err := NewAccessToken(uuid.New(), isAdmin, hasSubscription)
	if err != nil {
		t.Fatalf("could not create access token: %v", err)
	}

	return "Bearer " + tokenStr
}

func makeExpiredBearerToken(t *testing.T) string {
	t.Helper()

	claims := &AccessToken{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(accessPriv)
	if err != nil {
		t.Fatalf("could not sign expired token: %v", err)
	}

	return "Bearer " + tokenStr
}

// #################################################
// ################ ValidateToken ##################
// #################################################
func TestValidateToken_EmptyString_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	_, err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_WhitespaceOnly_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	// AuthMiddleware trims whitespace before calling ValidateToken
	_, err := ValidateToken("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only token")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_NoBearerPrefix_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	_, err := ValidateToken("sometoken")
	if err == nil {
		t.Fatal("expected error for missing Bearer prefix")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
	if !strings.Contains(err.Error(), "Bearer") {
		t.Errorf("error message should mention 'Bearer', got: %s", err.Error())
	}
}

func TestValidateToken_LowercaseBearer_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	// CutPrefix is case-sensitive thus "bearer " must not be accepted
	_, err := ValidateToken("bearer sometoken")
	if err == nil {
		t.Fatal("expected error for lowercase 'bearer' prefix")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_BearerWithNoToken_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	// "Bearer " is there but we missing the token bro
	_, err := ValidateToken("Bearer ")
	if err == nil {
		t.Fatal("expected error for 'Bearer ' with no token")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_GarbageToken_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	_, err := ValidateToken("Bearer not.a.real.jwt")
	if err == nil {
		t.Fatal("expected error for garbage JWT")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_TamperedToken_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	header := makeValidBearerToken(t, false, false)
	_, err := ValidateToken(header + "tampered")
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
}

func TestValidateToken_ExpiredToken_ReturnsForbidden(t *testing.T) {
	initTestKeys(t)

	_, err := ValidateToken(makeExpiredBearerToken(t))
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("want 403, got %d", err.StatusCode())
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error message should mention 'expired', got: %s", err.Error())
	}
}

func TestValidateToken_ValidToken_ReturnsToken(t *testing.T) {
	initTestKeys(t)

	_, err := ValidateToken(makeValidBearerToken(t, false, false))
	if err != nil {
		t.Fatalf("expected no error for valid token, got: %v", err)
	}
}

func TestValidateToken_ValidToken_PreservesSubject(t *testing.T) {
	initTestKeys(t)

	userID := uuid.New()
	tokenStr, _ := NewAccessToken(userID, false, false)

	token, err := ValidateToken("Bearer " + tokenStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.Subject != userID.String() {
		t.Errorf("want subject %s, got %s", userID, token.Subject)
	}
}

func TestValidateToken_ValidToken_PreservesIsAdmin(t *testing.T) {
	initTestKeys(t)

	token, err := ValidateToken(makeValidBearerToken(t, true, false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !token.IsAdmin {
		t.Error("expected IsAdmin to be true")
	}
}

func TestValidateToken_ValidToken_PreservesHasSubscription(t *testing.T) {
	initTestKeys(t)

	token, err := ValidateToken(makeValidBearerToken(t, false, true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !token.HasSubscription {
		t.Error("expected HasSubscription to be true")
	}
}
