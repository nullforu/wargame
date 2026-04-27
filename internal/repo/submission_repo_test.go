package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestSubmissionRepoCreateAndHasCorrect(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user.ID, ch.ID); err != nil || ok {
		t.Fatalf("expected unsolved, ok=%v err=%v", ok, err)
	}

	createSubmission(t, env, user.ID, ch.ID, true, time.Now().UTC())
	if ok, err := env.submissionRepo.HasCorrect(context.Background(), user.ID, ch.ID); err != nil || !ok {
		t.Fatalf("expected solved, ok=%v err=%v", ok, err)
	}
}

func TestSubmissionRepoSolvedChallengesPageAndIDs(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user.ID, ch1.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user.ID, ch2.ID, true, time.Now().Add(-time.Minute))

	rows, totalCount, err := env.submissionRepo.SolvedChallengesPage(context.Background(), user.ID, 1, 20)
	if err != nil {
		t.Fatalf("SolvedChallengesPage: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if totalCount != 2 {
		t.Fatalf("expected total count 2, got %d", totalCount)
	}
	if rows[0].ChallengeID != ch2.ID || rows[1].ChallengeID != ch1.ID {
		t.Fatalf("expected solved challenges ordered by latest solve time, got %+v", rows)
	}

	ids, err := env.submissionRepo.SolvedChallengeIDs(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("SolvedChallengeIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
}

func TestSubmissionRepoCreateCorrectIfNotSolvedByUser(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	sub1 := &models.Submission{UserID: user.ID, ChallengeID: ch.ID, Provided: "flag{1}", Correct: true, SubmittedAt: time.Now().UTC()}
	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), sub1)
	if err != nil || !inserted {
		t.Fatalf("first insert failed, inserted=%v err=%v", inserted, err)
	}
	if !sub1.IsFirstBlood {
		t.Fatalf("expected first blood on first solve")
	}

	sub2 := &models.Submission{UserID: user.ID, ChallengeID: ch.ID, Provided: "flag{1}", Correct: true, SubmittedAt: time.Now().UTC().Add(time.Second)}
	inserted, err = env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), sub2)
	if err != nil {
		t.Fatalf("second insert error: %v", err)
	}
	if inserted {
		t.Fatalf("expected duplicate correct blocked")
	}
}

func TestSubmissionRepoListAll(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "sub@example.com", "sub", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Sub", 100, "flag{sub}", true)

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC().Add(-time.Minute))
	createSubmission(t, env, user.ID, challenge.ID, false, time.Now().UTC())

	rows, err := env.submissionRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(rows))
	}
}

func TestSubmissionRepoCreateCorrectFalsePath(t *testing.T) {
	env := setupRepoTest(t)
	user := createUserForTestUserScope(t, env, "u2@example.com", "u2", "pass", models.UserRole)
	ch := createChallenge(t, env, "ch2", 100, "FLAG{2}", true)

	sub := &models.Submission{UserID: user.ID, ChallengeID: ch.ID, Provided: "bad", Correct: false, SubmittedAt: time.Now().UTC()}
	inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), sub)
	if err != nil || !inserted {
		t.Fatalf("expected false submission insert, inserted=%v err=%v", inserted, err)
	}

	rows, err := env.submissionRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(rows) != 1 || rows[0].Correct {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestSubmissionRepoEmptyAndErrorPaths(t *testing.T) {
	env := setupRepoTest(t)

	rows, totalCount, err := env.submissionRepo.SolvedChallengesPage(context.Background(), 999999, 1, 20)
	if err != nil {
		t.Fatalf("SolvedChallengesPage empty: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty solved rows, got %+v", rows)
	}
	if totalCount != 0 {
		t.Fatalf("expected total count 0, got %d", totalCount)
	}

	ids, err := env.submissionRepo.SolvedChallengeIDs(context.Background(), 999999)
	if err != nil {
		t.Fatalf("SolvedChallengeIDs empty: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty solved ids, got %+v", ids)
	}

	closedDB := newClosedRepoDB(t)
	closedRepo := NewSubmissionRepo(closedDB)
	if _, err := closedRepo.HasCorrect(context.Background(), 1, 1); err == nil {
		t.Fatalf("expected HasCorrect error on closed db")
	}
	if _, err := closedRepo.ListAll(context.Background()); err == nil {
		t.Fatalf("expected ListAll error on closed db")
	}

	if _, err := env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), &models.Submission{UserID: 999999, ChallengeID: 999999, Correct: true, SubmittedAt: time.Now().UTC()}); err == nil {
		t.Fatalf("expected insert error for invalid foreign keys")
	}

	if _, err := env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), &models.Submission{UserID: 0, ChallengeID: 0, Correct: true, SubmittedAt: time.Now().UTC()}); err == nil && !errors.Is(err, ErrNotFound) {
		// Any non-nil DB error is acceptable here; this assert only guards accidental silent success.
		t.Fatalf("expected failure for invalid ids")
	}
}

func TestSubmissionRepoChallengeSolversPageOrdersLatestFirst(t *testing.T) {
	env := setupRepoTest(t)
	ch := createChallenge(t, env, "solvers", 100, "FLAG{SOLVERS}", true)
	older := createUserForTestUserScope(t, env, "older@example.com", "older", "pass", models.UserRole)
	newer := createUserForTestUserScope(t, env, "newer@example.com", "newer", "pass", models.UserRole)

	now := time.Now().UTC()
	createSubmission(t, env, older.ID, ch.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, newer.ID, ch.ID, true, now.Add(-time.Minute))

	page1, totalCount, err := env.submissionRepo.ChallengeSolversPage(context.Background(), ch.ID, 1, 1)
	if err != nil {
		t.Fatalf("ChallengeSolversPage page1: %v", err)
	}
	if totalCount != 2 {
		t.Fatalf("expected total count 2, got %d", totalCount)
	}
	if len(page1) != 1 || page1[0].UserID != newer.ID {
		t.Fatalf("expected newest solver in first page, got %+v", page1)
	}

	page2, _, err := env.submissionRepo.ChallengeSolversPage(context.Background(), ch.ID, 2, 1)
	if err != nil {
		t.Fatalf("ChallengeSolversPage page2: %v", err)
	}
	if len(page2) != 1 || page2[0].UserID != older.ID {
		t.Fatalf("expected older solver in second page, got %+v", page2)
	}
}
