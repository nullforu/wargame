package repo

import (
	"context"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestFixedPointsMapAndSolveCounts(t *testing.T) {
	env := setupRepoTest(t)

	user1 := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUserForTestUserScope(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	challenge := createChallenge(t, env, "Challenge", 500, "FLAG{TEST}", true)

	createSubmission(t, env, user1.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	counts, err := solveCountsByChallenge(context.Background(), env.db, nil)
	if err != nil {
		t.Fatalf("solveCountsByChallenge: %v", err)
	}

	if counts[challenge.ID] != 2 {
		t.Fatalf("expected solve count 2, got %d", counts[challenge.ID])
	}

	points, err := fixedPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("fixedPointsMap: %v", err)
	}
	if points[challenge.ID] != challenge.Points {
		t.Fatalf("expected fixed points %d, got %d", challenge.Points, points[challenge.ID])
	}
}

func TestDecayFactor(t *testing.T) {
	env := setupRepoTest(t)

	_ = createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	_ = createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	_ = createUserForTestUserScope(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	decay, err := decayFactor(context.Background(), env.db)
	if err != nil {
		t.Fatalf("decayFactor: %v", err)
	}
	if decay != 3 {
		t.Fatalf("expected decay factor 3, got %d", decay)
	}
}
