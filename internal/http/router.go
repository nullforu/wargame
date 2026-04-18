package http

import (
	nethttp "net/http"

	"wargame/internal/config"
	"wargame/internal/http/handlers"
	"wargame/internal/http/middleware"
	"wargame/internal/logging"
	"wargame/internal/models"
	"wargame/internal/realtime"
	"wargame/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, authSvc *service.AuthService, wargameSvc *service.WargameService, userSvc *service.UserService, scoreSvc *service.ScoreboardService, stackSvc *service.StackService, redis *redis.Client, logger *logging.Logger, sse *realtime.SSEHub) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RecoveryLogger(logger))
	r.Use(middleware.RequestLogger(cfg.Logging, logger))
	r.Use(middleware.CORS(cfg.AppEnv != "production", cfg.CORS.AllowedOrigins))

	h := handlers.New(cfg, authSvc, wargameSvc, userSvc, scoreSvc, stackSvc, redis)
	sseHandler := handlers.NewSSEHandler(sse)

	r.GET("/healthz", func(ctx *gin.Context) { ctx.JSON(nethttp.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api")
	{
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/logout", h.Logout)

		api.GET("/challenges", h.ListChallenges)
		api.GET("/scoreboard/stream", sseHandler.ScoreboardStream)
		api.GET("/leaderboard", h.Leaderboard)
		api.GET("/timeline", h.Timeline)
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id", h.GetUser)
		api.GET("/users/:id/solved", h.GetUserSolved)

		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWT))
		auth.GET("/me", h.Me)
		auth.GET("/stacks", h.ListStacks)
		auth.GET("/challenges/:id/stack", h.GetStack)

		unblocked := auth.Group("")
		unblocked.Use(middleware.RequireActiveUser(userSvc))
		unblocked.PUT("/me", h.UpdateMe)
		unblocked.POST("/challenges/:id/submit", h.SubmitFlag)
		unblocked.POST("/challenges/:id/file/download", h.RequestChallengeFileDownload)
		unblocked.POST("/challenges/:id/stack", h.CreateStack)
		unblocked.DELETE("/challenges/:id/stack", h.DeleteStack)

		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.JWT), middleware.RequireActiveUser(userSvc), middleware.RequireRole(models.AdminRole))
		admin.POST("/challenges", h.CreateChallenge)
		admin.GET("/challenges/:id", h.AdminGetChallenge)
		admin.PUT("/challenges/:id", h.UpdateChallenge)
		admin.DELETE("/challenges/:id", h.DeleteChallenge)
		admin.POST("/challenges/:id/file/upload", h.RequestChallengeFileUpload)
		admin.DELETE("/challenges/:id/file", h.DeleteChallengeFile)
		admin.GET("/stacks", h.AdminListStacks)
		admin.GET("/stacks/:stack_id", h.AdminGetStack)
		admin.DELETE("/stacks/:stack_id", h.AdminDeleteStack)
		admin.POST("/users/:id/block", h.AdminBlockUser)
		admin.POST("/users/:id/unblock", h.AdminUnblockUser)
	}

	return r
}
