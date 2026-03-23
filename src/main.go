package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"ricehub/src/errs"
	"ricehub/src/handlers"
	"ricehub/src/polar"
	"ricehub/src/repository"
	"ricehub/src/security"
	"ricehub/src/utils"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	defer logger.Sync()

	utils.InitConfig(configPath)
	utils.InitValidator()
	polar.Init(utils.Config.Polar.Token, utils.Config.Polar.Sandbox)
	security.InitJWT(utils.Config.Server.KeysDir)

	if utils.Config.App.DisableRateLimits {
		logger.Warn("Rate limits disabled! Is it intentional?")
	}
	if utils.Config.App.Maintenance {
		logger.Warn("Maintenance mode toggled! Is it intentional?")
	}

	utils.InitCache(utils.Config.Database.RedisUrl)
	defer utils.CloseCache()

	repository.Init(utils.Config.Database.DatabaseUrl)
	defer repository.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	corsConfig := cors.Config{
		AllowOrigins:     []string{utils.Config.Server.CorsOrigin},
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

	setupRoutes(r)

	port := utils.Config.Server.Port
	logger.Info(
		"API is now available",
		zap.Uint16("port", port),
	)
	return r.Run(fmt.Sprintf(":%v", port))
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

func setupRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found on this server!"})
	})
	r.Static("/public", "./public")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"maintenance": utils.Config.App.Maintenance,
		})
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "I'm working and responding!"})
	})
	r.POST("/webhook", polar.WebhookListener)

	registerAuthRoutes(r)
	registerUserRoutes(r)
	registerRiceRoutes(r)
	registerCommentRoutes(r)
	registerReportRoutes(r)
	registerTagRoutes(r)
	registerProfileRoutes(r)
	registerAdminRoutes(r)

	// unaimeds: I dont think we'll ever expand this domain of endpoints
	// therefore they're not in their own 'register*Routes' function
	r.GET("/vars/:key", security.PathRateLimitMiddleware(5, 1*time.Minute), handlers.GetWebsiteVariable)
	r.GET("/links/:name", security.PathRateLimitMiddleware(5, 1*time.Minute), handlers.GetLinkByName)
}

func registerAuthRoutes(r *gin.Engine) {
	auth := r.Group("/auth")

	auth.POST(
		"/register",
		security.MaintenanceMiddleware(),
		handlers.Register,
	)
	auth.POST("/login", handlers.Login)
	auth.POST(
		"/refresh",
		security.PathRateLimitMiddleware(100, 1*time.Minute),
		handlers.RefreshToken,
	)
	auth.POST("/logout", handlers.LogOut)
}

func registerUserRoutes(r *gin.Engine) {
	users := r.Group("/users")

	defaultRL := security.PathRateLimitMiddleware(5, 1*time.Minute)
	users.GET("", handlers.FetchUsers)
	users.GET(
		"/:id/rices",
		security.PathRateLimitMiddleware(5, 1*time.Minute),
		handlers.FetchUserRices,
	)
	users.GET(
		"/:id/purchased",
		security.AuthMiddleware,
		security.PathRateLimitMiddleware(20, 1*time.Minute),
		handlers.FetchPurchasedRices,
	)
	users.GET(
		"/:id/rices/:slug",
		security.PathRateLimitMiddleware(30, 1*time.Minute),
		handlers.GetUserRiceBySlug,
	)

	authedOnly := users.Use(security.AuthMiddleware)
	authedOnly.GET("/:id", defaultRL, handlers.GetUserById)
	authedOnly.DELETE(
		"/:id",
		security.MaintenanceMiddleware(),
		defaultRL,
		handlers.DeleteUser,
	)
	authedOnly.PATCH("/:id/displayName", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.UpdateDisplayName)
	authedOnly.PATCH("/:id/password", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.UpdatePassword)
	authedOnly.POST("/:id/avatar", security.MaintenanceMiddleware(), security.FileSizeLimitMiddleware(utils.Config.Limits.UserAvatarSizeLimit), security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.UploadAvatar)
	authedOnly.DELETE("/:id/avatar", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.DeleteAvatar)

	adminOnly := users.Use(security.AdminMiddleware)
	adminOnly.POST("/:id/ban", handlers.BanUser)
	adminOnly.DELETE("/:id/ban", handlers.UnbanUser)
}

