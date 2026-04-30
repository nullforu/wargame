package repo

import (
	"context"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestChallengeCommentRepoCRUDAndPaging(t *testing.T) {
	env := setupRepoTest(t)
	u1 := createUser(t, env, "c1@example.com", "c1", "pass", models.UserRole)
	u2 := createUser(t, env, "c2@example.com", "c2", "pass", models.UserRole)
	ch := createChallenge(t, env, "Comment Ch", 100, "FLAG{C}", true)

	repo := NewChallengeCommentRepo(env.db)
	now := time.Now().UTC()
	c1 := &models.ChallengeCommentItem{UserID: u1.ID, ChallengeID: ch.ID, Content: "first", CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)}
	c2 := &models.ChallengeCommentItem{UserID: u2.ID, ChallengeID: ch.ID, Content: "second", CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}
	if err := repo.Create(context.Background(), c1); err != nil {
		t.Fatalf("Create c1: %v", err)
	}

	if err := repo.Create(context.Background(), c2); err != nil {
		t.Fatalf("Create c2: %v", err)
	}

	got, err := repo.GetByID(context.Background(), c1.ID)
	if err != nil || got.Content != "first" {
		t.Fatalf("GetByID: %+v err=%v", got, err)
	}

	detail, err := repo.GetDetailByID(context.Background(), c2.ID)
	if err != nil {
		t.Fatalf("GetDetailByID: %v", err)
	}

	if detail.Username != u2.Username || detail.ChallengeTitle != ch.Title {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	rows, total, err := repo.ChallengePage(context.Background(), ch.ID, 1, 1)
	if err != nil {
		t.Fatalf("ChallengePage: %v", err)
	}

	if total != 2 || len(rows) != 1 || rows[0].ID != c2.ID {
		t.Fatalf("unexpected page1 rows=%+v total=%d", rows, total)
	}

	rows2, _, err := repo.ChallengePage(context.Background(), ch.ID, 2, 1)
	if err != nil {
		t.Fatalf("ChallengePage page2: %v", err)
	}

	if len(rows2) != 1 || rows2[0].ID != c1.ID {
		t.Fatalf("unexpected page2 rows=%+v", rows2)
	}

	c1.Content = "first updated"
	c1.UpdatedAt = time.Now().UTC()
	if err := repo.Update(context.Background(), c1); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, _ := repo.GetByID(context.Background(), c1.ID)
	if updated.Content != "first updated" {
		t.Fatalf("update not applied: %+v", updated)
	}

	if err := repo.DeleteByID(context.Background(), c1.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	if _, err := repo.GetByID(context.Background(), c1.ID); err == nil {
		t.Fatalf("expected not found")
	}
}
