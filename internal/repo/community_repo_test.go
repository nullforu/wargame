package repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestCommunityRepoCRUDAndPaging(t *testing.T) {
	env := setupRepoTest(t)
	u1 := createUser(t, env, "p1@example.com", "p1", "pass", models.UserRole)
	u2 := createUser(t, env, "p2@example.com", "p2", "pass", models.AdminRole)

	repo := NewCommunityRepo(env.db)
	now := time.Now().UTC()
	p1 := &models.CommunityPost{UserID: u1.ID, Category: models.CommunityCategoryFree, Title: "free one", Content: "hello", ViewCount: 2, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)}
	p2 := &models.CommunityPost{UserID: u2.ID, Category: models.CommunityCategoryNotice, Title: "notice one", Content: "important", ViewCount: 10, CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}

	if err := repo.Create(context.Background(), p1); err != nil {
		t.Fatalf("Create p1: %v", err)
	}

	if err := repo.Create(context.Background(), p2); err != nil {
		t.Fatalf("Create p2: %v", err)
	}

	if err := repo.CreateLike(context.Background(), p1.ID, u1.ID); err != nil {
		t.Fatalf("CreateLike p1/u1: %v", err)
	}

	if err := repo.CreateLike(context.Background(), p1.ID, u2.ID); err != nil {
		t.Fatalf("CreateLike p1/u2: %v", err)
	}

	if err := repo.IncrementViewCount(context.Background(), p1.ID); err != nil {
		t.Fatalf("IncrementViewCount: %v", err)
	}

	got, err := repo.GetByID(context.Background(), p1.ID)
	if err != nil || got.ViewCount != 3 {
		t.Fatalf("GetByID: got=%+v err=%v", got, err)
	}

	detail, err := repo.GetDetailByID(context.Background(), p2.ID, u1.ID)
	if err != nil {
		t.Fatalf("GetDetailByID: %v", err)
	}

	if detail.Username != u2.Username || detail.Title != "notice one" {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	popularRows, total, err := repo.Page(context.Background(), CommunityListFilter{Sort: "popular"}, 1, 10, u1.ID)
	if err != nil {
		t.Fatalf("Page popular: %v", err)
	}

	if total != 2 || len(popularRows) != 2 || popularRows[0].ID != p1.ID {
		t.Fatalf("unexpected popular rows=%+v total=%d", popularRows, total)
	}

	filtered, totalFiltered, err := repo.Page(context.Background(), CommunityListFilter{Category: ptrInt(models.CommunityCategoryNotice), Query: "notice", Sort: "latest"}, 1, 10, u1.ID)
	if err != nil {
		t.Fatalf("Page filtered: %v", err)
	}

	if totalFiltered != 1 || len(filtered) != 1 || filtered[0].ID != p2.ID {
		t.Fatalf("unexpected filtered rows=%+v total=%d", filtered, totalFiltered)
	}

	p1.Title = "free updated"
	p1.Content = "updated"
	p1.UpdatedAt = time.Now().UTC()
	if err := repo.Update(context.Background(), p1); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := repo.DeleteByID(context.Background(), p1.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	if _, err := repo.GetByID(context.Background(), p1.ID); err == nil {
		t.Fatalf("expected not found")
	}
}

func ptrInt(v int) *int { return &v }

func TestCommunityRepoLikes(t *testing.T) {
	env := setupRepoTest(t)
	u1 := createUser(t, env, "lp1@example.com", "lp1", "pass", models.UserRole)
	u2 := createUser(t, env, "lp2@example.com", "lp2", "pass", models.UserRole)
	repo := NewCommunityRepo(env.db)
	post := &models.CommunityPost{UserID: u1.ID, Category: models.CommunityCategoryFree, Title: "liked", Content: "body", ViewCount: 0, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	if err := repo.CreateLike(context.Background(), post.ID, u1.ID); err != nil {
		t.Fatalf("create like1: %v", err)
	}

	if err := repo.CreateLike(context.Background(), post.ID, u2.ID); err != nil {
		t.Fatalf("create like2: %v", err)
	}

	exists, err := repo.HasLikeByPostAndUser(context.Background(), post.ID, u1.ID)
	if err != nil || !exists {
		t.Fatalf("expected like exists, exists=%v err=%v", exists, err)
	}

	count, err := repo.CountLikesByPostID(context.Background(), post.ID)
	if err != nil || count != 2 {
		t.Fatalf("count likes expected 2, got %d err=%v", count, err)
	}

	likes, total, err := repo.LikesByPostPage(context.Background(), post.ID, 1, 10)
	if err != nil || total != 2 || len(likes) != 2 {
		t.Fatalf("likes page unexpected total=%d len=%d err=%v", total, len(likes), err)
	}

	detail, err := repo.GetDetailByID(context.Background(), post.ID, u1.ID)
	if err != nil || !detail.LikedByMe || detail.LikeCount != 2 {
		t.Fatalf("detail like fields unexpected detail=%+v err=%v", detail, err)
	}

	if err := repo.DeleteLike(context.Background(), post.ID, u1.ID); err != nil {
		t.Fatalf("delete like: %v", err)
	}

	count, err = repo.CountLikesByPostID(context.Background(), post.ID)
	if err != nil || count != 1 {
		t.Fatalf("count likes expected 1, got %d err=%v", count, err)
	}
}

func TestCommunityRepoAdditionalBranches(t *testing.T) {
	env := setupRepoTest(t)
	u1 := createUser(t, env, "extra1@example.com", "extra1", "pass", models.UserRole)
	u2 := createUser(t, env, "extra2@example.com", "extra2", "pass", models.UserRole)
	repo := NewCommunityRepo(env.db)
	now := time.Now().UTC()
	oldPost := &models.CommunityPost{UserID: u1.ID, Category: models.CommunityCategoryFree, Title: "old", Content: "a", ViewCount: 0, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	newPost := &models.CommunityPost{UserID: u2.ID, Category: models.CommunityCategoryFree, Title: "new", Content: "b", ViewCount: 0, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(context.Background(), oldPost); err != nil {
		t.Fatalf("create old: %v", err)
	}

	if err := repo.Create(context.Background(), newPost); err != nil {
		t.Fatalf("create new: %v", err)
	}

	for i := 0; i < models.PopularPostLikeThreshold; i += 1 {
		u := createUser(t, env, "extra-like-"+itoa(i)+"@example.com", "extra-like-"+itoa(i), "pass", models.UserRole)
		if err := repo.CreateLike(context.Background(), newPost.ID, u.ID); err != nil {
			t.Fatalf("seed like: %v", err)
		}
	}

	oldestRows, _, err := repo.Page(context.Background(), CommunityListFilter{Sort: "oldest"}, 1, 10, u1.ID)
	if err != nil || len(oldestRows) < 2 || oldestRows[0].ID != oldPost.ID {
		t.Fatalf("oldest sort unexpected rows=%+v err=%v", oldestRows, err)
	}

	popularRows, _, err := repo.Page(context.Background(), CommunityListFilter{PopularOnly: true}, 1, 10, u1.ID)
	if err != nil || len(popularRows) != 1 || popularRows[0].ID != newPost.ID {
		t.Fatalf("popular only unexpected rows=%+v err=%v", popularRows, err)
	}

	exists, err := repo.HasLikeByPostAndUser(context.Background(), oldPost.ID, u1.ID)
	if err != nil || exists {
		t.Fatalf("expected no like exists=%v err=%v", exists, err)
	}

	if _, err := repo.GetDetailByID(context.Background(), 999999, u1.ID); err == nil {
		t.Fatalf("expected detail not found")
	}

	emptyLikes, emptyTotal, err := repo.LikesByPostPage(context.Background(), oldPost.ID, 1, 10)
	if err != nil || emptyTotal != 0 || len(emptyLikes) != 0 {
		t.Fatalf("expected empty likes page, rows=%+v total=%d err=%v", emptyLikes, emptyTotal, err)
	}
}

func itoa(v int) string { return fmt.Sprintf("%d", v) }
