package security

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"ricehub/src/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AccessToken struct {
	IsAdmin bool `json:"isAdmin"`
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
	root, err := os.OpenRoot(utils.Config.Server.KeysDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing EC private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key.(*ecdsa.PrivateKey), nil
}

func loadECPublicKey(fileName string) (*ecdsa.PublicKey, error) {
	root, err := os.OpenRoot(utils.Config.Server.KeysDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing EC public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pub.(*ecdsa.PublicKey), nil
}

func InitJWT(keysDir string) {
	logger := zap.L()
	logger.Info("Parsing JWT key pairs...", zap.String("dir", keysDir))

	priv := func(fileName string) *ecdsa.PrivateKey {
		key, err := loadECPrivateKey(fileName)
		if err != nil {
			log.Fatalf("Failed to load JWT private key: %v\n", err)
		}
		return key
	}
	pub := func(fileName string) *ecdsa.PublicKey {
		key, err := loadECPublicKey(fileName)
		if err != nil {
			log.Fatalf("Failed to load JWT public key: %v\n", err)
		}
		return key
	}

	accessPriv = priv("access_private.pem")
	accessPub = pub("access_public.pem")
	refreshPriv = priv("refresh_private.pem")
	refreshPub = pub("refresh_public.pem")

	logger.Info("JWT key-pairs successfully loaded")
}

func NewAccessToken(userID uuid.UUID, isAdmin bool) (token string, err error) {
	exp := time.Now().Add(utils.Config.JWT.AccessExpiration)
	claims := AccessToken{
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(accessPriv)
	return
}

func NewRefreshToken(userID uuid.UUID) (token string, err error) {
	exp := time.Now().Add(utils.Config.JWT.RefreshExpiration)
	claims := RefreshToken{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(refreshPriv)
	return
}

func decodeJWT[T jwt.Claims](tokenStr string, newClaims func() T, pubKey *ecdsa.PublicKey) (T, error) {
	claims := newClaims()

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		var null T
		return null, err
	}

	if claims, ok := token.Claims.(T); ok && token.Valid {
		return claims, nil
	}

	var null T
	return null, fmt.Errorf("could not parse and decode jwt")
}

func DecodeAccessToken(tokenStr string) (token *AccessToken, err error) {
	token, err = decodeJWT(tokenStr, func() *AccessToken { return &AccessToken{} }, accessPub)
	return
}

func DecodeRefreshToken(tokenStr string) (token *RefreshToken, err error) {
	token, err = decodeJWT(tokenStr, func() *RefreshToken { return &RefreshToken{} }, refreshPub)
	return
}
