package app

import (
	"net/http"
	"ricehub/internal/config"
	"ricehub/internal/errs"
	"ricehub/internal/handlers"
	"ricehub/internal/polar"
	"ricehub/internal/repository"
	"ricehub/internal/security"
	"ricehub/internal/services"
	"ricehub/internal/validation"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func New(pool *pgxpool.Pool, logger *zap.Logger) *gin.Engine {
	validation.InitValidator()

	// repositories
	adminRepo := repository.NewAdminRepository(pool)
	commentRepo := repository.NewCommentRepository(pool)
	linkRepo := repository.NewLinkRepository(pool)
	dotfilesPurchaseRepo := repository.NewDotfilesPurchaseRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	riceDotfilesRepo := repository.NewRiceDotfilesRepository(pool)
	riceLeaderboardRepo := repository.NewRiceLeaderboardRepository(pool)
	riceTagRepo := repository.NewRiceTagRepository(pool)
	riceRepo := repository.NewRiceRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	userBanRepo := repository.NewUserBanRepository(pool)
	userSubscriptionRepo := repository.NewUserSubscriptionRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	webhookEventRepo := repository.NewWebhookEventRepository(pool)
	webVarRepo := repository.NewWebVarRepository(pool)

	webhookListener := polar.NewWebhookListener(webhookEventRepo, userSubscriptionRepo, riceDotfilesRepo, dotfilesPurchaseRepo)

	// services
	adminService := services.NewAdminService(adminRepo)
	authService := services.NewAuthService(userRepo, userBanRepo, userSubscriptionRepo)
	commentService := services.NewCommentService(commentRepo, userRepo, userBanRepo)
	leaderboardService := services.NewLeaderboardService(riceLeaderboardRepo)
	linkService := services.NewLinkService(linkRepo, userRepo, userSubscriptionRepo, userBanRepo)
	profileService := services.NewProfileService(userRepo, riceRepo)
	reportService := services.NewReportService(reportRepo)
	riceDotfilesService := services.NewRiceDotfilesService(riceRepo, riceDotfilesRepo, userRepo, userBanRepo)
	riceScreenshotService := services.NewRiceScreenshotService(pool, riceRepo, userRepo, userBanRepo)
	riceStarService := services.NewRiceStarService(riceRepo)
	riceTagService := services.NewRiceTagService(riceRepo, riceTagRepo, userRepo, userBanRepo)
	riceService := services.NewRiceService(pool, riceRepo, riceDotfilesRepo, riceTagRepo, commentRepo, userRepo, userBanRepo)
	tagService := services.NewTagService(tagRepo)
	userService := services.NewUserService(userRepo, userBanRepo, riceRepo)
	webVarService := services.NewWebVarService(webVarRepo)

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

	corsConfig := cors.DefaultConfig()
	if config.Config.Server.CorsOrigin != "" {
		corsConfig = cors.Config{
			AllowOrigins:     []string{config.Config.Server.CorsOrigin},
			AllowMethods:     []string{"GET", "POST", "DELETE", "PATCH"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
			AllowCredentials: true,
		}
	} else {
		corsConfig.AllowAllOrigins = true
		logger.Warn("Using default permissive CORS! Did you set origin in config?")
	}

	r.Use(
		gin.Recovery(),
		cors.New(corsConfig),
		security.LoggerMiddleware(logger),
		errs.ErrorHandler(logger),
		security.RateLimitMiddleware(100, time.Minute),
	)

	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Fatal("Failed to set trusted proxies", zap.Error(err))
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found on this server!"})
	})
	r.Static("/public", "./public")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"maintenance": config.Config.App.Maintenance})
	})
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "I'm working and responding!"})
	})
	r.POST("/webhook", webhookListener.Handler)

	adminMw := security.AdminMiddleware(userRepo, userBanRepo)

	registerAuthRoutes(r, authHandler)
	registerUserRoutes(r, userHandler, adminMw)
	registerRiceRoutes(r, riceHandler, riceDotfilesHandler, riceTagHandler, riceScreenshotHandler, riceStarHandler, adminMw)
	registerCommentRoutes(r, commentHandler, adminMw)
	registerReportRoutes(r, reportHandler, adminMw)
	registerTagRoutes(r, tagHandler, adminMw)
	registerProfileRoutes(r, profileHandler)
	registerAdminRoutes(r, adminHandler, adminMw)
	registerLinkRoutes(r, linkHandler)
	registerLeaderboardRoutes(r, leaderboardHandler)

	r.GET("/vars/:key", security.PathRateLimitMiddleware(5, time.Minute), webVarHandler.GetWebVarByKey)

	return r
}

