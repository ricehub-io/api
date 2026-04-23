package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"ricehub/internal/app"
	"ricehub/internal/cache"
	"ricehub/internal/config"
	"ricehub/internal/grpc"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"strings"
	"testing"
	"time"

	pb "ricehub/proto"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const accessPrivPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgGSpGIiazFUSDONm0
XupELbMQbBnSmgocwcx+o0uTIWihRANCAATMogo6VBDQanJ+X2ZZjbn1V1+UN3re
WRYdG2kLyfjxERaKQJhiuPBUCN+itdyjXbrZNDC+Jf4SQa8fpdxy2X3P
-----END PRIVATE KEY-----
`
const accessPubPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEzKIKOlQQ0Gpyfl9mWY259VdflDd6
3lkWHRtpC8n48REWikCYYrjwVAjforXco1262TQwviX+EkGvH6Xcctl9zw==
-----END PUBLIC KEY-----
`
const refreshPrivPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgIeVCwhYdKkOuPW5M
HubmLnEL+HX90x/TkPUvyV2vvMqhRANCAASzMGtrh1E7zxxGsrbUWDjKAPIfUSBy
/U1Gjm1CaFza4QBuy8qR3h08njRT/IFSB4PH6SP0qbAWJhZxvv4sWK0s
-----END PRIVATE KEY-----
`
const refreshPubPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEszBra4dRO88cRrK21Fg4ygDyH1Eg
cv1NRo5tQmhc2uEAbsvKkd4dPJ40U/yBUgeDx+kj9KmwFiYWcb7+LFitLA==
-----END PUBLIC KEY-----
`

var randLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	dir, err := os.MkdirTemp("", "ricehub-test-keys-*")
	if err != nil {
		panic("testutil: create temp dir for JWT keys: " + err.Error())
	}

	writePEM := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			panic("testutil: write PEM " + name + ": " + err.Error())
		}
	}
	writePEM("access_private.pem", accessPrivPEM)
	writePEM("access_public.pem", accessPubPEM)
	writePEM("refresh_private.pem", refreshPrivPEM)
	writePEM("refresh_public.pem", refreshPubPEM)

	config.Config.Server.KeysDir = dir
	security.InitJWT(dir)

	config.Config.App.DisableRateLimits = true
	config.Config.App.PaginationLimit = 20
	config.Config.Limits.MaxScreenshotsPerRice = 10
	config.Config.Limits.UserAvatarSizeLimit = 5_000_000
	config.Config.Limits.DotfilesSizeLimit = 500_000_000
	config.Config.Limits.ScreenshotSizeLimit = 10_000_000
	config.Config.JWT.AccessExpiration = 5 * time.Minute
	config.Config.JWT.RefreshExpiration = 24 * time.Hour
	config.Config.App.CDNUrl = "http://localhost:3000"
	config.Config.App.DefaultAvatar = "/avatars/default.png"

	MockCleanScanner()
}

// SetupTestDB starts a PostgreSQL testcontainer, applies schema.sql, and
// returns a live pool. Registers t.Cleanup to terminate the container.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, stop := MustStartPostgres("../../schema.sql")
	t.Cleanup(func() { _ = stop() })
	t.Cleanup(pool.Close)
	return pool
}

// SetupTestRedis starts a miniredis instance in-process, initializes the
// cache package, and registers cleanup.
func SetupTestRedis(t *testing.T) {
	t.Helper()

	stop := MustStartRedis()
	t.Cleanup(stop)
}

// MustStartPostgres starts a Postgres testcontainer and applies the schema at
// schemaPath.
func MustStartPostgres(schemaPath string) (*pgxpool.Pool, func() error) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("ricehub_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.WithInitScripts(schemaPath),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic("testutil: start postgres container: " + err.Error())
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("testutil: get postgres connection string: " + err.Error())
	}

	pool := repository.NewPool(connStr)
	stop := func() error {
		pool.Close()
		return pgContainer.Terminate(ctx)
	}
	return pool, stop
}

// MustStartRedis starts a miniredis instance and initialises the cache package.
func MustStartRedis() func() {
	mr, err := miniredis.Run()
	if err != nil {
		panic("testutil: start miniredis: " + err.Error())
	}
	cache.InitCache("redis://" + mr.Addr())
	return func() {
		cache.CloseCache()
		mr.Close()
	}
}

// SetupTestApp creates and returns a Gin engine wired with all real handlers
// against the given DB pool.
func SetupTestApp(pool *pgxpool.Pool) *gin.Engine {
	return app.New(pool, zap.NewNop())
}

// MakeAccessToken returns a "Bearer <token>" string signed with the test key.
func MakeAccessToken(t *testing.T, userID uuid.UUID, isAdmin bool) string {
	t.Helper()

	tok, err := security.NewAccessToken(userID, isAdmin, false)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return "Bearer " + tok
}

// MakeRefreshToken returns a signed refresh token string.
func MakeRefreshToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	tok, err := security.NewRefreshToken(userID)
	if err != nil {
		t.Fatalf("sign refresh token: %v", err)
	}
	return tok
}

type cleanScanner struct{}

func (cleanScanner) ScanFile(_ string) (*pb.ScanResult, error) {
	return &pb.ScanResult{IsMalicious: false}, nil
}

func MockCleanScanner() {
	grpc.Scanner = cleanScanner{}
}

// DoRequest fires an HTTP request against the Gin engine and returns the recorded response.
func DoRequest(
	engine *gin.Engine,
	method, path, body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// DoRawRequest fires an HTTP request with an arbitrary body and content type.
func DoRawRequest(
	engine *gin.Engine,
	method, path string,
	body io.Reader,
	contentType string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// AuthHeader returns a header map with the Authorization key set.
func AuthHeader(token string) map[string]string {
	return map[string]string{"Authorization": token}
}

// TinyPNG returns the bytes of a minimal 1x1 PNG.
func TinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode tiny PNG: %v", err)
	}
	return buf.Bytes()
}

// TODO: use it in all integration tests for username and other stuff
// RandString generates a random alphabetic string of fixed length.
func RandString(l int64) string {
	if l <= 0 {
		panic("random string length must be greater than zero!")
	}

	b := make([]rune, l)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(l))
		b[i] = randLetters[idx.Uint64()]
	}
	return string(b)
}
