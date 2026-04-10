package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"ricehub/internal/cache"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/grpc"
	"ricehub/internal/handlers"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

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
	validation.InitValidator()
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

	repository.Init(config.Config.Database.DatabaseUrl)
	defer repository.Close()

	// TODO: read gRPC url from config file
	grpc.Scanner.Init("localhost:40400")
	defer grpc.Scanner.Close() //nolint:errcheck

	go polar.StartSyncThread()
	go updateLeaderboard()

	// services
	adminService := services.NewAdminService()
	authService := services.NewAuthService()
	commentService := services.NewCommentService()
	leaderboardService := services.NewLeaderboardService()
	linkService := services.NewLinkService()
	profileService := services.NewProfileService()
	reportService := services.NewReportService()
	riceDotfilesService := services.NewRiceDotfilesService()
	riceScreenshotService := services.NewRiceScreenshotService()
	riceStarService := services.NewRiceStarService()
	riceTagService := services.NewRiceTagService()
	riceService := services.NewRiceService()
	tagService := services.NewTagService()
	userService := services.NewUserService()
	webVarService := services.NewWebVarService()

	// handlers
	adminHandler := handlers.NewAdminHandler(adminService)
	authHandler := handlers.NewAuthHandler(authService)
	commentHandler := handlers.NewCommentHandler(commentService)
	leaderboardHandler := handlers.NewLeaderboardHandler(leaderboardService)
	linkHandler := handlers.NewLinkHandler(linkService)
	profileHandler := handlers.NewProfileHandler(profileService)
	reportHandler := handlers.NewReportHandler(reportService)
	riceDotfilesHandler := handlers.NewRiceDotfilesHandler(riceDotfilesService)
	riceScreenshotHandler := handlers.NewRiceScreenshotHandler(riceScreenshotService)
	riceStarHandler := handlers.NewRiceStarHandler(riceStarService)
	riceTagHandler := handlers.NewRiceTagHandler(riceTagService)
	riceHandler := handlers.NewRiceHandler(riceService)
	tagHandler := handlers.NewTagHandler(tagService)
	userHandler := handlers.NewUserHandler(userService)
	webVarHandler := handlers.NewWebVarHandler(webVarService)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	corsConfig := cors.Config{
		AllowOrigins:     []string{config.Config.Server.CorsOrigin},
		AllowMethods:     []string{"GET", "POST", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
	}

	r.Use(
		gin.Recovery(),
		cors.New(corsConfig),
		security.LoggerMiddleware(logger),
		errs.ErrorHandler(logger),
		security.RateLimitMiddleware(100, time.Minute),
	)

	if err := r.SetTrustedProxies(nil); err != nil {
		return err
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found on this server!"})
	})
	r.Static("/public", "./public")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"maintenance": config.Config.App.Maintenance,
		})
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "I'm working and responding!"})
	})
	r.POST("/webhook", polar.WebhookListener)

	registerAuthRoutes(r, authHandler)
	registerUserRoutes(r, userHandler)
	registerRiceRoutes(
		r,
		riceHandler,
		riceDotfilesHandler,
		riceTagHandler,
		riceScreenshotHandler,
		riceStarHandler,
	)
	registerCommentRoutes(r, commentHandler)
	registerReportRoutes(r, reportHandler)
	registerTagRoutes(r, tagHandler)
	registerProfileRoutes(r, profileHandler)
	registerAdminRoutes(r, adminHandler)
	registerLinkRoutes(r, linkHandler)
	registerLeaderboardRoutes(r, leaderboardHandler)

	r.GET("/vars/:key", security.PathRateLimitMiddleware(5, time.Minute), webVarHandler.GetWebVarByKey)

	port := config.Config.Server.Port
	logger.Info(
		"API is now available",
		zap.Uint16("port", port),
	)
	return r.Run(fmt.Sprintf(":%v", port))
}

