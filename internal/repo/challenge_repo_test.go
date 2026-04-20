package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/db"
	"wargame/internal/models"

	"github.com/uptrace/bun"
)

func TestChallengeRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	ch := createChallenge(t, env, "challenge", 100, "FLAG{1}", true)

	got, err := env.challengeRepo.GetByID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Title != ch.Title {
		t.Fatalf("unexpected title: %s", got.Title)
	}

	list, totalCount, err := env.challengeRepo.ListActive(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}

	if totalCount != 1 {
		t.Fatalf("expected total_count 1, got %d", totalCount)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(list))
	}

	got.Title = "updated"
	if err := env.challengeRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := env.challengeRepo.GetByID(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("GetByID updated: %v", err)
	}

	if updated.Title != "updated" {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}

	if err := env.challengeRepo.Delete(context.Background(), updated); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := env.challengeRepo.GetByID(context.Background(), ch.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChallengeRepoListActiveAndSearchPagination(t *testing.T) {
	env := setupRepoTest(t)
	_ = createChallenge(t, env, "Web Warmup", 100, "FLAG{1}", true)
	_ = createChallenge(t, env, "Web Advanced", 200, "FLAG{2}", true)
	_ = createChallenge(t, env, "Crypto One", 150, "FLAG{3}", true)

	pageRows, totalCount, err := env.challengeRepo.ListActive(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("ListActive page: %v", err)
	}

	if totalCount != 3 {
		t.Fatalf("expected total_count 3, got %d", totalCount)
	}

	if len(pageRows) != 1 || pageRows[0].Title != "Crypto One" {
		t.Fatalf("unexpected page rows: %+v", pageRows)
	}

	searchRows, searchCount, err := env.challengeRepo.SearchActive(context.Background(), "Web", 1, 10)
	if err != nil {
		t.Fatalf("SearchActive: %v", err)
	}

	if searchCount != 2 {
		t.Fatalf("expected search total_count 2, got %d", searchCount)
	}

	if len(searchRows) != 2 {
		t.Fatalf("expected 2 search rows, got %d", len(searchRows))
	}
}

func TestChallengeRepoListActiveFiltered(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "solver@example.com", "solver", "pass", models.UserRole)

	web := createChallenge(t, env, "Web Warmup", 300, "FLAG{1}", true)
	web.Level = 3
	web.Category = "Web"
	if err := env.challengeRepo.Update(context.Background(), web); err != nil {
		t.Fatalf("update web: %v", err)
	}

	crypto := createChallenge(t, env, "Crypto Warmup", 700, "FLAG{2}", true)
	crypto.Level = 7
	crypto.Category = "Crypto"
	if err := env.challengeRepo.Update(context.Background(), crypto); err != nil {
		t.Fatalf("update crypto: %v", err)
	}

	createSubmission(t, env, user.ID, web.ID, true, time.Now().UTC())

	level := 3
	solved := true
	userID := user.ID
	rows, total, err := env.challengeRepo.ListActiveFiltered(context.Background(), ChallengeListFilter{
		Query:          "Warmup",
		Category:       "Web",
		Level:          &level,
		Solved:         &solved,
		SolvedByUserID: &userID,
	}, 1, 20)
	if err != nil {
		t.Fatalf("ListActiveFiltered: %v", err)
	}

	if total != 1 || len(rows) != 1 || rows[0].ID != web.ID {
		t.Fatalf("unexpected filtered rows: total=%d rows=%+v", total, rows)
	}
}

func TestChallengeRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.challengeRepo.GetByID(context.Background(), 123)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChallengeRepoPointsAndSolveCounts(t *testing.T) {
	env := setupRepoTest(t)

	user1 := createUserForTestUserScope(t, env, "usera@example.com", "usera", "pass", models.UserRole)
	userSolo := createUserForTestUserScope(t, env, "solo@example.com", "solo", "pass", models.UserRole)

	challenge := createChallenge(t, env, "Fixed", 500, "FLAG{DYN}", true)

	other := createChallenge(t, env, "Static", 200, "FLAG{STATIC}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, challenge.ID, true, now.Add(-time.Minute))
	createSubmission(t, env, userSolo.ID, challenge.ID, true, now)

	points, err := env.challengeRepo.ChallengePoints(context.Background())
	if err != nil {
		t.Fatalf("ChallengePoints: %v", err)
	}

	if points[challenge.ID] != 500 {
		t.Fatalf("expected fixed challenge to be 500, got %d", points[challenge.ID])
	}

	if points[other.ID] != other.Points {
		t.Fatalf("expected static challenge to be %d, got %d", other.Points, points[other.ID])
	}

	solveCounts, err := env.challengeRepo.SolveCounts(context.Background())
	if err != nil {
		t.Fatalf("SolveCounts: %v", err)
	}

	if solveCounts[challenge.ID] != 2 {
		t.Fatalf("expected solve count 2, got %d", solveCounts[challenge.ID])
	}

	if _, ok := solveCounts[other.ID]; ok {
		t.Fatalf("expected no solve count entry for unsolved challenge")
	}

	pointsByIDs, err := env.challengeRepo.ChallengePointsByIDs(context.Background(), []int64{challenge.ID})
	if err != nil {
		t.Fatalf("ChallengePointsByIDs: %v", err)
	}

	if len(pointsByIDs) != 1 || pointsByIDs[challenge.ID] != 500 {
		t.Fatalf("unexpected points by ids: %+v", pointsByIDs)
	}

	solveCountsByIDs, err := env.challengeRepo.SolveCountsByIDs(context.Background(), []int64{challenge.ID})
	if err != nil {
		t.Fatalf("SolveCountsByIDs: %v", err)
	}

	if solveCountsByIDs[challenge.ID] != 2 {
		t.Fatalf("expected solve count by ids 2, got %d", solveCountsByIDs[challenge.ID])
	}
}

func TestChallengeRepoChallengePointsError(t *testing.T) {
	if skipRepoIntegration {
		t.Skip("integration tests disabled via WARGAME_SKIP_INTEGRATION")
	}

	closedDB := newClosedRepoDB(t)
	repo := NewChallengeRepo(closedDB)

	if _, err := repo.ChallengePoints(context.Background()); err == nil {
		t.Fatalf("expected error from ChallengePoints")
	}
}

func newClosedRepoDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := db.New(repoCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}

	_ = conn.Close()
	return conn
}
