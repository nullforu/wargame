package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/db"
	apphttp "wargame/internal/http"
	"wargame/internal/logging"
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

type testEnv struct {
	cfg             config.Config
	router          *gin.Engine
	userRepo        *repo.UserRepo
	affiliationRepo *repo.AffiliationRepo
	challengeRepo   *repo.ChallengeRepo
	submissionRepo  *repo.SubmissionRepo
	stackRepo       *repo.StackRepo
	authSvc         *service.AuthService
	wargameSvc      *service.WargameService
	stackSvc        *service.StackService
}

type errorResp struct {
	Error     string                 `json:"error"`
	Details   []service.FieldError   `json:"details"`
	RateLimit *service.RateLimitInfo `json:"rate_limit"`
}

var (
	testDB          *bun.DB
	testRedis       *redis.Client
	testCfg         config.Config
	pgContainer     testcontainers.Container
	redisServer     *miniredis.Miniredis
	skipIntegration bool
	testLogger      *logging.Logger
	logDir          string
)

func TestMain(m *testing.M) {
	skipIntegration = os.Getenv("WARGAME_SKIP_INTEGRATION") != ""
	if skipIntegration {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}

	pgContainer = container

	testDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, testDB); err != nil {
		panic(err)
	}

	redisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	testRedis = redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	testCfg = config.Config{
		AppEnv:          "test",
		HTTPAddr:        ":0",
		ShutdownTimeout: 5 * time.Second,
		AutoMigrate:     false,
		BcryptCost:      bcrypt.MinCost,
		DB:              dbCfg,
		Redis: config.RedisConfig{
			Addr:     redisServer.Addr(),
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
		Logging: config.LoggingConfig{
			Dir:          "",
			FilePrefix:   "test",
			MaxBodyBytes: 1024 * 1024,
		},
		Stack: config.StackConfig{
			Enabled:      true,
			MaxPer:       3,
			CreateWindow: time.Minute,
			CreateMax:    1,
		},
	}

	logDir, err = os.MkdirTemp("", "wargame-logs-*")
	if err != nil {
		panic(err)
	}

	testCfg.Logging.Dir = logDir

	testLogger, err = logging.New(testCfg.Logging, logging.Options{Service: "wargame", Env: "test"})
	if err != nil {
		panic(err)
	}

	code := m.Run()

	if testRedis != nil {
		_ = testRedis.Close()
	}

	if redisServer != nil {
		redisServer.Close()
	}

	if testDB != nil {
		_ = testDB.Close()
	}

	if testLogger != nil {
		_ = testLogger.Close()
	}

	if pgContainer != nil {
		_ = pgContainer.Terminate(ctx)
	}

	if logDir != "" {
		_ = os.RemoveAll(logDir)
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

func setupTest(t *testing.T, cfg config.Config) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	affiliationRepo := repo.NewAffiliationRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	voteRepo := repo.NewChallengeVoteRepo(testDB)
	writeupRepo := repo.NewWriteupRepo(testDB)
	scoreRepo := repo.NewScoreboardRepo(testDB)
	stackRepo := repo.NewStackRepo(testDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := service.NewAuthService(cfg, userRepo, testRedis)
	userSvc := service.NewUserService(userRepo, affiliationRepo, storage.NewMemoryProfileImageStore(10*time.Minute))
	affiliationSvc := service.NewAffiliationService(affiliationRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	wargameSvc := service.NewWargameService(cfg, challengeRepo, submissionRepo, voteRepo, writeupRepo, repo.NewChallengeCommentRepo(testDB), repo.NewCommunityRepo(testDB), testRedis, fileStore)
	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, &stack.MockClient{}, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, wargameSvc, userSvc, affiliationSvc, scoreSvc, stackSvc, testRedis, testLogger)

	env := testEnv{
		cfg:             cfg,
		router:          router,
		userRepo:        userRepo,
		affiliationRepo: affiliationRepo,
		challengeRepo:   challengeRepo,
		submissionRepo:  submissionRepo,
		stackRepo:       stackRepo,
		authSvc:         authSvc,
		wargameSvc:      wargameSvc,
		stackSvc:        stackSvc,
	}

	return env
}

func resetState(t *testing.T) {
	t.Helper()

	if _, err := testDB.ExecContext(context.Background(), "TRUNCATE TABLE community_comments, challenge_comments, challenge_votes, writeups, community_post_likes, community_posts, submissions, stacks, challenges, users, affiliations RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := testRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfIntegrationDisabled(t *testing.T) {
	t.Helper()

	if skipIntegration {
		t.Skip("integration tests disabled via WARGAME_SKIP_INTEGRATION")
	}
}

func doRequest(t *testing.T, router *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			reader = bytes.NewBufferString(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			reader = bytes.NewBuffer(data)
		}
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		if req.Header.Get("X-CSRF-Token") == "" {
			if cookieHeader := req.Header.Get("Cookie"); strings.Contains(cookieHeader, "csrf_token=") {
				for part := range strings.SplitSeq(cookieHeader, ";") {
					trimmed := strings.TrimSpace(part)
					if after, ok := strings.CutPrefix(trimmed, "csrf_token="); ok {
						req.Header.Set("X-CSRF-Token", after)
						break
					}
				}
			}
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func authHeader(token string) map[string]string {
	return map[string]string{"Cookie": "access_token=" + token + "; csrf_token=test-csrf", "X-CSRF-Token": "test-csrf"}
}

func refreshHeader(token string) map[string]string {
	return map[string]string{"Cookie": "refresh_token=" + token + "; csrf_token=test-csrf", "X-CSRF-Token": "test-csrf"}
}

func cookieValueFromSetCookie(rec *httptest.ResponseRecorder, name string) string {
	prefix := name + "="
	for _, c := range rec.Header().Values("Set-Cookie") {
		if after, ok := strings.CutPrefix(c, prefix); ok {
			return strings.SplitN(after, ";", 2)[0]
		}
	}

	return ""
}

func registerAndLogin(t *testing.T, env testEnv, email, username, password string) (string, string, int64) {
	t.Helper()
	regBody := map[string]string{
		"email":    email,
		"username": username,
		"password": password,
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}

	decodeJSON(t, rec, &regResp)

	loginBody := map[string]string{
		"email":    email,
		"password": password,
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/login", loginBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		User struct {
			ID         int64 `json:"id"`
			StackCount int   `json:"stack_count"`
			StackLimit int   `json:"stack_limit"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &loginResp)
	if loginResp.User.StackCount != 0 {
		t.Fatalf("expected stack_count 0, got %d", loginResp.User.StackCount)
	}

	if loginResp.User.StackLimit != env.cfg.Stack.MaxPer {
		t.Fatalf("expected stack_limit %d, got %d", env.cfg.Stack.MaxPer, loginResp.User.StackLimit)
	}

	accessToken := cookieValueFromSetCookie(rec, "access_token")
	refreshToken := cookieValueFromSetCookie(rec, "refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("missing auth cookies from login response")
	}

	return accessToken, refreshToken, loginResp.User.ID
}

func createUser(t *testing.T, env testEnv, email, username, password, role string) *models.User {
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

func loginUser(t *testing.T, router *gin.Engine, email, password string) (string, string, int64) {
	t.Helper()
	body := map[string]string{"email": email, "password": password}
	rec := doRequest(t, router, http.MethodPost, "/api/auth/login", body, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		User struct {
			ID         int64 `json:"id"`
			StackCount int   `json:"stack_count"`
			StackLimit int   `json:"stack_limit"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &resp)
	accessToken := cookieValueFromSetCookie(rec, "access_token")
	refreshToken := cookieValueFromSetCookie(rec, "refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("missing auth cookies from login response")
	}

	return accessToken, refreshToken, resp.User.ID
}

func ensureAdminUser(t *testing.T, env testEnv) *models.User {
	t.Helper()

	user, err := env.userRepo.GetByEmail(context.Background(), "admin@example.com")
	if err == nil {
		return user
	}
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("get admin: %v", err)
	}

	return createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
}

func createChallenge(t *testing.T, env testEnv, title string, points int, flag string, active bool) *models.Challenge {
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

func createStackChallenge(t *testing.T, env testEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 80\n          protocol: TCP\n"

	challenge := &models.Challenge{
		Title:        title,
		Description:  "stack desc",
		Category:     "Web",
		Points:       100,
		StackEnabled: true,
		StackTargetPorts: stack.TargetPortSpecs{
			{ContainerPort: 80, Protocol: "TCP"},
		},
		StackPodSpec: &podSpec,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	hash, err := utils.HashFlag("flag{stack}", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}

	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func setupStackTest(t *testing.T, cfg config.Config, mockClient stack.API) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	affiliationRepo := repo.NewAffiliationRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	voteRepo := repo.NewChallengeVoteRepo(testDB)
	writeupRepo := repo.NewWriteupRepo(testDB)
	scoreRepo := repo.NewScoreboardRepo(testDB)
	stackRepo := repo.NewStackRepo(testDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := service.NewAuthService(cfg, userRepo, testRedis)
	userSvc := service.NewUserService(userRepo, affiliationRepo, storage.NewMemoryProfileImageStore(10*time.Minute))
	affiliationSvc := service.NewAffiliationService(affiliationRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	wargameSvc := service.NewWargameService(cfg, challengeRepo, submissionRepo, voteRepo, writeupRepo, repo.NewChallengeCommentRepo(testDB), repo.NewCommunityRepo(testDB), testRedis, fileStore)
	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, mockClient, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, wargameSvc, userSvc, affiliationSvc, scoreSvc, stackSvc, testRedis, testLogger)

	return testEnv{
		cfg:             cfg,
		router:          router,
		userRepo:        userRepo,
		affiliationRepo: affiliationRepo,
		challengeRepo:   challengeRepo,
		submissionRepo:  submissionRepo,
		stackRepo:       stackRepo,
		authSvc:         authSvc,
		wargameSvc:      wargameSvc,
		stackSvc:        stackSvc,
	}
}

func createSubmission(t *testing.T, env testEnv, userID, challengeID int64, correct bool, submittedAt time.Time) {
	t.Helper()

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    "flag{test}",
		Correct:     correct,
		SubmittedAt: submittedAt,
	}

	if err := env.submissionRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}
}

func createAffiliation(t *testing.T, env testEnv, name string) *models.Affiliation {
	t.Helper()

	affiliation := &models.Affiliation{
		Name:      name,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := env.affiliationRepo.Create(context.Background(), affiliation); err != nil {
		t.Fatalf("create affiliation: %v", err)
	}

	return affiliation
}

func assertFieldErrors(t *testing.T, got []service.FieldError, expected map[string]string) {
	t.Helper()

	found := make(map[string]string, len(got))

	for _, fe := range got {
		found[fe.Field] = fe.Reason
	}

	for field, reason := range expected {
		if found[field] != reason {
			t.Fatalf("expected field %s reason %s, got %q", field, reason, found[field])
		}
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
