package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"
	"wargame/internal/models"
)

func TestScoreboard(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.LeaderboardResponse
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].UserID != user2.ID || resp.Entries[0].Score != 300 {
		t.Fatalf("unexpected first row: %+v", resp.Entries[0])
	}
}

func TestScoreboardTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))
	createSubmission(t, env, user1.ID, challenge2.ID, true, base.Add(16*time.Minute))
	createSubmission(t, env, blocked.ID, challenge2.ID, true, base.Add(20*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			Timestamp      time.Time `json:"timestamp"`
			UserID         int64     `json:"user_id"`
			Username       string    `json:"username"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 3 {
		t.Fatalf("expected 3 submissions, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].UserID != 1 || resp.Submissions[0].Points != 100 || resp.Submissions[0].ChallengeCount != 1 {
		t.Fatalf("unexpected first submission: %+v", resp.Submissions[0])
	}

	if resp.Submissions[1].UserID != 2 || resp.Submissions[1].Points != 200 || resp.Submissions[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second submission: %+v", resp.Submissions[1])
	}

	if resp.Submissions[2].UserID != 1 || resp.Submissions[2].Points != 200 || resp.Submissions[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third submission: %+v", resp.Submissions[2])
	}
}

func TestScoreboardDynamicScoring(t *testing.T) {
	env := setupTest(t, testCfg)
	userA := createUser(t, env, "usera@example.com", "usera", "pass123", models.UserRole)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass123", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass123", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}

	challenge := createChallenge(t, env, "Dynamic", 500, "flag{dynamic}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	createSubmission(t, env, userA.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, userSolo.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.LeaderboardResponse
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].Score != 100 || resp.Entries[1].Score != 100 {
		t.Fatalf("expected dynamic scores 100, got %+v", resp.Entries)
	}
}
