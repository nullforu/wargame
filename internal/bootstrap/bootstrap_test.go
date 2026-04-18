package bootstrap

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"wargame/internal/config"
	"wargame/internal/logging"
	"wargame/internal/models"
	"wargame/internal/repo"

	"golang.org/x/crypto/bcrypt"
)

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	dir := t.TempDir()
	logger, err := logging.New(config.LoggingConfig{
		Dir:          filepath.Join(dir, "logs"),
		FilePrefix:   "test",
		MaxBodyBytes: 1024,
	}, logging.Options{
		Service: "bootstrap-test",
		Env:     "test",
	})

	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	t.Cleanup(func() {
		_ = logger.Close()
	})

	return logger
}

func baseBootstrapConfig() config.Config {
	return config.Config{
		AppEnv:     "test",
		BcryptCost: bcrypt.MinCost,
		Bootstrap: config.BootstrapConfig{
			AdminUserEnabled: true,
			AdminEmail:       "admin@example.com",
			AdminPassword:    "adminpass",
			AdminUsername:    "admin",
		},
	}
}

func TestIsDatabaseEmpty(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	empty, err := isDatabaseEmpty(context.Background(), db)
	if err != nil {
		t.Fatalf("isDatabaseEmpty: %v", err)
	}

	if !empty {
		t.Fatalf("expected empty database")
	}

	now := time.Now().UTC()
	seed := &models.User{
		Email:        "seed@example.com",
		Username:     "seed",
		PasswordHash: "hash",
		Role:         models.UserRole,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userRepo.Create(context.Background(), seed); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	empty, err = isDatabaseEmpty(context.Background(), db)
	if err != nil {
		t.Fatalf("isDatabaseEmpty after insert: %v", err)
	}

	if empty {
		t.Fatalf("expected database to be non-empty")
	}
}

func TestEnsureAdminUser(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()

	user, err := ensureAdminUser(context.Background(), cfg, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser: %v", err)
	}

	if user == nil || user.Role != models.AdminRole {
		t.Fatalf("unexpected admin user")
	}

	user, err = ensureAdminUser(context.Background(), cfg, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser second call: %v", err)
	}

	if user != nil {
		t.Fatalf("expected no user on duplicate create")
	}
}

func TestBootstrapAdminCreatesUser(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected 1 user, got %d", userCount)
	}
}

func TestBootstrapAdminSkipsWhenNotEmpty(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	now := time.Now().UTC()
	if err := userRepo.Create(context.Background(), &models.User{Email: "existing@example.com", Username: "existing", PasswordHash: "hash", Role: models.UserRole, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	cfg := baseBootstrapConfig()
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected bootstrap skip with existing user, got count %d", userCount)
	}
}

func TestBootstrapAdminDisabled(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()
	cfg.Bootstrap.AdminUserEnabled = false
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 0 {
		t.Fatalf("expected 0 users, got %d", userCount)
	}
}

func TestEnsureAdminUserSkipsWhenCredentialsMissing(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()
	cfg.Bootstrap.AdminEmail = " "
	cfg.Bootstrap.AdminPassword = ""

	user, err := ensureAdminUser(context.Background(), cfg, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser: %v", err)
	}

	if user != nil {
		t.Fatalf("expected nil user when credentials missing")
	}
}

func TestEnsureAdminUserDefaultsUsername(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()
	cfg.Bootstrap.AdminUsername = " "

	user, err := ensureAdminUser(context.Background(), cfg, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser: %v", err)
	}

	if user == nil || user.Username != "admin" {
		t.Fatalf("expected default username 'admin', got %+v", user)
	}
}

func TestBootstrapAdminDisabledLeavesExistingUsersUntouched(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	now := time.Now().UTC()
	if err := userRepo.Create(context.Background(), &models.User{Email: "existing@example.com", Username: "existing", PasswordHash: "hash", Role: models.UserRole, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	cfg := baseBootstrapConfig()
	cfg.Bootstrap.AdminUserEnabled = false
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected 1 existing user, got %d", userCount)
	}
}

func TestBootstrapAdminEnabledCreatesOnlyOnce(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)

	cfg := baseBootstrapConfig()
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)
	BootstrapAdmin(context.Background(), cfg, db, userRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected exactly 1 admin user, got %d", userCount)
	}
}
