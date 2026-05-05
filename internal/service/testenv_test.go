package service

import (
	"context"
	"os"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"
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

type serviceEnv struct {
	cfg             config.Config
	db              *bun.DB
	redis           *redis.Client
	userRepo        *repo.UserRepo
	affiliationRepo *repo.AffiliationRepo
	challengeRepo   *repo.ChallengeRepo
	submissionRepo  *repo.SubmissionRepo
	scoreRepo       *repo.ScoreboardRepo
	stackRepo       *repo.StackRepo
	authSvc         *AuthService
	userSvc         *UserService
	affiliationSvc  *AffiliationService
	scoreSvc        *ScoreboardService
	wargameSvc      *WargameService
	stackSvc        *StackService
}

var (
	serviceDB          *bun.DB
	serviceRedis       *redis.Client
	serviceCfg         config.Config
	servicePGContainer testcontainers.Container
	serviceRedisServer *miniredis.Miniredis
	skipServiceEnv     bool
)

func TestMain(m *testing.M) {
	skipServiceEnv = os.Getenv("WARGAME_SKIP_INTEGRATION") != ""
	if skipServiceEnv {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}
	servicePGContainer = container

	serviceDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, serviceDB); err != nil {
		panic(err)
	}

	serviceRedisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	serviceRedis = redis.NewClient(&redis.Options{Addr: serviceRedisServer.Addr()})

	serviceCfg = config.Config{
		AppEnv:          "test",
		HTTPAddr:        ":0",
		ShutdownTimeout: 5 * time.Second,
		AutoMigrate:     false,
		BcryptCost:      bcrypt.MinCost,
		DB:              dbCfg,
		Redis: config.RedisConfig{
			Addr:     serviceRedisServer.Addr(),
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
	}

	code := m.Run()

	if serviceRedis != nil {
		_ = serviceRedis.Close()
	}

	if serviceRedisServer != nil {
		serviceRedisServer.Close()
	}

	if serviceDB != nil {
		_ = serviceDB.Close()
	}

	if servicePGContainer != nil {
		_ = servicePGContainer.Terminate(ctx)
	}

	os.Exit(code)
}

func startPostgres(ctx context.Context) (testcontainers.Container, config.DBConfig, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "wargame",
			"POSTGRES_PASSWORD": "wargame",
			"POSTGRES_DB":       "wargame_test",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp").SkipExternalCheck(),
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

func setupServiceTest(t *testing.T) serviceEnv {
	t.Helper()
	skipIfServiceDisabled(t)
	resetServiceState(t)

	userRepo := repo.NewUserRepo(serviceDB)
	affiliationRepo := repo.NewAffiliationRepo(serviceDB)
	challengeRepo := repo.NewChallengeRepo(serviceDB)
	submissionRepo := repo.NewSubmissionRepo(serviceDB)
	voteRepo := repo.NewChallengeVoteRepo(serviceDB)
	writeupRepo := repo.NewWriteupRepo(serviceDB)
	scoreRepo := repo.NewScoreboardRepo(serviceDB)
	stackRepo := repo.NewStackRepo(serviceDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := NewAuthService(serviceCfg, userRepo, serviceRedis)
	userSvc := NewUserService(userRepo, affiliationRepo, storage.NewMemoryProfileImageStore(10*time.Minute))
	affiliationSvc := NewAffiliationService(affiliationRepo)
	scoreSvc := NewScoreboardService(scoreRepo)
	wargameSvc := NewWargameService(serviceCfg, challengeRepo, submissionRepo, voteRepo, writeupRepo, repo.NewChallengeCommentRepo(serviceDB), repo.NewCommunityRepo(serviceDB), serviceRedis, fileStore)
	stackSvc := NewStackService(serviceCfg.Stack, stackRepo, challengeRepo, submissionRepo, &stack.MockClient{}, serviceRedis)

	env := serviceEnv{
		cfg:             serviceCfg,
		db:              serviceDB,
		redis:           serviceRedis,
		userRepo:        userRepo,
		affiliationRepo: affiliationRepo,
		challengeRepo:   challengeRepo,
		submissionRepo:  submissionRepo,
		scoreRepo:       scoreRepo,
		stackRepo:       stackRepo,
		authSvc:         authSvc,
		userSvc:         userSvc,
		affiliationSvc:  affiliationSvc,
		scoreSvc:        scoreSvc,
		wargameSvc:      wargameSvc,
		stackSvc:        stackSvc,
	}

	return env
}

func resetServiceState(t *testing.T) {
	t.Helper()

	if _, err := serviceDB.ExecContext(context.Background(), "TRUNCATE TABLE community_comments, challenge_comments, challenge_votes, writeups, community_post_likes, community_posts, submissions, stacks, challenges, users, affiliations RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := serviceRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfServiceDisabled(t *testing.T) {
	t.Helper()

	if skipServiceEnv {
		t.Skip("service tests disabled via WARGAME_SKIP_INTEGRATION")
	}
}

func createUser(t *testing.T, env serviceEnv, email, username, password, role string) *models.User {
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

func createChallenge(t *testing.T, env serviceEnv, title string, points int, flag string, active bool) *models.Challenge {
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

func createSubmission(t *testing.T, env serviceEnv, userID, challengeID int64, correct bool, submittedAt time.Time) *models.Submission {
	t.Helper()

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Correct:     correct,
		SubmittedAt: submittedAt,
	}

	if err := env.submissionRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}

	return sub
}