func updateLeaderboard() {
	update := func(tx pgx.Tx, period models.LeaderboardPeriod) error {
		err := repository.UpsertRiceLeaderboard(tx, period)
		if err != nil {
			return err
		}
		return repository.CleanupRiceLeaderboard(tx, period)
	}

	logger := zap.L()
	for {
		logger.Info("Updating rice leaderboard...")

		ctx := context.Background()
		tx, err := repository.StartTx(ctx)
		if err != nil {
			logger.Error("Failed to start tx", zap.Error(err))
			goto Skip
		}

		if err := update(tx, models.Week); err != nil {
			logger.Error("Failed to update weekly leaderboard", zap.Error(err))
			goto Skip
		}

		if err := update(tx, models.Month); err != nil {
			logger.Error("Failed to update monthly leaderboard", zap.Error(err))
			goto Skip
		}

		if err := update(tx, models.Year); err != nil {
			logger.Error("Failed to update yearly leaderboard", zap.Error(err))
			goto Skip
		}

		if err := tx.Commit(ctx); err != nil {
			logger.Error("Failed to commit tx", zap.Error(err))
		}
	Skip:
		time.Sleep(24 * time.Hour)
	}
}

func setupLogger() *zap.Logger {
	// json file logger with rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./logs/gin.json",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     7, // in days
	})
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// console logger
	// https://last9.io/blog/zap-logger/
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encodeCfg)
	consoleWriter := zapcore.AddSync(os.Stdout)

	// levels
	level := zap.InfoLevel
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWriter, level),
		zapcore.NewCore(fileEncoder, fileWriter, level),
	)

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	return logger
}

