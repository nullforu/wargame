package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
	"wargame/internal/stack"
)

func createStack(t *testing.T, env repoEnv, userID, challengeID int64, stackID string, createdAt time.Time) *models.Stack {
	t.Helper()
	stackModel := &models.Stack{
		UserID:      userID,
		ChallengeID: challengeID,
		StackID:     stackID,
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := env.stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}
	return stackModel
}

func TestStackRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "stacker@example.com", "stacker", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Stacked", 100, "flag{1}", true)

	now := time.Now().UTC()
	stackModel := createStack(t, env, user.ID, challenge.ID, "stack-1", now)

	got, err := env.stackRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByUserAndChallenge: %v", err)
	}
	if got.StackID != stackModel.StackID {
		t.Fatalf("unexpected stack: %+v", got)
	}

	count, err := env.stackRepo.CountByUser(context.Background(), user.ID)
	if err != nil || count != 1 {
		t.Fatalf("CountByUser=%d err=%v", count, err)
	}

	if err := env.stackRepo.DeleteByUserAndChallenge(context.Background(), user.ID, challenge.ID); err != nil {
		t.Fatalf("DeleteByUserAndChallenge: %v", err)
	}
	if _, err := env.stackRepo.GetByStackID(context.Background(), stackModel.StackID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStackRepoListAndCounts(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "count@example.com", "count", "pass", models.UserRole)
	challenge := createChallenge(t, env, "CountCh", 100, "flag{count}", true)
	terminalChallenge := createChallenge(t, env, "CountChTerm", 100, "flag{count2}", true)

	now := time.Now().UTC()
	createStack(t, env, user.ID, challenge.ID, "stack-running", now)
	terminal := &models.Stack{
		UserID:      user.ID,
		ChallengeID: terminalChallenge.ID,
		StackID:     "stack-stopped",
		Status:      "stopped",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
	}
	if err := env.stackRepo.Create(context.Background(), terminal); err != nil {
		t.Fatalf("create terminal stack: %v", err)
	}

	stacks, err := env.stackRepo.ListByUser(context.Background(), user.ID)
	if err != nil || len(stacks) != 2 {
		t.Fatalf("ListByUser len=%d err=%v", len(stacks), err)
	}

	activeCount, err := env.stackRepo.CountByUserExcludingStatuses(context.Background(), user.ID, []string{"stopped"})
	if err != nil || activeCount != 1 {
		t.Fatalf("CountByUserExcludingStatuses=%d err=%v", activeCount, err)
	}
}

func TestStackRepoListAdmin(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "admin-stack@example.com", "adminstack", "pass", models.UserRole)
	challenge := createChallenge(t, env, "AdminStack", 200, "flag{admin}", true)
	createStack(t, env, user.ID, challenge.ID, "stack-admin", time.Now().UTC())

	stacks, err := env.stackRepo.ListAdmin(context.Background())
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}
	if stacks[0].Username != user.Username || stacks[0].ChallengeTitle != challenge.Title {
		t.Fatalf("unexpected admin row: %+v", stacks[0])
	}
}

func TestStackRepoListAllAndGetByStackID(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "all@example.com", "all", "pass", models.UserRole)
	challenge1 := createChallenge(t, env, "All1", 100, "flag{a1}", true)
	challenge2 := createChallenge(t, env, "All2", 100, "flag{a2}", true)

	now := time.Now().UTC()
	createStack(t, env, user.ID, challenge1.ID, "stack-old", now.Add(-time.Minute))
	created := createStack(t, env, user.ID, challenge2.ID, "stack-new", now)

	rows, err := env.stackRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(rows) != 2 || rows[0].StackID != "stack-new" {
		t.Fatalf("unexpected list order: %+v", rows)
	}

	got, err := env.stackRepo.GetByStackID(context.Background(), created.StackID)
	if err != nil {
		t.Fatalf("GetByStackID: %v", err)
	}
	if got.UserID != user.ID || got.ChallengeID != challenge2.ID {
		t.Fatalf("unexpected stack row: %+v", got)
	}
}

func TestStackRepoNotFoundAndUpdateDelete(t *testing.T) {
	env := setupRepoTest(t)

	if _, err := env.stackRepo.GetByStackID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	user := createUserForTestUserScope(t, env, "upd@example.com", "upd", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Upd", 100, "flag{upd}", true)
	created := createStack(t, env, user.ID, challenge.ID, "stack-upd", time.Now().UTC())

	created.Status = "running"
	if err := env.stackRepo.Update(context.Background(), created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := env.stackRepo.Delete(context.Background(), created); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := env.stackRepo.GetByStackID(context.Background(), "stack-upd"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted stack, got %v", err)
	}
}
