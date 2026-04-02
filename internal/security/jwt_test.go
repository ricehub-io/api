package security

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"ricehub/internal/config"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const accessPrivPEM = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgGSpGIiazFUSDONm0
XupELbMQbBnSmgocwcx+o0uTIWihRANCAATMogo6VBDQanJ+X2ZZjbn1V1+UN3re
WRYdG2kLyfjxERaKQJhiuPBUCN+itdyjXbrZNDC+Jf4SQa8fpdxy2X3P
-----END PRIVATE KEY-----

`
const accessPubPEM = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEzKIKOlQQ0Gpyfl9mWY259VdflDd6
3lkWHRtpC8n48REWikCYYrjwVAjforXco1262TQwviX+EkGvH6Xcctl9zw==
-----END PUBLIC KEY-----
`

const refreshPrivPEM = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgIeVCwhYdKkOuPW5M
HubmLnEL+HX90x/TkPUvyV2vvMqhRANCAASzMGtrh1E7zxxGsrbUWDjKAPIfUSBy
/U1Gjm1CaFza4QBuy8qR3h08njRT/IFSB4PH6SP0qbAWJhZxvv4sWK0s
-----END PRIVATE KEY-----
`

const refreshPubPEM = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEszBra4dRO88cRrK21Fg4ygDyH1Eg
cv1NRo5tQmhc2uEAbsvKkd4dPJ40U/yBUgeDx+kj9KmwFiYWcb7+LFitLA==
-----END PUBLIC KEY-----
`

func init() {
	// initialize config variables that are used by tests
	config.Config.JWT.AccessExpiration = 5 * time.Minute
	config.Config.JWT.RefreshExpiration = 24 * time.Hour
	config.Config.Server.KeysDir = os.TempDir()
}

// writePEM writes a provided content string to a temporary '.pem' file and returns its path.
func writePEM(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "*.pem")
	if err != nil {
		t.Fatalf("failed to create a temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	path, _ := strings.CutPrefix(f.Name(), os.TempDir()+"/")
	return path
}

// initTestKeys initializes key-pair variables so that tests
// that call New*Token / Decode* don't need real key files.
func initTestKeys(t *testing.T) {
	t.Helper()

	parsePriv := func(pemStr string) *ecdsa.PrivateKey {
		block, _ := pem.Decode([]byte(pemStr))
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("parsePriv: %v", err)
		}
		return key.(*ecdsa.PrivateKey)
	}
	parsePub := func(pemStr string) *ecdsa.PublicKey {
		block, _ := pem.Decode([]byte(pemStr))
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			t.Fatalf("parsePub: %v", err)
		}
		return key.(*ecdsa.PublicKey)
	}

	accessPriv = parsePriv(accessPrivPEM)
	accessPub = parsePub(accessPubPEM)
	refreshPriv = parsePriv(refreshPrivPEM)
	refreshPub = parsePub(refreshPubPEM)
}

