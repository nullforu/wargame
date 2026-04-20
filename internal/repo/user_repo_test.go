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
	userB := createUserForTestUserScope(t, env, "b@example.com", "user-b", "pass", "user")
	_ = createUserForTestUserScope(t, env, "c@example.com", "user-c", "pass", "user")

	firstPage, totalCount, err := env.userRepo.List(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if totalCount != 3 {
		t.Fatalf("expected total_count 3, got %d", totalCount)
	}

	if len(firstPage) != 2 {
		t.Fatalf("expected 2 users in first page, got %d", len(firstPage))
	}

	if firstPage[0].ID != userA.ID || firstPage[1].ID != userB.ID {
		t.Fatalf("expected ordered users by id, got %+v", firstPage)
	}
}

func TestUserRepoSearch(t *testing.T) {
	env := setupRepoTest(t)
	_ = createUserForTestUserScope(t, env, "alpha@example.com", "alpha-user", "pass", "user")
	_ = createUserForTestUserScope(t, env, "beta@example.com", "beta-user", "pass", "user")
	_ = createUserForTestUserScope(t, env, "gamma@example.com", "alpha-second", "pass", "user")

	rows, totalCount, err := env.userRepo.Search(context.Background(), "alpha", 1, 10)
	if err != nil {
		t.Fatalf("search users: %v", err)
	}

	if totalCount != 2 {
		t.Fatalf("expected total_count 2, got %d", totalCount)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 matched users, got %d", len(rows))
	}

	pagedRows, pagedTotalCount, err := env.userRepo.Search(context.Background(), "user", 2, 1)
	if err != nil {
		t.Fatalf("search users paged: %v", err)
	}

	if pagedTotalCount != 2 {
		t.Fatalf("expected paged total_count 2, got %d", pagedTotalCount)
	}

	if len(pagedRows) != 1 || pagedRows[0].Username != "beta-user" {
		t.Fatalf("unexpected paged rows: %+v", pagedRows)
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
