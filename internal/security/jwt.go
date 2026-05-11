package security

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ricehub-io/api/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessToken struct {
	IsAdmin         bool `json:"isAdmin"`
	HasSubscription bool `json:"hasSubscription"`
	jwt.RegisteredClaims
}

type RefreshToken struct {
	jwt.RegisteredClaims
}

var (
	accessPriv  *ecdsa.PrivateKey
	accessPub   *ecdsa.PublicKey
	refreshPriv *ecdsa.PrivateKey
	refreshPub  *ecdsa.PublicKey
)

func loadECPrivateKey(fileName string) (*ecdsa.PrivateKey, error) {
	filePath := filepath.Join(config.Config.Server.KeysDir, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("os read file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing EC private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509 parse private key: %w", err)
	}

	return key.(*ecdsa.PrivateKey), nil
}

func loadECPublicKey(fileName string) (*ecdsa.PublicKey, error) {
	filePath := filepath.Join(config.Config.Server.KeysDir, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("os read file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing EC public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509 parse public key: %w", err)
	}

	return pub.(*ecdsa.PublicKey), nil
}

func InitJWT(keysDir string) error {
	priv := func(fileName string) (*ecdsa.PrivateKey, error) {
		key, err := loadECPrivateKey(fileName)
		if err != nil {
			return nil, fmt.Errorf("load ec private key: %w", err)
		}
		return key, nil
	}
	pub := func(fileName string) (*ecdsa.PublicKey, error) {
		key, err := loadECPublicKey(fileName)
		if err != nil {
			return nil, fmt.Errorf("load ec public key: %w", err)
		}
		return key, nil
	}

	var err error

	if accessPriv, err = priv("access_private.pem"); err != nil {
		return fmt.Errorf("access private: %w", err)
	}
	if accessPub, err = pub("access_public.pem"); err != nil {
		return fmt.Errorf("access public: %w", err)
	}

	if refreshPriv, err = priv("refresh_private.pem"); err != nil {
		return fmt.Errorf("refresh private: %w", err)
	}
	if refreshPub, err = pub("refresh_public.pem"); err != nil {
		return fmt.Errorf("refresh public: %w", err)
	}

	return nil
}

func NewAccessToken(userID uuid.UUID, isAdmin, hasSubscription bool) (string, error) {
	exp := time.Now().Add(config.Config.JWT.AccessExpiration)
	claims := AccessToken{
		IsAdmin:         isAdmin,
		HasSubscription: hasSubscription,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(accessPriv)
}

func NewRefreshToken(userID uuid.UUID) (string, error) {
	exp := time.Now().Add(config.Config.JWT.RefreshExpiration)
	claims := RefreshToken{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(refreshPriv)
}

func decodeJWT[T jwt.Claims](
	tokenStr string,
	newClaims func() T,
	pubKey *ecdsa.PublicKey,
) (T, error) {
	var null T
	claims := newClaims()

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return null, fmt.Errorf("jwt parse with claims: %w", err)
	}

	if claims, ok := token.Claims.(T); ok && token.Valid {
		return claims, nil
	}

	return null, errors.New("could not parse and decode jwt")
}

func DecodeAccessToken(tokenStr string) (token *AccessToken, err error) {
	token, err = decodeJWT(tokenStr, func() *AccessToken { return &AccessToken{} }, accessPub)
	return
}

func DecodeRefreshToken(tokenStr string) (token *RefreshToken, err error) {
	token, err = decodeJWT(tokenStr, func() *RefreshToken { return &RefreshToken{} }, refreshPub)
	return
}
