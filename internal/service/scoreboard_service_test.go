package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestAggregateUserTimeline(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	raw := []models.UserTimelineRow{
		{SubmittedAt: base.Add(1 * time.Minute), UserID: 1, Username: "a", Points: 50},
		{SubmittedAt: base.Add(5 * time.Minute), UserID: 1, Username: "a", Points: 100},
		{SubmittedAt: base.Add(11 * time.Minute), UserID: 2, Username: "b", Points: 25},
		{SubmittedAt: base.Add(10 * time.Minute), UserID: 1, Username: "a", Points: 10},
	}

	got := aggregateUserTimeline(raw)
	want := []models.TimelineSubmission{
		{Timestamp: base.Truncate(10 * time.Minute), UserID: 1, Username: "a", Points: 150, ChallengeCount: 2},
		{Timestamp: base.Add(10 * time.Minute), UserID: 1, Username: "a", Points: 10, ChallengeCount: 1},
		{Timestamp: base.Add(10 * time.Minute), UserID: 2, Username: "b", Points: 25, ChallengeCount: 1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected timeline: %+v", got)
	}
}

func TestScoreboardServiceLeaderboardAndTimeline(t *testing.T) {
	env := setupServiceTest(t)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	blocked := createUser(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, ch1.ID, true, base.Add(2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, base.Add(5*time.Minute))
	createSubmission(t, env, admin.ID, ch2.ID, true, base.Add(5*time.Minute))
	createSubmission(t, env, blocked.ID, ch1.ID, true, base.Add(6*time.Minute))

	board, err := env.scoreSvc.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(board.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(board.Entries))
	}

	userScores := map[int64]int{}
	for _, entry := range board.Entries {
		userScores[entry.UserID] = entry.Score
	}
	if userScores[user1.ID] == 0 || userScores[user2.ID] == 0 {
		t.Fatalf("expected positive scores for both users, got %+v", userScores)
	}
	if userScores[admin.ID] == 0 {
		t.Fatalf("expected positive score for admin, got %+v", userScores)
	}
	if _, ok := userScores[blocked.ID]; ok {
		t.Fatalf("blocked user must be excluded from leaderboard")
	}

	timeline, err := env.scoreSvc.UserTimeline(context.Background(), nil)
	if err != nil {
		t.Fatalf("UserTimeline: %v", err)
	}
	if len(timeline) != 3 {
		t.Fatalf("expected 3 timeline rows, got %d", len(timeline))
	}
}
