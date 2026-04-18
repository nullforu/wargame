package repo

import (
	"context"
	"errors"
	"testing"
)

func TestUserRepoCreateGetByIDUpdate(t *testing.T) {
	env := setupRepoTest(t)
	user := createUser(t, env, "update@example.com", "update-user", "pass", "user")

	got, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("unexpected user: %+v", got)
	}

	got.Username = "updated-name"
	if err := env.userRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("update user: %v", err)
	}

	updated, err := env.userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if updated.Username != "updated-name" {
		t.Fatalf("expected updated username, got %q", updated.Username)
	}
}

func TestUserRepoGetByEmailOrUsername(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "lookup@example.com", "lookup-user", "pass", "user")

	got, err := env.userRepo.GetByEmailOrUsername(context.Background(), "lookup@example.com", "nope")
	if err != nil {
		t.Fatalf("get by email or username (email): %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user by email, got %+v", got)
	}

	got, err = env.userRepo.GetByEmailOrUsername(context.Background(), "missing@example.com", "lookup-user")
	if err != nil {
		t.Fatalf("get by email or username (username): %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user by username, got %+v", got)
	}
}

func TestUserRepoList(t *testing.T) {
	env := setupRepoTest(t)
	userA := createUserForTestUserScope(t, env, "a@example.com", "user-a", "pass", "user")
	_ = createUserForTestUserScope(t, env, "b@example.com", "user-b", "pass", "user")

	allUsers, err := env.userRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(allUsers) != 2 {
		t.Fatalf("expected 2 users, got %d", len(allUsers))
	}
	if allUsers[0].ID != userA.ID {
		t.Fatalf("expected ordered users by id, got %+v", allUsers)
	}
}

func TestUserRepoNotFoundCases(t *testing.T) {
	env := setupRepoTest(t)

	if _, err := env.userRepo.GetByID(context.Background(), 123456); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByID not found, got %v", err)
	}
	if _, err := env.userRepo.GetByEmail(context.Background(), "missing@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByEmail not found, got %v", err)
	}
	if _, err := env.userRepo.GetByEmailOrUsername(context.Background(), "missing@example.com", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected GetByEmailOrUsername not found, got %v", err)
	}
}
