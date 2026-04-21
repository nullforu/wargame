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

	users, pagination, err := env.userSvc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("unexpected list: %+v", users)
	}
	if pagination.Page != 1 || pagination.PageSize != DefaultPageSize {
		t.Fatalf("unexpected pagination: %+v", pagination)
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

func TestUserServiceSearchAndPagination(t *testing.T) {
	env := setupServiceTest(t)
	_ = createUser(t, env, "alpha@example.com", "alpha-user", "pass", models.UserRole)
	_ = createUser(t, env, "beta@example.com", "beta-user", "pass", models.UserRole)

	rows, pagination, err := env.userSvc.Search(context.Background(), "user", 1, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if pagination.TotalCount != 2 || pagination.TotalPages != 2 || !pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	if _, _, err := env.userSvc.Search(context.Background(), " ", 1, 10); err == nil {
		t.Fatalf("expected required query validation error")
	}

	if _, _, err := env.userSvc.ListPage(context.Background(), 1, MaxPageSize+1); err == nil {
		t.Fatalf("expected max page size validation error")
	}
}
