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

	auth.POST("/register", security.MaintenanceMiddleware(), handlers.Register)
	auth.POST("/login", handlers.Login)
	auth.POST("/refresh", security.PathRateLimitMiddleware(20, time.Minute), handlers.RefreshToken)
	auth.POST("/logout", handlers.LogOut)
}

func registerUserRoutes(r *gin.Engine) {
	maintenance := security.MaintenanceMiddleware()
	accountRL := security.PathRateLimitMiddleware(10, 24*time.Hour)

	users := r.Group("/users")

	// Public
	users.GET("", handlers.FetchUsers)
	users.GET("/:id/rices", security.PathRateLimitMiddleware(5, time.Minute), handlers.FetchUserRices)
	users.GET("/:id/rices/:slug", security.PathRateLimitMiddleware(30, time.Minute), handlers.GetUserRiceBySlug)

	// Authenticated
	auth := users.Group("", security.AuthMiddleware)
	auth.GET("/:id", security.PathRateLimitMiddleware(5, time.Minute), handlers.GetUserById)
	auth.GET("/:id/purchased", security.PathRateLimitMiddleware(20, time.Minute), handlers.FetchPurchasedRices)
	auth.DELETE("/:id", maintenance, security.PathRateLimitMiddleware(5, time.Minute), handlers.DeleteUser)
	auth.PATCH("/:id/displayName", maintenance, accountRL, handlers.UpdateDisplayName)
	auth.PATCH("/:id/password", maintenance, accountRL, handlers.UpdatePassword)
	auth.POST("/:id/avatar", maintenance, security.FileSizeLimitMiddleware(utils.Config.Limits.UserAvatarSizeLimit), accountRL, handlers.UploadAvatar)
	auth.DELETE("/:id/avatar", maintenance, accountRL, handlers.DeleteAvatar)

	// Admin
	admin := users.Group("", security.AdminMiddleware)
	admin.POST("/:id/ban", handlers.BanUser)
	admin.DELETE("/:id/ban", handlers.UnbanUser)
}

func registerRiceRoutes(r *gin.Engine) {
	maintenance := security.MaintenanceMiddleware()
	updateRL := security.PathRateLimitMiddleware(10, time.Hour)
	limits := utils.Config.Limits

	rices := r.Group("/rices")

	// Public
	rices.GET("", handlers.FetchRices)
	rices.GET("/:id", handlers.GetRiceById)
	rices.GET("/:id/comments", handlers.GetRiceComments)
	rices.GET("/:id/dotfiles", handlers.DownloadDotfiles)

	// Authenticated
	auth := rices.Group("", security.AuthMiddleware)
	auth.POST("",
		maintenance,
		security.FileSizeLimitMiddleware(limits.DotfilesSizeLimit+int64(limits.MaxPreviewsPerRice)*limits.PreviewSizeLimit),
		security.PathRateLimitMiddleware(15, 24*time.Hour),
		handlers.CreateRice,
	)
	auth.PATCH("/:id", maintenance, updateRL, handlers.UpdateRiceMetadata)
	auth.POST("/:id/dotfiles",
		maintenance,
		security.PathRateLimitMiddleware(3, time.Hour),
		security.FileSizeLimitMiddleware(limits.DotfilesSizeLimit),
		handlers.UpdateDotfiles,
	)
	auth.PATCH("/:id/dotfiles/type", maintenance, updateRL, handlers.UpdateDotfilesType)
	auth.PATCH("/:id/dotfiles/price", maintenance, updateRL, handlers.UpdateDotfilesPrice)
	auth.POST("/:id/screenshots",
		maintenance,
		security.FileSizeLimitMiddleware(limits.PreviewSizeLimit),
		security.PathRateLimitMiddleware(25, time.Hour),
		handlers.AddScreenshot,
	)
	auth.POST("/:id/purchase", maintenance, security.PathRateLimitMiddleware(5, time.Hour), handlers.PurchaseDotfiles)
	auth.PATCH("/:id/state", maintenance, security.AdminMiddleware, handlers.UpdateRiceState)
	auth.POST("/:id/star", maintenance, handlers.AddRiceStar)
	auth.DELETE("/:id/star", maintenance, handlers.DeleteRiceStar)
	auth.DELETE("/:id/screenshots/:previewId", maintenance, handlers.DeleteScreenshot)
	auth.DELETE("/:id", maintenance, handlers.DeleteRice)
}

func registerCommentRoutes(r *gin.Engine) {
	maintenance := security.MaintenanceMiddleware()

	comments := r.Group("/comments", security.AuthMiddleware)

	comments.GET("", security.AdminMiddleware, handlers.GetRecentComments)
	comments.GET("/:id", security.PathRateLimitMiddleware(10, time.Minute), handlers.GetCommentById)
	comments.POST("", maintenance, security.PathRateLimitMiddleware(10, time.Hour), handlers.AddComment)
	comments.PATCH("/:id", maintenance, security.PathRateLimitMiddleware(10, time.Hour), handlers.UpdateComment)
	comments.DELETE("/:id", maintenance, handlers.DeleteComment)
}

func registerReportRoutes(r *gin.Engine) {
	reports := r.Group("/reports", security.AuthMiddleware)

	reports.POST("", security.PathRateLimitMiddleware(10, 24*time.Hour), handlers.CreateReport)

	admin := reports.Group("", security.AdminMiddleware)
	admin.GET("", handlers.FetchReports)
	admin.GET("/:reportId", handlers.GetReportById)
	admin.POST("/:reportId/close", handlers.CloseReport)
}

func registerTagRoutes(r *gin.Engine) {
	tags := r.Group("/tags")

	tags.GET("", handlers.GetAllTags)

	admin := tags.Group("", security.AuthMiddleware, security.AdminMiddleware)
	admin.POST("", handlers.CreateTag)
	admin.PATCH("/:id", handlers.UpdateTag)
	admin.DELETE("/:id", handlers.DeleteTag)
}

func registerProfileRoutes(r *gin.Engine) {
	profiles := r.Group("/profiles")

	profiles.GET("/:username", handlers.GetUserProfile)
}

func registerAdminRoutes(r *gin.Engine) {
	admin := r.Group("/admin", security.AuthMiddleware, security.AdminMiddleware)

	admin.GET("/stats", handlers.ServiceStatistics)
}
