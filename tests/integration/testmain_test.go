package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ricehub-io/api/internal/testutil"

	"github.com/gin-gonic/gin"
)

var testApp *gin.Engine

func TestMain(m *testing.M) {
	migrationDir, err := filepath.Abs("../../migrations")
	if err != nil {
		panic(err)
	}

	tmpDir, err := os.MkdirTemp("", "ricehub-inttest-*")
	if err != nil {
		panic(err)
	}
	for _, sub := range []string{"public/dotfiles", "public/screenshots"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, sub), 0755); err != nil {
			panic(err)
		}
	}
	if err := os.Chdir(tmpDir); err != nil {
		panic(err)
	}

	pool, stopDB := testutil.MustStartPostgres(migrationDir)
	stopRedis := testutil.MustStartRedis()

	testApp = testutil.SetupTestApp(pool)

	code := m.Run()

	stopRedis()
	_ = stopDB()
	_ = os.RemoveAll(tmpDir)

	os.Exit(code)
}
