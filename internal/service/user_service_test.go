package service

import (
	"context"
	"errors"
	"testing"

	"wargame/internal/models"
)

func TestUserServiceGetByIDListUpdateProfile(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user", "pass", models.UserRole)

	got, err := env.userSvc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("unexpected user: %+v", got)
	}

	users, err := env.userSvc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("unexpected list: %+v", users)
	}

	newName := "newname"
	updated, err := env.userSvc.UpdateProfile(context.Background(), user.ID, &newName)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if updated.Username != newName {
		t.Fatalf("expected username %q, got %q", newName, updated.Username)
	}
}

func TestUserServiceBlockUnblock(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "block@example.com", "block", "pass", models.UserRole)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	blocked, err := env.userSvc.BlockUser(context.Background(), user.ID, "policy")
	if err != nil {
		t.Fatalf("BlockUser: %v", err)
	}
	if blocked.Role != models.BlockedRole {
		t.Fatalf("expected blocked role, got %s", blocked.Role)
	}

	if _, err := env.userSvc.BlockUser(context.Background(), admin.ID, "policy"); err == nil {
		t.Fatalf("expected admin block error")
	}

	unblocked, err := env.userSvc.UnblockUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UnblockUser: %v", err)
	}
	if unblocked.Role != models.UserRole {
		t.Fatalf("expected user role after unblock, got %s", unblocked.Role)
	}
}

func TestUserServiceValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.userSvc.GetByID(context.Background(), 0); err == nil {
		t.Fatalf("expected validation error")
	}
	if _, err := env.userSvc.GetByID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := env.userSvc.BlockUser(context.Background(), 1, " "); err == nil {
		t.Fatalf("expected empty reason validation error")
	}
	if _, err := env.userSvc.UnblockUser(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
