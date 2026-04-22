package repo

import (
	"context"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestChallengeVoteRepoRepresentativeAndCounts(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewChallengeVoteRepo(env.db)
	ch := createChallenge(t, env, "vote", 100, "FLAG{V}", true)
	u1 := createUserForTestUserScope(t, env, "v1@example.com", "v1", "pass", models.UserRole)
	u2 := createUserForTestUserScope(t, env, "v2@example.com", "v2", "pass", models.UserRole)
	u3 := createUserForTestUserScope(t, env, "v3@example.com", "v3", "pass", models.UserRole)

	now := time.Now().UTC()
	mustUpsertVote(t, repo, ch.ID, u1.ID, 3, now.Add(-3*time.Minute))
	mustUpsertVote(t, repo, ch.ID, u2.ID, 7, now.Add(-2*time.Minute))
	mustUpsertVote(t, repo, ch.ID, u3.ID, 7, now.Add(-time.Minute))

	level, err := repo.RepresentativeLevelByChallengeID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("RepresentativeLevelByChallengeID: %v", err)
	}

	if level != 7 {
		t.Fatalf("expected level 7, got %d", level)
	}

	counts, err := repo.VoteCountsByChallengeID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("VoteCountsByChallengeID: %v", err)
	}

	if len(counts) != 2 {
		t.Fatalf("expected 2 count rows, got %+v", counts)
	}

	if counts[0].Level != 3 || counts[1].Level != 7 {
		t.Fatalf("expected counts ordered by level asc, got %+v", counts)
	}
}

func TestChallengeVoteRepoTieBreakAndPagination(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewChallengeVoteRepo(env.db)
	ch := createChallenge(t, env, "vote-paged", 100, "FLAG{VP}", true)
	u1 := createUserForTestUserScope(t, env, "pv1@example.com", "pv1", "pass", models.UserRole)
	u2 := createUserForTestUserScope(t, env, "pv2@example.com", "pv2", "pass", models.UserRole)

	now := time.Now().UTC()
	mustUpsertVote(t, repo, ch.ID, u1.ID, 6, now.Add(-2*time.Minute))
	mustUpsertVote(t, repo, ch.ID, u2.ID, 7, now.Add(-time.Minute))
	mustUpsertVote(t, repo, ch.ID, u1.ID, 6, now)

	level, err := repo.RepresentativeLevelByChallengeID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("RepresentativeLevelByChallengeID tie: %v", err)
	}

	if level != 6 {
		t.Fatalf("expected level 6 by latest tie-break, got %d", level)
	}

	rows, total, err := repo.VotesByChallengePage(context.Background(), ch.ID, 1, 1)
	if err != nil {
		t.Fatalf("VotesByChallengePage: %v", err)
	}

	if total != 2 || len(rows) != 1 || rows[0].UserID != u1.ID {
		t.Fatalf("unexpected page1 rows=%+v total=%d", rows, total)
	}
}

func TestChallengeVoteRepoRepresentativeLevelsByChallengeIDs(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewChallengeVoteRepo(env.db)

	ch1 := createChallenge(t, env, "multi-1", 100, "FLAG{M1}", true)
	ch2 := createChallenge(t, env, "multi-2", 100, "FLAG{M2}", true)
	u1 := createUserForTestUserScope(t, env, "m1@example.com", "m1", "pass", models.UserRole)
	u2 := createUserForTestUserScope(t, env, "m2@example.com", "m2", "pass", models.UserRole)

	now := time.Now().UTC()
	mustUpsertVote(t, repo, ch1.ID, u1.ID, 4, now.Add(-2*time.Minute))
	mustUpsertVote(t, repo, ch1.ID, u2.ID, 4, now.Add(-time.Minute))
	mustUpsertVote(t, repo, ch2.ID, u1.ID, 9, now)

	levels, err := repo.RepresentativeLevelsByChallengeIDs(context.Background(), []int64{ch1.ID, ch2.ID, 999999})
	if err != nil {
		t.Fatalf("RepresentativeLevelsByChallengeIDs: %v", err)
	}

	if levels[ch1.ID] != 4 || levels[ch2.ID] != 9 || levels[999999] != models.UnknownLevel {
		t.Fatalf("unexpected representative levels: %+v", levels)
	}

	empty, err := repo.RepresentativeLevelsByChallengeIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("RepresentativeLevelsByChallengeIDs empty: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected empty map for empty ids, got %+v", empty)
	}
}