func registerRiceRoutes(r *gin.Engine) {
	updateResourceMiddleware := []gin.HandlerFunc{
		security.MaintenanceMiddleware(),
		security.PathRateLimitMiddleware(10, time.Hour),
	}

	rices := r.Group("/rices")

	rices.GET("", handlers.FetchRices)
	rices.GET("/:id", handlers.GetRiceById)
	rices.GET("/:id/comments", handlers.GetRiceComments)
	rices.GET("/:id/dotfiles", handlers.DownloadDotfiles)

	auth := rices.Use(security.AuthMiddleware)
	createRiceMiddleware := []gin.HandlerFunc{
		security.MaintenanceMiddleware(),
		security.FileSizeLimitMiddleware(utils.Config.Limits.DotfilesSizeLimit + int64(utils.Config.Limits.MaxPreviewsPerRice)*utils.Config.Limits.PreviewSizeLimit),
		security.PathRateLimitMiddleware(15, 24*time.Hour),
	}
	auth.POST("", append(createRiceMiddleware, handlers.CreateRice)...)
	auth.PATCH(
		"/:id",
		append(updateResourceMiddleware, handlers.UpdateRiceMetadata)...,
	)
	updateDotfilesMiddleware := []gin.HandlerFunc{
		security.MaintenanceMiddleware(),
		security.PathRateLimitMiddleware(3, time.Hour),
		security.FileSizeLimitMiddleware(utils.Config.Limits.DotfilesSizeLimit),
	}
	auth.POST(
		"/:id/dotfiles",
		append(updateDotfilesMiddleware, handlers.UpdateDotfiles)...,
	)
	auth.PATCH(
		"/:id/dotfiles/type",
		append(updateResourceMiddleware, handlers.UpdateDotfilesType)...,
	)
	auth.PATCH(
		"/:id/dotfiles/price",
		append(updateResourceMiddleware, handlers.UpdateDotfilesPrice)...,
	)
	addScreenshotMiddleware := []gin.HandlerFunc{
		security.MaintenanceMiddleware(),
		security.FileSizeLimitMiddleware(utils.Config.Limits.PreviewSizeLimit),
		security.PathRateLimitMiddleware(25, time.Hour),
	}
	auth.POST(
		"/:id/screenshots",
		append(addScreenshotMiddleware, handlers.AddScreenshot)...,
	)
	auth.POST(
		"/:id/purchase",
		security.MaintenanceMiddleware(),
		security.PathRateLimitMiddleware(5, time.Hour),
		handlers.PurchaseDotfiles,
	)
	auth.PATCH("/:id/state", security.MaintenanceMiddleware(), security.AdminMiddleware, handlers.UpdateRiceState)
	auth.POST("/:id/star", security.MaintenanceMiddleware(), handlers.AddRiceStar)
	auth.DELETE("/:id/star", security.MaintenanceMiddleware(), handlers.DeleteRiceStar)
	auth.DELETE("/:id/screenshots/:previewId", security.MaintenanceMiddleware(), handlers.DeleteScreenshot)
	auth.DELETE("/:id", security.MaintenanceMiddleware(), handlers.DeleteRice)
}

func registerCommentRoutes(r *gin.Engine) {
	comments := r.Group("/comments").Use(security.AuthMiddleware)

	comments.GET("", security.AdminMiddleware, handlers.GetRecentComments)

	comments.POST("", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, time.Hour), handlers.AddComment)
	comments.GET("/:id", security.PathRateLimitMiddleware(10, time.Minute), handlers.GetCommentById)
	comments.PATCH("/:id", security.MaintenanceMiddleware(), security.PathRateLimitMiddleware(10, time.Hour), handlers.UpdateComment)
	comments.DELETE("/:id", security.MaintenanceMiddleware(), handlers.DeleteComment)
}

func registerReportRoutes(r *gin.Engine) {
	reports := r.Group("/reports").Use(security.AuthMiddleware)

	reports.POST(
		"",
		security.PathRateLimitMiddleware(50, 24*time.Hour),
		handlers.CreateReport,
	)

	adminOnly := reports.Use(security.AdminMiddleware)
	adminOnly.GET("", handlers.FetchReports)
	adminOnly.GET("/:reportId", handlers.GetReportById)
	adminOnly.POST("/:reportId/close", handlers.CloseReport)
}

func registerTagRoutes(r *gin.Engine) {
	tags := r.Group("/tags")

	tags.GET("", handlers.GetAllTags)

	adminOnly := tags.Use(security.AuthMiddleware, security.AdminMiddleware)
	adminOnly.POST("", handlers.CreateTag)
	adminOnly.PATCH("/:id", handlers.UpdateTag)
	adminOnly.DELETE("/:id", handlers.DeleteTag)
}

func registerProfileRoutes(r *gin.Engine) {
	profiles := r.Group("/profiles")

	profiles.GET("/:username", handlers.GetUserProfile)
}

func registerAdminRoutes(r *gin.Engine) {
	admin := r.Group("/admin").Use(
		security.AuthMiddleware,
		security.AdminMiddleware,
	)

	admin.GET("/stats", handlers.ServiceStatistics)
}
