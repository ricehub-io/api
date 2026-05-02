package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"ricehub/internal/app"
	"ricehub/internal/cache"
	"ricehub/internal/config"
	"ricehub/internal/grpc"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const logLevel = zap.InfoLevel
const configPath = "config.toml"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	logger := setupLogger()
	defer logger.Sync() //nolint:errcheck

	config.InitConfig(configPath)
	// validation.InitValidator()
	polar.Init(config.Config.Polar.Token, config.Config.Polar.Sandbox)
	security.InitJWT(config.Config.Server.KeysDir)

	if config.Config.App.DisableRateLimits {
		logger.Warn("Rate limits disabled! Is it intentional?")
	}
	if config.Config.App.Maintenance {
		logger.Warn("Maintenance mode toggled! Is it intentional?")
	}

	cache.InitCache(config.Config.Database.RedisUrl)
	defer cache.CloseCache()

	dbPool := repository.NewPool(config.Config.Database.DatabaseUrl)
	defer dbPool.Close()

	// TODO: read gRPC url from config file
	grpc.InitScanner("localhost:40400")
	defer grpc.CloseScanner() //nolint:errcheck

	dotfilesPurchaseRepo := repository.NewDotfilesPurchaseRepository(dbPool)
	riceDotfilesRepo := repository.NewRiceDotfilesRepository(dbPool)
	riceLeaderboardRepo := repository.NewRiceLeaderboardRepository(dbPool)
	userSubscriptionRepo := repository.NewUserSubscriptionRepository(dbPool)

	go polar.StartSyncThread(dbPool, riceDotfilesRepo, dotfilesPurchaseRepo, userSubscriptionRepo)
	go startLeaderboardSync(dbPool, riceLeaderboardRepo)

	r := app.New(dbPool, logger)

	port := config.Config.Server.Port
	logger.Info(
		"API is now available",
		zap.Uint16("port", port),
	)
	return r.Run(fmt.Sprintf(":%v", port))
}

func startLeaderboardSync(dbPool *pgxpool.Pool, leaderboard *repository.RiceLeaderboardRepository) {
	for {
		zap.L().Info("Updating rice leaderboard...")
		updateLeaderboard(dbPool, leaderboard)
		time.Sleep(24 * time.Hour)
	}
}

func updateLeaderboard(dbPool *pgxpool.Pool, leaderboard *repository.RiceLeaderboardRepository) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	l := zap.L()

	update := func(tx *repository.RiceLeaderboardRepository, period models.LeaderboardPeriod) error {
		err := tx.UpsertRiceLeaderboard(ctx, period)
		if err != nil {
			return err
		}
		return tx.CleanupRiceLeaderboard(ctx, period)
	}

	tx, err := dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		l.Error("Failed to start tx", zap.Error(err))
		return
	}
	txRepo := leaderboard.WithTx(tx)

	if err := update(txRepo, models.Week); err != nil {
		l.Error("Failed to update weekly leaderboard", zap.Error(err))
		return
	}

	if err := update(txRepo, models.Month); err != nil {
		l.Error("Failed to update monthly leaderboard", zap.Error(err))
		return
	}

	if err := update(txRepo, models.Year); err != nil {
		l.Error("Failed to update yearly leaderboard", zap.Error(err))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		l.Error("Failed to commit tx", zap.Error(err))
	}
}

func setupLogger() *zap.Logger {
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encodeCfg)
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel)

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)

	return logger
}
