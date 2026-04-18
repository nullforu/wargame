package repo

import (
	"context"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestScoreboardRepoLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user1 := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUserForTestUserScope(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user1.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, false, time.Now().Add(-time.Minute))
	createSubmission(t, env, blocked.ID, ch1.ID, true, time.Now().Add(-30*time.Second))

	leaderboard, err := scoreRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(leaderboard.Entries) != 2 {
		t.Fatalf("expected 2 leaderboard rows, got %d", len(leaderboard.Entries))
	}
	if leaderboard.Entries[0].UserID != user1.ID {
		t.Fatalf("unexpected first row: %+v", leaderboard.Entries[0])
	}
	if len(leaderboard.Challenges) != 2 {
		t.Fatalf("expected 2 challenges, got %d", len(leaderboard.Challenges))
	}

	since := time.Now().Add(-2*time.Minute - time.Second)
	rows, err := scoreRepo.TimelineSubmissions(context.Background(), &since)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}
	if len(rows) != 1 || rows[0].UserID != user1.ID {
		t.Fatalf("unexpected timeline rows: %+v", rows)
	}
}

func TestScoreboardRepoLeaderboardTieBreak(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user1 := createUserForTestUserScope(t, env, "a@example.com", "a", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "b@example.com", "b", "pass", models.UserRole)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, ch.ID, true, time.Now().UTC())

	rows, err := scoreRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(rows.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows.Entries))
	}
	if rows.Entries[0].UserID != user1.ID {
		t.Fatalf("expected lower id first, got %+v", rows.Entries)
	}
}
