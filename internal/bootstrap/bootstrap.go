package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/db"
	"wargame/internal/logging"
	"wargame/internal/models"
	"wargame/internal/repo"

	"github.com/uptrace/bun"
)

func BootstrapAdmin(ctx context.Context, cfg config.Config, database *bun.DB, userRepo *repo.UserRepo, logger *logging.Logger) {
	if !cfg.Bootstrap.AdminUserEnabled {
		return
	}

	empty, err := isDatabaseEmpty(ctx, database)
	if err != nil {
		logger.Error("bootstrap database check error", slog.Any("error", err))
		return
	}
	if !empty {
		logger.Info("bootstrap skipped: database is not empty")
		return
	}

	user, err := ensureAdminUser(ctx, cfg, userRepo)
	if err != nil {
		logger.Error("bootstrap admin user error", slog.Any("error", err))
		return
	}
	if user != nil {
		logger.Info("admin user created", slog.Any("user_id", user.ID))
	}
}

func ensureAdminUser(ctx context.Context, cfg config.Config, userRepo *repo.UserRepo) (*models.User, error) {
	email := strings.TrimSpace(cfg.Bootstrap.AdminEmail)
	password := strings.TrimSpace(cfg.Bootstrap.AdminPassword)
	if email == "" || password == "" {
		return nil, nil
	}

	username := strings.TrimSpace(cfg.Bootstrap.AdminUsername)
	if username == "" {
		username = "admin"
	}

	hash, err := auth.HashPassword(password, cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{Email: email, Username: username, PasswordHash: hash, Role: models.AdminRole, CreatedAt: now, UpdatedAt: now}
	if err := userRepo.Create(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("create admin user: %w", err)
	}

	return user, nil
}

func isDatabaseEmpty(ctx context.Context, database *bun.DB) (bool, error) {
	tables := []string{"users"}
	for _, table := range tables {
		count, err := database.NewSelect().TableExpr(table).Count(ctx)
		if err != nil {
			return false, fmt.Errorf("count %s: %w", table, err)
		}
		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}
