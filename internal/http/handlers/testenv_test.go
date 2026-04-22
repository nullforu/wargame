package handlers

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/service"
	"wargame/internal/stack"
	"wargame/internal/storage"
	"wargame/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type handlerEnv struct {
	cfg            config.Config
	db             *bun.DB
	redis          *redis.Client
	userRepo       *repo.UserRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	stackRepo      *repo.StackRepo
	authSvc        *service.AuthService
	userSvc        *service.UserService
	scoreSvc       *service.ScoreboardService
	wargameSvc     *service.WargameService
	stackSvc       *service.StackService
	handler        *Handler
}

var (
	handlerDB          *bun.DB
	handlerRedis       *redis.Client
	handlerCfg         config.Config
	handlerPGContainer testcontainers.Container
	handlerRedisServer *miniredis.Miniredis
	skipHandlerEnv     bool
)

func TestMain(m *testing.M) {
	skipHandlerEnv = os.Getenv("WARGAME_SKIP_INTEGRATION") != ""
	if skipHandlerEnv {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	container, dbCfg, err := startHandlerPostgres(ctx)
	if err != nil {
		panic(err)
	}
	handlerPGContainer = container

	handlerDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, handlerDB); err != nil {
		panic(err)
	}

	handlerRedisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	handlerRedis = redis.NewClient(&redis.Options{Addr: handlerRedisServer.Addr()})

	handlerCfg = config.Config{
		AppEnv:          "test",
		HTTPAddr:        ":0",
		ShutdownTimeout: 5 * time.Second,
		AutoMigrate:     false,
		BcryptCost:      bcrypt.MinCost,
		DB:              dbCfg,
		Redis: config.RedisConfig{
			Addr:     handlerRedisServer.Addr(),
			Password: "",
			DB:       0,
			PoolSize: 5,
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Issuer:     "wargame-test",
			AccessTTL:  time.Hour,
			RefreshTTL: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			SubmissionWindow: 2 * time.Minute,
			SubmissionMax:    5,
		},
		Cache: config.CacheConfig{
			TimelineTTL:    2 * time.Minute,
			LeaderboardTTL: 2 * time.Minute,
		},
		Stack: config.StackConfig{
			Enabled:      true,
			MaxPer:       3,
			CreateWindow: time.Minute,
			CreateMax:    1,
		},
	}

	code := m.Run()

	if handlerRedis != nil {
		_ = handlerRedis.Close()
	}

	if handlerRedisServer != nil {
		handlerRedisServer.Close()
	}

	if handlerDB != nil {
		_ = handlerDB.Close()
	}

	if handlerPGContainer != nil {
		_ = handlerPGContainer.Terminate(ctx)
	}

	os.Exit(code)
}

func startHandlerPostgres(ctx context.Context) (testcontainers.Container, config.DBConfig, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "wargame",
			"POSTGRES_PASSWORD": "wargame",
			"POSTGRES_DB":       "wargame_test",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, config.DBConfig{}, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, config.DBConfig{}, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, config.DBConfig{}, err
	}

	cfg := config.DBConfig{
		Host:            host,
		Port:            port.Int(),
		User:            "wargame",
		Password:        "wargame",
		Name:            "wargame_test",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 2 * time.Minute,
	}

	return container, cfg, nil
}

func setupHandlerTest(t *testing.T) handlerEnv {
	t.Helper()
	skipIfHandlerDisabled(t)
	resetHandlerState(t)

	userRepo := repo.NewUserRepo(handlerDB)
	challengeRepo := repo.NewChallengeRepo(handlerDB)
	submissionRepo := repo.NewSubmissionRepo(handlerDB)
	voteRepo := repo.NewChallengeVoteRepo(handlerDB)
	scoreRepo := repo.NewScoreboardRepo(handlerDB)
	stackRepo := repo.NewStackRepo(handlerDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(handlerCfg, userRepo, handlerRedis)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	wargameSvc := service.NewWargameService(handlerCfg, challengeRepo, submissionRepo, voteRepo, handlerRedis, fileStore)
	stackSvc := service.NewStackService(handlerCfg.Stack, stackRepo, challengeRepo, submissionRepo, &stack.MockClient{}, handlerRedis)

	handler := New(handlerCfg, authSvc, wargameSvc, userSvc, scoreSvc, stackSvc, handlerRedis)

	env := handlerEnv{
		cfg:            handlerCfg,
		db:             handlerDB,
		redis:          handlerRedis,
		userRepo:       userRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		stackRepo:      stackRepo,
		authSvc:        authSvc,
		userSvc:        userSvc,
		scoreSvc:       scoreSvc,
		wargameSvc:     wargameSvc,
		stackSvc:       stackSvc,
		handler:        handler,
	}

	return env
}

func resetHandlerState(t *testing.T) {
	t.Helper()

	if _, err := handlerDB.ExecContext(context.Background(), "TRUNCATE TABLE challenge_votes, submissions, stacks, challenges, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := handlerRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfHandlerDisabled(t *testing.T) {
	t.Helper()

	if skipHandlerEnv {
		t.Skip("handler tests disabled via WARGAME_SKIP_INTEGRATION")
	}
}

func createHandlerUser(t *testing.T, env handlerEnv, email, username, password, role string) *models.User {
	t.Helper()

	hash, err := auth.HashPassword(password, env.cfg.BcryptCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := env.userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

func createHandlerChallenge(t *testing.T, env handlerEnv, title string, points int, flag string, active bool) *models.Challenge {
	t.Helper()
	challenge := &models.Challenge{
		Title:       title,
		Description: "desc",
		Category:    "Misc",
		Points:      points,
		IsActive:    active,
		CreatedAt:   time.Now().UTC(),
	}

	hash, err := utils.HashFlag(flag, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}

	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func createHandlerSubmission(t *testing.T, env handlerEnv, userID, challengeID int64, correct bool, submittedAt time.Time) *models.Submission {
	t.Helper()

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    "flag",
		Correct:     correct,
		SubmittedAt: submittedAt,
	}

	if err := env.submissionRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}

	return sub
}

func newJSONContext(t *testing.T, method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, path, nil)
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	return ctx, rec
}