// #################################################
// ############### loadECPrivateKey ################
// #################################################
func TestLoadECPrivateKey_ValidPEM(t *testing.T) {
	path := writePEM(t, accessPrivPEM)

	key, err := loadECPrivateKey(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key == nil {
		t.Fatal("expected a non-nil key")
	}
}

func TestLoadECPrivateKey_InvalidPEM(t *testing.T) {
	path := writePEM(t, "this is not a pem block")
	_, err := loadECPrivateKey(path)
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}

func TestLoadECPrivateKey_FileNotFound(t *testing.T) {
	_, err := loadECPrivateKey("/nonexistent/path/key.pem")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadECPrivateKey_WrongPEMType(t *testing.T) {
	// public key has "PUBLIC KEY" type, not "PRIVATE KEY"
	path := writePEM(t, accessPubPEM)
	_, err := loadECPrivateKey(path)
	if err == nil {
		t.Fatal("expected error when PEM block type is not PRIVATE KEY")
	}
}

// #################################################
// ############### loadECPublicKey #################
// #################################################
func TestLoadECPublicKey_ValidPEM(t *testing.T) {
	path := writePEM(t, accessPubPEM)
	key, err := loadECPublicKey(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key == nil {
		t.Fatal("expected a non-nil key")
	}
}

func TestLoadECPublicKey_InvalidPEM(t *testing.T) {
	path := writePEM(t, "very invalid pem block")
	_, err := loadECPublicKey(path)
	if err == nil {
		t.Fatal("expected error for invalid PEM content")
	}
}

func TestLoadECPublicKey_FileNotFound(t *testing.T) {
	_, err := loadECPublicKey("/nonexistent/path/key.pem")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadECPublicKey_WrongPEMType(t *testing.T) {
	// private key has "PRIVATE KEY" type, not "PUBLIC KEY"
	path := writePEM(t, accessPrivPEM)
	_, err := loadECPublicKey(path)
	if err == nil {
		t.Fatal("expected error when PEM block type is not PUBLIC KEY")
	}
}

// #################################################
// ########### decodeJWT (error paths) #############
// #################################################
func TestDecodeAccessToken_TamperedToken(t *testing.T) {
	initTestKeys(t)

	tokenStr, _ := NewAccessToken(uuid.New(), false, false)
	tampered := tokenStr + "x"

	_, err := DecodeAccessToken(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestDecodeAccessToken_WrongKey(t *testing.T) {
	initTestKeys(t)

	// sign with the refresh key then try to verify with the access public key
	userID := uuid.New()
	claims := &AccessToken{
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(refreshPriv)
	if err != nil {
		t.Fatalf("could not sign with refresh key: %v", err)
	}

	_, err = DecodeAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error when verifying with the wrong public key")
	}
}

func TestDecodeAccessToken_ExpiredToken(t *testing.T) {
	initTestKeys(t)

	userID := uuid.New()
	claims := &AccessToken{
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(accessPriv)
	if err != nil {
		t.Fatalf("could not create expired token: %v", err)
	}

	_, err = DecodeAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestDecodeAccessToken_RefreshTokenRejected(t *testing.T) {
	initTestKeys(t)

	// valid refresh token must not be accepted by DecodeAccessToken
	refreshStr, _ := NewRefreshToken(uuid.New())
	_, err := DecodeAccessToken(refreshStr)
	if err == nil {
		t.Fatal("expected access token decoder to reject a refresh token")
	}
}

func TestDecodeAccessToken_GarbageInput(t *testing.T) {
	initTestKeys(t)

	_, err := DecodeAccessToken("hehe.not.jwt")
	if err == nil {
		t.Fatal("expected error for garbage input")
	}
}

func TestDecodeRefreshToken_TamperedToken(t *testing.T) {
	initTestKeys(t)

	tokenStr, _ := NewRefreshToken(uuid.New())
	_, err := DecodeRefreshToken(tokenStr + "x")
	if err == nil {
		t.Fatal("expected error for tampered refresh token")
	}
}

func TestDecodeRefreshToken_AccessTokenRejected(t *testing.T) {
	initTestKeys(t)

	// valid access token must not be accepted by DecodeRefreshToken
	accessStr, _ := NewAccessToken(uuid.New(), false, false)
	_, err := DecodeRefreshToken(accessStr)
	if err == nil {
		t.Fatal("expected refresh token decoder to reject an access token")
	}
}

// #################################################
// ###### NewAccessToken & DecodeAccessToken #######
// #################################################
func TestNewAccessToken_ValidToken(t *testing.T) {
	initTestKeys(t)
	userID := uuid.New()

	tokenStr, err := NewAccessToken(userID, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected a non-empty token string")
	}
}

func TestNewAccessToken_DecodesCorrectly(t *testing.T) {
	initTestKeys(t)
	userID := uuid.New()

	tokenStr, _ := NewAccessToken(userID, true, true)
	claims, err := DecodeAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("subject: want %s, got %s", userID, claims.Subject)
	}
	if !claims.IsAdmin {
		t.Error("expected IsAdmin to be true")
	}
	if !claims.HasSubscription {
		t.Error("expected HasSubscription to be true")
	}
}

func TestNewAccessToken_IsAdminFalse(t *testing.T) {
	initTestKeys(t)
	userID := uuid.New()

	tokenStr, _ := NewAccessToken(userID, false, false)
	claims, err := DecodeAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if claims.IsAdmin {
		t.Error("expected IsAdmin to be false")
	}
}

func TestNewAccessToken_HasSubscriptionFalse(t *testing.T) {
	initTestKeys(t)
	userID := uuid.New()

	tokenStr, _ := NewAccessToken(userID, false, false)
	claims, err := DecodeAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if claims.IsAdmin {
		t.Error("expected HasSubscription to be false")
	}
}

func TestNewAccessToken_HasExpiry(t *testing.T) {
	initTestKeys(t)

	tokenStr, _ := NewAccessToken(uuid.New(), false, false)
	claims, _ := DecodeAccessToken(tokenStr)

	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if !claims.ExpiresAt.After(time.Now()) {
		t.Error("expected expiry to be in the future")
	}
}

// #################################################
// ##### NewRefreshToken & DecodeRefreshToken ######
// #################################################
func TestNewRefreshToken_ValidToken(t *testing.T) {
	initTestKeys(t)

	tokenStr, err := NewRefreshToken(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected a non-empty token string")
	}
}

func TestNewRefreshToken_DecodesCorrectly(t *testing.T) {
	initTestKeys(t)
	userID := uuid.New()

	tokenStr, _ := NewRefreshToken(userID)
	claims, err := DecodeRefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("subject: want %s, got %s", userID, claims.Subject)
	}
}

func TestNewRefreshToken_HasExpiry(t *testing.T) {
	initTestKeys(t)

	tokenStr, _ := NewRefreshToken(uuid.New())
	claims, _ := DecodeRefreshToken(tokenStr)

	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if !claims.ExpiresAt.After(time.Now()) {
		t.Error("expected expiry to be in the future")
	}
}
