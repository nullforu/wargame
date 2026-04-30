package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wargame/internal/config"
	"wargame/internal/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func New(cfg config.DBConfig, appEnv string) (*bun.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	db := bun.NewDB(sqldb, pgdialect.New())
	if appEnv != "production" {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(false)))
	}

	return db, nil
}

func AutoMigrate(ctx context.Context, db *bun.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	modelsToCreate := []any{
		(*models.User)(nil),
		(*models.Affiliation)(nil),
		(*models.Challenge)(nil),
		(*models.Stack)(nil),
		(*models.Submission)(nil),
		(*models.ChallengeVote)(nil),
		(*models.Writeup)(nil),
		(*models.ChallengeCommentItem)(nil),
	}

	if err := createTables(ctx, db, modelsToCreate); err != nil {
		return err
	}

	return createIndexes(ctx, db)
}

func createTables(ctx context.Context, db *bun.DB, modelsToCreate []any) error {
	for _, m := range modelsToCreate {
		if _, err := db.NewCreateTable().Model(m).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("auto migrate create table %T: %w", m, err)
		}
	}

	return nil
}

func createIndexes(ctx context.Context, db *bun.DB) error {
	indexes := []struct {
		name  string
		query string
	}{
		{name: "idx_challenges_category", query: "CREATE INDEX IF NOT EXISTS idx_challenges_category ON challenges (category)"},
		{name: "idx_submissions_user", query: "CREATE INDEX IF NOT EXISTS idx_submissions_user ON submissions (user_id)"},
		{name: "idx_submissions_challenge", query: "CREATE INDEX IF NOT EXISTS idx_submissions_challenge ON submissions (challenge_id)"},
		{name: "idx_submissions_user_challenge", query: "CREATE INDEX IF NOT EXISTS idx_submissions_user_challenge ON submissions (user_id, challenge_id)"},
		{name: "idx_submissions_correct_time", query: "CREATE INDEX IF NOT EXISTS idx_submissions_correct_time ON submissions (correct, submitted_at) WHERE correct = true"},
		{name: "idx_challenge_votes_challenge", query: "CREATE INDEX IF NOT EXISTS idx_challenge_votes_challenge ON challenge_votes (challenge_id)"},
		{name: "idx_challenge_votes_challenge_level", query: "CREATE INDEX IF NOT EXISTS idx_challenge_votes_challenge_level ON challenge_votes (challenge_id, level)"},
		{name: "idx_challenge_votes_user_challenge", query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_challenge_votes_user_challenge ON challenge_votes (user_id, challenge_id)"},
		{name: "idx_stacks_user_id", query: "CREATE INDEX IF NOT EXISTS idx_stacks_user_id ON stacks (user_id)"},
		{name: "idx_stacks_user_challenge", query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_stacks_user_challenge ON stacks (user_id, challenge_id)"},
		{name: "idx_stacks_stack_id", query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_stacks_stack_id ON stacks (stack_id)"},
		{name: "idx_writeups_challenge_created", query: "CREATE INDEX IF NOT EXISTS idx_writeups_challenge_created ON writeups (challenge_id, created_at DESC, id DESC)"},
		{name: "idx_writeups_user_updated", query: "CREATE INDEX IF NOT EXISTS idx_writeups_user_updated ON writeups (user_id, updated_at DESC, id DESC)"},
		{name: "idx_writeups_user_challenge", query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_writeups_user_challenge ON writeups (user_id, challenge_id)"},
		{name: "idx_challenge_comments_challenge_created", query: "CREATE INDEX IF NOT EXISTS idx_challenge_comments_challenge_created ON challenge_comments (challenge_id, created_at DESC, id DESC)"},
		{name: "idx_challenge_comments_user_updated", query: "CREATE INDEX IF NOT EXISTS idx_challenge_comments_user_updated ON challenge_comments (user_id, updated_at DESC, id DESC)"},
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx.query); err != nil {
			return fmt.Errorf("auto migrate create index %s: %w", idx.name, err)
		}
	}

	return nil
}