func registerAuthRoutes(r *gin.Engine, h *handlers.AuthHandler) {
	auth := r.Group("/auth")
	auth.POST("/register", security.MaintenanceMiddleware(), h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", security.PathRateLimitMiddleware(20, time.Minute), h.RefreshToken)
	auth.POST("/logout", h.LogOut)
}

func registerUserRoutes(r *gin.Engine, h *handlers.UserHandler, adminMw gin.HandlerFunc) {
	maintenance := security.MaintenanceMiddleware()
	accountRL := security.PathRateLimitMiddleware(10, 24*time.Hour)

	users := r.Group("/users")

	users.GET("", h.ListUsers)
	users.GET("/:id/rices", security.PathRateLimitMiddleware(5, time.Minute), h.ListUserRices)
	users.GET("/:id/rices/:slug", security.PathRateLimitMiddleware(30, time.Minute), h.GetUserRiceBySlug)

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

	admin := users.Group("", adminMw)
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
	adminMw gin.HandlerFunc,
) {
	maintenance := security.MaintenanceMiddleware()
	updateRL := security.PathRateLimitMiddleware(10, time.Hour)
	limits := config.Config.Limits

	rices := r.Group("/rices")

	rices.GET("", rh.ListRices)
	rices.GET("/:id", rh.GetRiceByID)
	rices.GET("/:id/comments", rh.ListRiceComments)
	rices.GET("/:id/dotfiles",
		security.PathRateLimitMiddleware(3, time.Minute),
		dfh.DownloadDotfiles,
	)

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
	auth.PATCH("/:id/state", maintenance, adminMw, rh.UpdateRiceState)
	auth.POST("/:id/star", maintenance, sth.CreateRiceStar)
	auth.DELETE("/:id/star", maintenance, sth.DeleteRiceStar)
	auth.DELETE("/:id/screenshots/:previewId", maintenance, sch.DeleteScreenshot)
	auth.DELETE("/:id", maintenance, rh.DeleteRice)
}

func registerCommentRoutes(r *gin.Engine, h *handlers.CommentHandler, adminMw gin.HandlerFunc) {
	maintenance := security.MaintenanceMiddleware()
	comments := r.Group("/comments", security.AuthMiddleware)
	comments.GET("", adminMw, h.ListComments)
	comments.GET("/:id", security.PathRateLimitMiddleware(10, time.Minute), h.GetCommentByID)
	comments.POST("", maintenance, security.PathRateLimitMiddleware(10, time.Hour), h.CreateComment)
	comments.PATCH("/:id", maintenance, security.PathRateLimitMiddleware(10, time.Hour), h.UpdateComment)
	comments.DELETE("/:id", maintenance, h.DeleteComment)
}

func registerReportRoutes(r *gin.Engine, h *handlers.ReportHandler, adminMw gin.HandlerFunc) {
	reports := r.Group("/reports", security.AuthMiddleware)
	reports.POST("", security.PathRateLimitMiddleware(10, 24*time.Hour), h.CreateReport)
	admin := reports.Group("", adminMw)
	admin.GET("", h.ListReports)
	admin.GET("/:id", h.GetReportByID)
	admin.POST("/:id/close", h.CloseReport)
}

func registerTagRoutes(r *gin.Engine, h *handlers.TagHandler, adminMw gin.HandlerFunc) {
	tags := r.Group("/tags")
	tags.GET("", h.ListTags)
	admin := tags.Group("", security.AuthMiddleware, adminMw)
	admin.POST("", h.CreateTag)
	admin.PATCH("/:id", h.UpdateTag)
	admin.DELETE("/:id", h.DeleteTag)
}

func registerProfileRoutes(r *gin.Engine, h *handlers.ProfileHandler) {
	profiles := r.Group("/profiles")
	profiles.GET("/:username", h.GetProfileByUsername)
}

func registerAdminRoutes(r *gin.Engine, h *handlers.AdminHandler, adminMw gin.HandlerFunc) {
	admin := r.Group("/admin", security.AuthMiddleware, adminMw)
	admin.GET("/stats", h.ServiceStatistics)
}

func registerLinkRoutes(r *gin.Engine, h *handlers.LinkHandler) {
	links := r.Group("/links")
	links.GET("/subscription",
		security.PathRateLimitMiddleware(5, time.Minute),
		security.AuthMiddleware,
		h.GetSubscriptionLink,
	)
	links.GET("/:name",
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
