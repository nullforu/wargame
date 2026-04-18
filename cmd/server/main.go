package main

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wargame/internal/bootstrap"
	"wargame/internal/cache"
	"wargame/internal/config"
	"wargame/internal/db"
	httpserver "wargame/internal/http"
	"wargame/internal/logging"
	"wargame/internal/realtime"
	"wargame/internal/repo"
	"wargame/internal/service"
	"wargame/internal/stack"
	"wargame/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		boot := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		boot.Error("config error", slog.Any("error", err))
		os.Exit(1)
	}

	logger, err := logging.New(cfg.Logging, logging.Options{Service: "wargame", Env: cfg.AppEnv, AddSource: false})
	if err != nil {
		boot := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		boot.Error("logging init error", slog.Any("error", err))
		os.Exit(1)
	}
	slog.SetDefault(logger.Logger)
	defer func() { _ = logger.Close() }()

	ctx := context.Background()
	database, err := db.New(cfg.DB, cfg.AppEnv)
	if err != nil {
		logger.Error("db init error", slog.Any("error", err))
		os.Exit(1)
	}

	if err := database.PingContext(ctx); err != nil {
		logger.Error("db ping error", slog.Any("error", err))
		os.Exit(1)
	}

	redisClient := cache.New(cfg.Redis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("redis ping error", slog.Any("error", err))
		os.Exit(1)
	}

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(ctx, database); err != nil {
			logger.Error("auto migrate error", slog.Any("error", err))
			os.Exit(1)
		}
	}

	userRepo := repo.NewUserRepo(database)
	challengeRepo := repo.NewChallengeRepo(database)
	submissionRepo := repo.NewSubmissionRepo(database)
	scoreRepo := repo.NewScoreboardRepo(database)
	appConfigRepo := repo.NewAppConfigRepo(database)
	stackRepo := repo.NewStackRepo(database)

	var fileStore storage.ChallengeFileStore
	if cfg.S3.Enabled {
		store, err := storage.NewS3ChallengeFileStore(ctx, cfg.S3)
		if err != nil {
			logger.Error("s3 init error", slog.Any("error", err))
			os.Exit(1)
		}
		fileStore = store
	}

	authSvc := service.NewAuthService(cfg, userRepo, redisClient)
	userSvc := service.NewUserService(userRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	wargameSvc := service.NewWargameService(cfg, challengeRepo, submissionRepo, redisClient, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, redisClient, cfg.Cache.AppConfigTTL)

	var stackClient stack.API
	var stackClientCloser func() error
	if cfg.Stack.ProvisionerUseGRPC {
		client, err := stack.NewGRPCClient(cfg.Stack.ProvisionerGRPCAddr, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
		if err != nil {
			logger.Error("grpc stack client init error", slog.Any("error", err))
			os.Exit(1)
		}
		stackClient = client
		stackClientCloser = client.Close
	} else {
		stackClient = stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	}
	if stackClientCloser != nil {
		defer func() { _ = stackClientCloser() }()
	}

	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, stackClient, redisClient)
	bootstrap.BootstrapAdmin(ctx, cfg, database, userRepo, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sseHub := realtime.NewSSEHub()
	leaderboardBus := realtime.NewScoreboardBus(redisClient, cfg, scoreSvc, logger, sseHub)
	leaderboardBus.Start(ctx)

	router := httpserver.NewRouter(cfg, authSvc, wargameSvc, appConfigSvc, userSvc, scoreSvc, stackSvc, redisClient, logger, sseHub)
	srv := &nethttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)
	_ = redisClient.Close()
	_ = database.Close()
}