func registerAuthRoutes(r *gin.Engine, h *handlers.AuthHandler) {
	auth := r.Group("/auth")

	auth.POST("/register", security.MaintenanceMiddleware(), h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", security.PathRateLimitMiddleware(20, time.Minute), h.RefreshToken)
	auth.POST("/logout", h.LogOut)
}

func registerUserRoutes(r *gin.Engine, h *handlers.UserHandler) {
	maintenance := security.MaintenanceMiddleware()
	accountRL := security.PathRateLimitMiddleware(10, 24*time.Hour)

	users := r.Group("/users")

	// Public
	users.GET("", h.ListUsers)
	users.GET("/:id/rices", security.PathRateLimitMiddleware(5, time.Minute), h.ListUserRices)
	users.GET("/:id/rices/:slug", security.PathRateLimitMiddleware(30, time.Minute), h.GetUserRiceBySlug)

	// Authenticated
	auth := users.Group("", security.AuthMiddleware)
	auth.GET("/:id", security.PathRateLimitMiddleware(5, time.Minute), h.GetUserByID)
	auth.GET("/:id/purchased", security.PathRateLimitMiddleware(20, time.Minute), h.ListPurchasedRices)
	auth.DELETE("/:id", maintenance, security.PathRateLimitMiddleware(5, time.Minute), h.DeleteUser)
	auth.PATCH("/:id/displayName", maintenance, accountRL, h.UpdateDisplayName)
	auth.PATCH("/:id/password", maintenance, accountRL, h.UpdatePassword)
	auth.POST("/:id/avatar",
		maintenance,
		security.FileSizeLimitMiddleware(config.Config.Limits.UserAvatarSizeLimit),
		accountRL,
		h.UpdateAvatar,
	)
	auth.DELETE("/:id/avatar", maintenance, accountRL, h.DeleteAvatar)

	// Admin
	admin := users.Group("", security.AdminMiddleware)
	admin.POST("/:id/ban", h.BanUser)
	admin.DELETE("/:id/ban", h.UnbanUser)
}

func registerRiceRoutes(
	r *gin.Engine,
	rh *handlers.RiceHandler,
	dfh *handlers.RiceDotfilesHandler,
	th *handlers.RiceTagHandler,
	sch *handlers.RiceScreenshotHandler,
	sth *handlers.RiceStarHandler,
) {
	maintenance := security.MaintenanceMiddleware()
	updateRL := security.PathRateLimitMiddleware(10, time.Hour)
	limits := config.Config.Limits

	rices := r.Group("/rices")

	// Public
	rices.GET("", rh.ListRices)
	rices.GET("/:id", rh.GetRiceByID)
	rices.GET("/:id/comments", rh.ListRiceComments)
	rices.GET("/:id/dotfiles",
		security.PathRateLimitMiddleware(3, time.Minute),
		dfh.DownloadDotfiles,
	)

	// Authenticated
	auth := rices.Group("", security.AuthMiddleware)
	auth.POST("",
		maintenance,
		security.FileSizeLimitMiddleware(limits.DotfilesSizeLimit+limits.MaxScreenshotsPerRice*limits.ScreenshotSizeLimit),
		security.PathRateLimitMiddleware(15, 24*time.Hour),
		rh.CreateRice,
	)
	auth.PATCH("/:id", maintenance, updateRL, rh.UpdateRiceMetadata)
	auth.POST("/:id/tags", maintenance, updateRL, th.AddRiceTags)
	auth.DELETE("/:id/tags", maintenance, updateRL, th.RemoveRiceTags)
	auth.POST("/:id/dotfiles",
		maintenance,
		security.PathRateLimitMiddleware(3, time.Hour),
		security.FileSizeLimitMiddleware(limits.DotfilesSizeLimit),
		dfh.UpdateDotfiles,
	)
	auth.PATCH("/:id/dotfiles/type", maintenance, updateRL, dfh.UpdateDotfilesType)
	auth.PATCH("/:id/dotfiles/price", maintenance, updateRL, dfh.UpdateDotfilesPrice)
	auth.POST("/:id/screenshots",
		maintenance,
		security.FileSizeLimitMiddleware(limits.ScreenshotSizeLimit),
		security.PathRateLimitMiddleware(25, time.Hour),
		sch.CreateScreenshot,
	)
	auth.POST("/:id/purchase", maintenance, security.PathRateLimitMiddleware(5, time.Hour), dfh.PurchaseDotfiles)
	auth.PATCH("/:id/state", maintenance, security.AdminMiddleware, rh.UpdateRiceState)
	auth.POST("/:id/star", maintenance, sth.CreateRiceStar)
	auth.DELETE("/:id/star", maintenance, sth.DeleteRiceStar)
	auth.DELETE("/:id/screenshots/:previewId", maintenance, sch.DeleteScreenshot)
	auth.DELETE("/:id", maintenance, rh.DeleteRice)
}

func registerCommentRoutes(r *gin.Engine, h *handlers.CommentHandler) {
	maintenance := security.MaintenanceMiddleware()

	comments := r.Group("/comments", security.AuthMiddleware)

	comments.GET("", security.AdminMiddleware, h.ListComments)
	comments.GET("/:id", security.PathRateLimitMiddleware(10, time.Minute), h.GetCommentByID)
	comments.POST("", maintenance, security.PathRateLimitMiddleware(10, time.Hour), h.CreateComment)
	comments.PATCH("/:id", maintenance, security.PathRateLimitMiddleware(10, time.Hour), h.UpdateComment)
	comments.DELETE("/:id", maintenance, h.DeleteComment)
}

func registerReportRoutes(r *gin.Engine, h *handlers.ReportHandler) {
	reports := r.Group("/reports", security.AuthMiddleware)

	reports.POST("", security.PathRateLimitMiddleware(10, 24*time.Hour), h.CreateReport)

	admin := reports.Group("", security.AdminMiddleware)
	admin.GET("", h.ListReports)
	admin.GET("/:id", h.GetReportByID)
	admin.POST("/:id/close", h.CloseReport)
}

func registerTagRoutes(r *gin.Engine, h *handlers.TagHandler) {
	tags := r.Group("/tags")

	tags.GET("", h.ListTags)

	admin := tags.Group("", security.AuthMiddleware, security.AdminMiddleware)
	admin.POST("", h.CreateTag)
	admin.PATCH("/:id", h.UpdateTag)
	admin.DELETE("/:id", h.DeleteTag)
}

func registerProfileRoutes(r *gin.Engine, h *handlers.ProfileHandler) {
	profiles := r.Group("/profiles")

	profiles.GET("/:username", h.GetProfileByUsername)
}

func registerAdminRoutes(r *gin.Engine, h *handlers.AdminHandler) {
	admin := r.Group("/admin", security.AuthMiddleware, security.AdminMiddleware)

	admin.GET("/stats", h.ServiceStatistics)
}

func registerLinkRoutes(r *gin.Engine, h *handlers.LinkHandler) {
	links := r.Group("/links")

	links.GET(
		"/subscription",
		security.PathRateLimitMiddleware(5, time.Minute),
		security.AuthMiddleware,
		h.GetSubscriptionLink,
	)
	links.GET(
		"/:name",
		security.PathRateLimitMiddleware(5, time.Minute),
		h.GetLinkByName,
	)
}

func registerLeaderboardRoutes(r *gin.Engine, h *handlers.LeaderboardHandler) {
	leaderboard := r.Group("/leaderboard")

	rl := security.PathRateLimitMiddleware(10, time.Minute)
	leaderboard.GET("/week", rl, h.GetWeeklyLeaderboard)
	leaderboard.GET("/month", rl, h.GetMonthlyLeaderboard)
	leaderboard.GET("/year", rl, h.GetYearlyLeaderboard)
}
