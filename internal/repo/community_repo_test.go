package repo

import (
	"context"
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
