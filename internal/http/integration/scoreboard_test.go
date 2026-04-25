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
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
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
	createSubmission(t, env, admin.ID, challenge2.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard?page=1&page_size=2", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Challenges []models.LeaderboardChallenge `json:"challenges"`
		Entries    []models.LeaderboardEntry     `json:"entries"`
		Pagination models.Pagination             `json:"pagination"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 rows on first page, got %d", len(resp.Entries))
	}

	if resp.Entries[0].UserID != user2.ID || resp.Entries[0].Score != 300 {
		t.Fatalf("unexpected first row: %+v", resp.Entries[0])
	}
	if resp.Pagination.TotalCount != 3 || !resp.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", resp.Pagination)
	}
}

func TestScoreboardTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
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
	createSubmission(t, env, admin.ID, challenge1.ID, true, base.Add(18*time.Minute))
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

	if len(resp.Submissions) != 4 {
		t.Fatalf("expected 4 submissions, got %d", len(resp.Submissions))
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
	if resp.Submissions[3].UserID != admin.ID || resp.Submissions[3].Points != 100 || resp.Submissions[3].ChallengeCount != 1 {
		t.Fatalf("unexpected fourth submission: %+v", resp.Submissions[3])
	}
}

func TestScoreboardFixedScoring(t *testing.T) {
	env := setupTest(t, testCfg)
	userA := createUser(t, env, "usera@example.com", "usera", "pass123", models.UserRole)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass123", models.UserRole)
	admin := createUser(t, env, "admin@example.com", "admin", "pass123", models.AdminRole)
	blocked := createUser(t, env, "blocked@example.com", models.BlockedRole, "pass123", models.UserRole)
	blocked.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), blocked); err != nil {
		t.Fatalf("block user: %v", err)
	}

	challenge := createChallenge(t, env, "Fixed", 500, "flag{fixed}", true)

	createSubmission(t, env, userA.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, userSolo.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, admin.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard?page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Challenges []models.LeaderboardChallenge `json:"challenges"`
		Entries    []models.LeaderboardEntry     `json:"entries"`
		Pagination models.Pagination             `json:"pagination"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Entries) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(resp.Entries))
	}

	if resp.Entries[0].Score != 500 || resp.Entries[1].Score != 500 {
		t.Fatalf("expected fixed scores 500, got %+v", resp.Entries)
	}
}

func TestScoreboardLeaderboardInvalidPagination(t *testing.T) {
	env := setupTest(t, testCfg)
	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard?page=bad&page_size=10", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRankingEndpoints(t *testing.T) {
	env := setupTest(t, testCfg)
	affA := createAffiliation(t, env, "Team A")
	affB := createAffiliation(t, env, "Team B")

	user1 := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	user3 := createUser(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	user1Bio := "user1 bio"
	user2Bio := "user2 bio"
	user3Bio := "user3 bio"
	user1.AffiliationID = &affA.ID
	user2.AffiliationID = &affA.ID
	user3.AffiliationID = &affB.ID
	user1.Bio = &user1Bio
	user2.Bio = &user2Bio
	user3.Bio = &user3Bio
	if err := env.userRepo.Update(context.Background(), user1); err != nil {
		t.Fatalf("update user1: %v", err)
	}

	if err := env.userRepo.Update(context.Background(), user2); err != nil {
		t.Fatalf("update user2: %v", err)
	}

	if err := env.userRepo.Update(context.Background(), user3); err != nil {
		t.Fatalf("update user3: %v", err)
	}

	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)
	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, ch1.ID, true, time.Now().UTC())
	createSubmission(t, env, user3.ID, ch2.ID, true, time.Now().UTC())

	userRec := doRequest(t, env.router, http.MethodGet, "/api/rankings/users?page=1&page_size=10", nil, nil)
	if userRec.Code != http.StatusOK {
		t.Fatalf("ranking users status %d: %s", userRec.Code, userRec.Body.String())
	}

	var userResp struct {
		Entries []struct {
			UserID      int64   `json:"user_id"`
			Score       int     `json:"score"`
			SolvedCount int     `json:"solved_count"`
			Bio         *string `json:"bio"`
		} `json:"entries"`
	}
	decodeJSON(t, userRec, &userResp)
	if len(userResp.Entries) != 3 || userResp.Entries[0].UserID != user3.ID || userResp.Entries[0].Score != 200 || userResp.Entries[0].SolvedCount != 1 {
		t.Fatalf("unexpected users ranking: %+v", userResp.Entries)
	}
	if userResp.Entries[0].Bio == nil || *userResp.Entries[0].Bio != user3Bio {
		t.Fatalf("unexpected user bio in ranking: %+v", userResp.Entries[0])
	}

	affRec := doRequest(t, env.router, http.MethodGet, "/api/rankings/affiliations?page=1&page_size=10", nil, nil)
	if affRec.Code != http.StatusOK {
		t.Fatalf("ranking affiliations status %d: %s", affRec.Code, affRec.Body.String())
	}

	var affResp struct {
		Entries []struct {
			AffiliationID int64 `json:"affiliation_id"`
			Score         int   `json:"score"`
			UserCount     int   `json:"user_count"`
		} `json:"entries"`
	}
	decodeJSON(t, affRec, &affResp)
	if len(affResp.Entries) != 2 || affResp.Entries[0].AffiliationID != affA.ID || affResp.Entries[0].Score != 200 || affResp.Entries[0].UserCount != 2 {
		t.Fatalf("unexpected affiliations ranking: %+v", affResp.Entries)
	}

	affUsersRec := doRequest(t, env.router, http.MethodGet, "/api/rankings/affiliations/"+itoa(affA.ID)+"/users?page=1&page_size=10", nil, nil)
	if affUsersRec.Code != http.StatusOK {
		t.Fatalf("ranking affiliation users status %d: %s", affUsersRec.Code, affUsersRec.Body.String())
	}

	var affUsersResp struct {
		Entries []struct {
			UserID int64   `json:"user_id"`
			Bio    *string `json:"bio"`
		} `json:"entries"`
	}
	decodeJSON(t, affUsersRec, &affUsersResp)
	if len(affUsersResp.Entries) != 2 {
		t.Fatalf("unexpected affiliation users ranking entries: %+v", affUsersResp.Entries)
	}
	if affUsersResp.Entries[0].Bio == nil {
		t.Fatalf("expected bio in affiliation users ranking entries: %+v", affUsersResp.Entries)
	}
}
