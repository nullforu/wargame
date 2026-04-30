package http

import (
	nethttp "net/http"

	"wargame/internal/config"
	"wargame/internal/http/handlers"
	"wargame/internal/http/middleware"
	"wargame/internal/logging"
	"wargame/internal/models"
	"wargame/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, authSvc *service.AuthService, wargameSvc *service.WargameService, userSvc *service.UserService, affiliationSvc *service.AffiliationService, scoreSvc *service.ScoreboardService, stackSvc *service.StackService, redis *redis.Client, logger *logging.Logger) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RecoveryLogger(logger))
	r.Use(middleware.RequestLogger(cfg.Logging, logger))
	r.Use(middleware.CORS(false, cfg.CORS.AllowedOrigins))
	r.Use(middleware.CSRF())

	h := handlers.New(cfg, authSvc, wargameSvc, userSvc, affiliationSvc, scoreSvc, stackSvc, redis)

	r.GET("/healthz", func(ctx *gin.Context) { ctx.JSON(nethttp.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api")
	{
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/logout", h.Logout)

		api.GET("/challenges", h.ListChallenges)
		api.GET("/challenges/search", h.SearchChallenges)
		api.GET("/challenges/:id", h.GetChallenge)
		api.GET("/challenges/:id/solvers", h.ChallengeSolvers)
		api.GET("/challenges/:id/writeups", h.ChallengeWriteups)
		api.GET("/challenges/:id/challenge-comments", h.ChallengeComments)
		api.GET("/challenges/:id/votes", h.ChallengeVotes)
		api.GET("/writeups/:id", h.GetWriteup)
		api.GET("/leaderboard", h.Leaderboard)
		api.GET("/timeline", h.Timeline)
		api.GET("/rankings/users", h.RankingUsers)
		api.GET("/rankings/affiliations", h.RankingAffiliations)
		api.GET("/rankings/affiliations/:id/users", h.RankingAffiliationUsers)
		api.GET("/affiliations", h.ListAffiliations)
		api.GET("/affiliations/search", h.SearchAffiliations)
		api.GET("/affiliations/:id/users", h.ListAffiliationUsers)
		api.GET("/users", h.ListUsers)
		api.GET("/users/search", h.SearchUsers)
		api.GET("/users/:id", h.GetUser)
		api.GET("/users/:id/solved", h.GetUserSolved)
		api.GET("/users/:id/writeups", h.GetUserWriteups)

		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWT))
		auth.GET("/me", h.Me)
		auth.GET("/me/writeups", h.MyWriteups)
		auth.GET("/challenges/:id/my-writeup", h.MyChallengeWriteup)
		auth.GET("/stacks", h.ListStacks)
		auth.GET("/challenges/:id/stack", h.GetStack)
		auth.GET("/challenges/:id/my-vote", h.ChallengeMyVote)

		unblocked := auth.Group("")
		unblocked.Use(middleware.RequireActiveUser(userSvc))
		unblocked.PUT("/me", h.UpdateMe)
		unblocked.POST("/challenges/:id/submit", h.SubmitFlag)
		unblocked.POST("/challenges/:id/writeups", h.CreateWriteup)
		unblocked.POST("/challenges/:id/challenge-comments", h.CreateChallengeCommentItem)
		unblocked.PATCH("/writeups/:id", h.UpdateWriteup)
		unblocked.DELETE("/writeups/:id", h.DeleteWriteup)
		unblocked.PATCH("/challenges/challenge-comments/:id", h.UpdateChallengeCommentItem)
		unblocked.DELETE("/challenges/challenge-comments/:id", h.DeleteChallengeCommentItem)
		unblocked.POST("/challenges/:id/vote", h.VoteChallengeLevel)
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
		admin.POST("/affiliations", h.AdminCreateAffiliation)
	}

	return r
}
