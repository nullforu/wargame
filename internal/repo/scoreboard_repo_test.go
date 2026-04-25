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
	admin := createUserForTestUserScope(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	blocked := createUserForTestUserScope(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user1.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, admin.ID, ch2.ID, true, time.Now().Add(-90*time.Second))
	createSubmission(t, env, user2.ID, ch2.ID, false, time.Now().Add(-time.Minute))
	createSubmission(t, env, blocked.ID, ch1.ID, true, time.Now().Add(-30*time.Second))

	leaderboard, totalCount, err := scoreRepo.Leaderboard(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if totalCount != 3 {
		t.Fatalf("expected total count 3, got %d", totalCount)
	}
	if len(leaderboard.Entries) != 3 {
		t.Fatalf("expected 3 leaderboard rows, got %d", len(leaderboard.Entries))
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
	if len(rows) != 2 {
		t.Fatalf("unexpected timeline row count: %+v", rows)
	}
	if rows[0].UserID != user1.ID || rows[1].UserID != admin.ID {
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

	rows, totalCount, err := scoreRepo.Leaderboard(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if totalCount != 2 {
		t.Fatalf("expected total count 2, got %d", totalCount)
	}
	if len(rows.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows.Entries))
	}
	if rows.Entries[0].UserID != user1.ID {
		t.Fatalf("expected lower id first, got %+v", rows.Entries)
	}
}

func TestScoreboardRepoLeaderboardPagination(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user1 := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	user3 := createUserForTestUserScope(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, ch.ID, true, time.Now().UTC())
	_ = user3

	page1, totalCount, err := scoreRepo.Leaderboard(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("Leaderboard page1: %v", err)
	}
	if totalCount != 3 {
		t.Fatalf("expected total count 3, got %d", totalCount)
	}
	if len(page1.Entries) != 2 {
		t.Fatalf("expected 2 rows in page1, got %d", len(page1.Entries))
	}

	page2, _, err := scoreRepo.Leaderboard(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("Leaderboard page2: %v", err)
	}
	if len(page2.Entries) != 1 {
		t.Fatalf("expected 1 row in page2, got %d", len(page2.Entries))
	}
	if page2.Entries[0].UserID != user3.ID {
		t.Fatalf("unexpected page2 row: %+v", page2.Entries[0])
	}
}

func TestScoreboardRepoRankings(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)
	affiliationRepo := NewAffiliationRepo(env.db)

	affA := &models.Affiliation{Name: "Team A", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	affB := &models.Affiliation{Name: "Team B", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := affiliationRepo.Create(context.Background(), affA); err != nil {
		t.Fatalf("create affiliation A: %v", err)
	}

	if err := affiliationRepo.Create(context.Background(), affB); err != nil {
		t.Fatalf("create affiliation B: %v", err)
	}

	user1 := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	user3 := createUserForTestUserScope(t, env, "u3@example.com", "u3", "pass", models.UserRole)
	user1.AffiliationID = &affA.ID
	user2.AffiliationID = &affA.ID
	user3.AffiliationID = &affB.ID
	if err := env.userRepo.Update(context.Background(), user1); err != nil {
		t.Fatalf("update user1 affiliation: %v", err)
	}

	if err := env.userRepo.Update(context.Background(), user2); err != nil {
		t.Fatalf("update user2 affiliation: %v", err)
	}

	if err := env.userRepo.Update(context.Background(), user3); err != nil {
		t.Fatalf("update user3 affiliation: %v", err)
	}

	ch1 := createChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 200, "FLAG{2}", true)
	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, ch1.ID, true, time.Now().UTC())
	createSubmission(t, env, user3.ID, ch2.ID, true, time.Now().UTC())

	userRows, totalUsers, err := scoreRepo.UserRanking(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("user ranking: %v", err)
	}

	if totalUsers != 3 || len(userRows) != 3 {
		t.Fatalf("unexpected user ranking rows: total=%d rows=%d", totalUsers, len(userRows))
	}

	if userRows[0].UserID != user3.ID || userRows[0].Score != 200 || userRows[0].SolvedCount != 1 {
		t.Fatalf("unexpected top user row: %+v", userRows[0])
	}

	affRows, totalAffiliations, err := scoreRepo.AffiliationRanking(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("affiliation ranking: %v", err)
	}

	if totalAffiliations != 2 || len(affRows) != 2 {
		t.Fatalf("unexpected affiliation ranking rows: total=%d rows=%d", totalAffiliations, len(affRows))
	}

	if affRows[0].AffiliationID != affA.ID || affRows[0].Score != 200 || affRows[0].UserCount != 2 {
		t.Fatalf("unexpected top affiliation row: %+v", affRows[0])
	}

	affUserRows, totalAffUsers, err := scoreRepo.AffiliationUserRanking(context.Background(), affA.ID, 1, 20)
	if err != nil {
		t.Fatalf("affiliation user ranking: %v", err)
	}

	if totalAffUsers != 2 || len(affUserRows) != 2 {
		t.Fatalf("unexpected affiliation user ranking rows: total=%d rows=%d", totalAffUsers, len(affUserRows))
	}

	for _, row := range affUserRows {
		if row.AffiliationID == nil || *row.AffiliationID != affA.ID {
			t.Fatalf("unexpected affiliation in row: %+v", row)
		}
	}
}