func TestChallengeVoteRepoVoteCountsByChallengeIDs(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewChallengeVoteRepo(env.db)

	ch1 := createChallenge(t, env, "count-1", 100, "FLAG{C1}", true)
	ch2 := createChallenge(t, env, "count-2", 100, "FLAG{C2}", true)
	u1 := createUserForTestUserScope(t, env, "c1@example.com", "c1", "pass", models.UserRole)
	u2 := createUserForTestUserScope(t, env, "c2@example.com", "c2", "pass", models.UserRole)
	u3 := createUserForTestUserScope(t, env, "c3@example.com", "c3", "pass", models.UserRole)

	now := time.Now().UTC()
	mustUpsertVote(t, repo, ch1.ID, u1.ID, 6, now.Add(-3*time.Minute))
	mustUpsertVote(t, repo, ch1.ID, u2.ID, 6, now.Add(-2*time.Minute))
	mustUpsertVote(t, repo, ch1.ID, u3.ID, 7, now.Add(-time.Minute))

	counts, err := repo.VoteCountsByChallengeIDs(context.Background(), []int64{ch1.ID, ch2.ID})
	if err != nil {
		t.Fatalf("VoteCountsByChallengeIDs: %v", err)
	}

	if len(counts[ch2.ID]) != 0 {
		t.Fatalf("expected empty vote counts for no-vote challenge, got %+v", counts[ch2.ID])
	}

	if len(counts[ch1.ID]) != 2 {
		t.Fatalf("expected 2 grouped rows, got %+v", counts[ch1.ID])
	}

	if counts[ch1.ID][0].Level != 6 || counts[ch1.ID][1].Level != 7 {
		t.Fatalf("expected per-challenge counts ordered by level asc, got %+v", counts[ch1.ID])
	}

	empty, err := repo.VoteCountsByChallengeIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("VoteCountsByChallengeIDs empty: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected empty map for empty ids, got %+v", empty)
	}
}

func TestChallengeVoteRepoVoteLevelByUserAndChallengeID(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewChallengeVoteRepo(env.db)
	ch := createChallenge(t, env, "my-vote", 100, "FLAG{MV}", true)
	user := createUserForTestUserScope(t, env, "mv@example.com", "mv", "pass", models.UserRole)

	now := time.Now().UTC()
	mustUpsertVote(t, repo, ch.ID, user.ID, 8, now)

	level, err := repo.VoteLevelByUserAndChallengeID(context.Background(), user.ID, ch.ID)
	if err != nil {
		t.Fatalf("VoteLevelByUserAndChallengeID: %v", err)
	}

	if level == nil || *level != 8 {
		t.Fatalf("expected level=8, got %+v", level)
	}

	none, err := repo.VoteLevelByUserAndChallengeID(context.Background(), user.ID, 999999)
	if err != nil {
		t.Fatalf("VoteLevelByUserAndChallengeID none: %v", err)
	}

	if none != nil {
		t.Fatalf("expected nil for missing vote, got %+v", none)
	}
}

func mustUpsertVote(t *testing.T, repo *ChallengeVoteRepo, challengeID, userID int64, level int, ts time.Time) {
	t.Helper()
	if err := repo.Upsert(context.Background(), &models.ChallengeVote{
		ChallengeID: challengeID,
		UserID:      userID,
		Level:       level,
		CreatedAt:   ts,
		UpdatedAt:   ts,
	}); err != nil {
		t.Fatalf("upsert vote: %v", err)
	}
}
