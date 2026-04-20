package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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
		WaitingFor: wait.ForListeningPort("5432/tcp"),
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
	scoreRepo := repo.NewScoreboardRepo(handlerDB)
	stackRepo := repo.NewStackRepo(handlerDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(handlerCfg, userRepo, handlerRedis)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	wargameSvc := service.NewWargameService(handlerCfg, challengeRepo, submissionRepo, handlerRedis, fileStore)
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

	if _, err := handlerDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, stacks, challenges, users RESTART IDENTITY CASCADE"); err != nil {
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
		Title:         title,
		Description:   "desc",
		Category:      "Misc",
		Points:        points,
		MinimumPoints: points,
		IsActive:      active,
		CreatedAt:     time.Now().UTC(),
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

func TestParsePaginationParams(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/users", nil)
		page, pageSize, ok := parsePaginationParams(ctx)
		if !ok {
			t.Fatalf("expected ok")
		}

		if page != 0 || pageSize != 0 {
			t.Fatalf("expected zero values before normalization, got page=%d pageSize=%d", page, pageSize)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=abc", nil)
		page, pageSize, ok := parsePaginationParams(ctx)
		if ok || page != 0 || pageSize != 0 {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid page_size", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page_size=abc", nil)
		_, _, ok := parsePaginationParams(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestParseSearchQuery(t *testing.T) {
	t.Run("success with trim", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=%20web%20", nil)
		q, ok := parseSearchQuery(ctx)
		if !ok || q != "web" {
			t.Fatalf("unexpected q=%q ok=%v", q, ok)
		}
	})

	t.Run("required", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=", nil)
		_, ok := parseSearchQuery(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestParseChallengeFilters(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges?category=Web&level=3&solved=true", nil)
		filters, ok := parseChallengeFilters(ctx)
		if !ok || filters.Category != "Web" || filters.Level == nil || *filters.Level != 3 || filters.Solved == nil || !*filters.Solved {
			t.Fatalf("unexpected filters: ok=%v filters=%+v", ok, filters)
		}
	})

	t.Run("invalid level", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?level=bad", nil)
		_, ok := parseChallengeFilters(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid solved", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?solved=maybe", nil)
		_, ok := parseChallengeFilters(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestPreviousChallengeForResponse(t *testing.T) {
	env := setupHandlerTest(t)
	prev := createHandlerChallenge(t, env, "Prev", 100, "FLAG{P}", true)
	other := createHandlerChallenge(t, env, "Other", 100, "FLAG{O}", true)

	t.Run("from current page map", func(t *testing.T) {
		byID := map[int64]*models.Challenge{prev.ID: prev}
		got := env.handler.previousChallengeForResponse(context.Background(), byID, &prev.ID)
		if got == nil || got.ID != prev.ID {
			t.Fatalf("expected previous from map, got %+v", got)
		}
	})

	t.Run("from repository fallback", func(t *testing.T) {
		byID := map[int64]*models.Challenge{}
		got := env.handler.previousChallengeForResponse(context.Background(), byID, &other.ID)
		if got == nil || got.ID != other.ID {
			t.Fatalf("expected previous from fallback, got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		missingID := int64(999999)
		got := env.handler.previousChallengeForResponse(context.Background(), map[int64]*models.Challenge{}, &missingID)
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})
}

func TestHandlerSearchChallengesAndUsers(t *testing.T) {
	env := setupHandlerTest(t)
	_ = createHandlerUser(t, env, "alpha@example.com", "alpha", "pass", models.UserRole)
	_ = createHandlerUser(t, env, "beta@example.com", "beta", "pass", models.UserRole)

	prev := createHandlerChallenge(t, env, "Web Prev", 100, "FLAG{1}", true)
	locked := createHandlerChallenge(t, env, "Web Locked", 200, "FLAG{2}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update challenge prerequisite: %v", err)
	}

	t.Run("search users success", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/search?q=alp&page=1&page_size=10", nil)
		env.handler.SearchUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp usersListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Users) != 1 || resp.Users[0].Username != "alpha" {
			t.Fatalf("unexpected users response: %+v", resp)
		}
	})

	t.Run("search users invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/search?q=alpha&page=abc", nil)
		env.handler.SearchUsers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("search challenges success with fallback previous challenge", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=Locked&page=1&page_size=1", nil)
		env.handler.SearchChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Challenges []lockedChallengeResponse `json:"challenges"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Challenges) != 1 {
			t.Fatalf("expected one challenge, got %d", len(resp.Challenges))
		}

		if resp.Challenges[0].PreviousChallengeTitle == nil || *resp.Challenges[0].PreviousChallengeTitle != prev.Title {
			t.Fatalf("expected previous challenge title fallback, got %+v", resp.Challenges[0])
		}
	})

	t.Run("search challenges missing query", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=", nil)
		env.handler.SearchChallenges(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandlerListChallengesAndUsers(t *testing.T) {
	env := setupHandlerTest(t)
	user1 := createHandlerUser(t, env, "user1@example.com", "user1", "pass", models.UserRole)
	_ = createHandlerUser(t, env, "user2@example.com", "user2", "pass", models.UserRole)
	_ = createHandlerChallenge(t, env, "Challenge 1", 100, "FLAG{1}", true)
	_ = createHandlerChallenge(t, env, "Challenge 2", 200, "FLAG{2}", true)

	t.Run("list users", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=1&page_size=1", nil)
		env.handler.ListUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp usersListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Users) != 1 || resp.Pagination.TotalCount != 2 {
			t.Fatalf("unexpected users list response: %+v", resp)
		}
	})

	t.Run("list users invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=bad", nil)
		env.handler.ListUsers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list challenges", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page=1&page_size=1", nil)
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengesListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Challenges) != 1 || resp.Pagination.TotalCount != 2 {
			t.Fatalf("unexpected challenges list response: %+v", resp)
		}
	})

	t.Run("list challenges with auth and prerequisite", func(t *testing.T) {
		prev := createHandlerChallenge(t, env, "Prev Auth", 100, "FLAG{P}", true)
		next := createHandlerChallenge(t, env, "Next Auth", 200, "FLAG{N}", true)
		next.PreviousChallengeID = &prev.ID
		if err := env.challengeRepo.Update(context.Background(), next); err != nil {
			t.Fatalf("update next challenge prerequisite: %v", err)
		}

		accessToken, _, _, err := env.authSvc.Login(context.Background(), user1.Email, "pass")
		if err != nil {
			t.Fatalf("login for auth list challenge: %v", err)
		}

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page=1&page_size=10", nil)
		ctx.Request.Header.Set("Authorization", "Bearer "+accessToken)
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengesListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.Pagination.TotalCount < 4 {
			t.Fatalf("expected expanded challenge count, got %+v", resp.Pagination)
		}
	})

	t.Run("list challenges invalid page_size", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page_size=bad", nil)
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}
