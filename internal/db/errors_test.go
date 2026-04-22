package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestIsUniqueViolation(t *testing.T) {
	if IsUniqueViolation(nil) {
		t.Error("expected IsUniqueViolation to return false for nil error")
	}

	genericErr := errors.New("some error")
	if IsUniqueViolation(genericErr) {
		t.Error("expected IsUniqueViolation to return false for generic error")
	}

	db := setupDBTest(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE challenge_votes, submissions, stacks, challenges, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		Email:        "dup@example.com",
		Username:     "dup-user-1",
		PasswordHash: "hash",
		Role:         models.UserRole,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("insert first user: %v", err)
	}

	duplicate := &models.User{
		Email:        "dup@example.com",
		Username:     "dup-user-2",
		PasswordHash: "hash",
		Role:         models.UserRole,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err := db.NewInsert().Model(duplicate).Exec(ctx)
	if err == nil {
		t.Fatal("expected duplicate insert error")
	}

	if !IsUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}
