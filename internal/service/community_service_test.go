package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"wargame/internal/models"
)

func TestWargameServiceCommunityCRUDAndPolicies(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "community-user@example.com", "community-user", "pass", models.UserRole)
	admin := createUser(t, env, "community-admin@example.com", "community-admin", "pass", models.AdminRole)

	if _, err := env.wargameSvc.CreateCommunityPost(context.Background(), user.ID, models.UserRole, models.CommunityCategoryNotice, "notice", "content"); !errors.Is(err, ErrCommunityForbidden) {
		t.Fatalf("expected notice create forbidden, got %v", err)
	}

	notice, err := env.wargameSvc.CreateCommunityPost(context.Background(), admin.ID, models.AdminRole, models.CommunityCategoryNotice, "notice", "content")
	if err != nil {
		t.Fatalf("CreateCommunityPost notice: %v", err)
	}

	freePost, err := env.wargameSvc.CreateCommunityPost(context.Background(), user.ID, models.UserRole, models.CommunityCategoryFree, "free", "body")
	if err != nil {
		t.Fatalf("CreateCommunityPost free: %v", err)
	}

	if _, err := env.wargameSvc.UpdateCommunityPost(context.Background(), user.ID, models.UserRole, notice.ID, nil, strPtr("hack"), nil); !errors.Is(err, ErrCommunityForbidden) {
		t.Fatalf("expected update notice forbidden, got %v", err)
	}

	updatedNotice, err := env.wargameSvc.UpdateCommunityPost(context.Background(), admin.ID, models.AdminRole, notice.ID, nil, strPtr("notice updated"), nil)
	if err != nil || updatedNotice.Title != "notice updated" {
		t.Fatalf("admin update notice failed: row=%+v err=%v", updatedNotice, err)
	}

	if _, err := env.wargameSvc.UpdateCommunityPost(context.Background(), admin.ID, models.AdminRole, freePost.ID, intPtr(models.CommunityCategoryNotice), nil, nil); err != nil {
		t.Fatalf("admin promote to notice failed: %v", err)
	}

	if err := env.wargameSvc.DeleteCommunityPost(context.Background(), user.ID, models.UserRole, freePost.ID); !errors.Is(err, ErrCommunityForbidden) {
		t.Fatalf("expected user delete promoted notice forbidden, got %v", err)
	}

	if err := env.wargameSvc.DeleteCommunityPost(context.Background(), admin.ID, models.AdminRole, freePost.ID); err != nil {
		t.Fatalf("admin delete promoted notice: %v", err)
	}

	list, pagination, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 10, "notice", intPtr(models.CommunityCategoryNotice), false, false, "popular", admin.ID)
	if err != nil {
		t.Fatalf("CommunityPostsPage: %v", err)
	}

	if pagination.TotalCount != 1 || len(list) != 1 || list[0].ID != notice.ID {
		t.Fatalf("unexpected list result: rows=%+v pagination=%+v", list, pagination)
	}

	normalOnly, normalPagination, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 10, "", nil, true, false, "latest", admin.ID)
	if err != nil {
		t.Fatalf("CommunityPostsPage exclude notice: %v", err)
	}

	if normalPagination.TotalCount != 0 || len(normalOnly) != 0 {
		t.Fatalf("expected no non-notice posts, got rows=%d pagination=%+v", len(normalOnly), normalPagination)
	}

	detail, err := env.wargameSvc.CommunityPostByID(context.Background(), notice.ID, user.ID, true)
	if err != nil {
		t.Fatalf("CommunityPostByID: %v", err)
	}

	if detail.ViewCount != 1 {
		t.Fatalf("expected view_count 1, got %d", detail.ViewCount)
	}

	liked, likeCount, err := env.wargameSvc.ToggleCommunityPostLike(context.Background(), user.ID, notice.ID)
	if err != nil || !liked || likeCount != 1 {
		t.Fatalf("toggle like on failed: liked=%v likeCount=%d err=%v", liked, likeCount, err)
	}

	detail, err = env.wargameSvc.CommunityPostByID(context.Background(), notice.ID, user.ID, false)
	if err != nil || !detail.LikedByMe || detail.LikeCount != 1 {
		t.Fatalf("expected liked_by_me with count=1, detail=%+v err=%v", detail, err)
	}

	liked, likeCount, err = env.wargameSvc.ToggleCommunityPostLike(context.Background(), user.ID, notice.ID)
	if err != nil || liked || likeCount != 0 {
		t.Fatalf("toggle like off failed: liked=%v likeCount=%d err=%v", liked, likeCount, err)
	}

	for i := 0; i < models.PopularPostLikeThreshold; i += 1 {
		u := createUser(t, env, "popular-like-"+toString(i)+"@example.com", "popular-like-"+toString(i), "pass", models.UserRole)
		if _, _, err := env.wargameSvc.ToggleCommunityPostLike(context.Background(), u.ID, notice.ID); err != nil {
			t.Fatalf("seed popular likes: %v", err)
		}
	}
	popularOnly, popularPagination, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 10, "", nil, false, true, "latest", admin.ID)
	if err != nil {
		t.Fatalf("popular-only list failed: %v", err)
	}

	if popularPagination.TotalCount != 1 || len(popularOnly) != 1 || popularOnly[0].ID != notice.ID {
		t.Fatalf("unexpected popular-only rows=%+v pagination=%+v", popularOnly, popularPagination)
	}
}

func TestWargameServiceCommunityValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "community-validation@example.com", "community-validation", "pass", models.UserRole)

	if _, err := env.wargameSvc.CreateCommunityPost(context.Background(), user.ID, models.UserRole, 99, "", ""); err == nil {
		t.Fatalf("expected create validation error")
	}

	if _, err := env.wargameSvc.UpdateCommunityPost(context.Background(), user.ID, models.UserRole, 1, nil, nil, nil); err == nil {
		t.Fatalf("expected update empty-request validation error")
	}

	if _, err := env.wargameSvc.UpdateCommunityPost(context.Background(), user.ID, models.UserRole, 999999, nil, strPtr("x"), nil); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected update not found, got %v", err)
	}

	if err := env.wargameSvc.DeleteCommunityPost(context.Background(), user.ID, models.UserRole, 999999); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected delete not found, got %v", err)
	}

	if _, err := env.wargameSvc.CommunityPostByID(context.Background(), 0, user.ID, false); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for post by id, got %v", err)
	}

	if _, _, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 101, "", nil, false, false, "latest", user.ID); err == nil {
		t.Fatalf("expected invalid pagination")
	}

	if _, _, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 10, "", nil, false, false, "bad-sort", user.ID); err == nil {
		t.Fatalf("expected invalid sort")
	}

	if _, _, err := env.wargameSvc.CommunityPostsPage(context.Background(), 1, 10, "", intPtr(99), false, false, "latest", user.ID); err == nil {
		t.Fatalf("expected invalid category")
	}

	if _, _, err := env.wargameSvc.ToggleCommunityPostLike(context.Background(), 0, 1); err == nil {
		t.Fatalf("expected toggle like validation error")
	}

	if _, _, err := env.wargameSvc.ToggleCommunityPostLike(context.Background(), user.ID, 999999); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected toggle like not found, got %v", err)
	}

	if _, _, err := env.wargameSvc.CommunityPostLikesPage(context.Background(), 0, 1, 20); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected likes invalid input, got %v", err)
	}

	if _, _, err := env.wargameSvc.CommunityPostLikesPage(context.Background(), 999999, 1, 20); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected likes not found, got %v", err)
	}
}

func TestWargameServiceCommunityComments(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "community-comment-user@example.com", "community-comment-user", "pass", models.UserRole)
	other := createUser(t, env, "community-comment-other@example.com", "community-comment-other", "pass", models.UserRole)
	post, err := env.wargameSvc.CreateCommunityPost(context.Background(), user.ID, models.UserRole, models.CommunityCategoryFree, "free", "body")
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	if _, err := env.wargameSvc.CreateCommunityComment(context.Background(), 0, post.ID, "x"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid user id, got %v", err)
	}

	if _, err := env.wargameSvc.CreateCommunityComment(context.Background(), user.ID, post.ID, "  "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid empty content, got %v", err)
	}

	if _, err := env.wargameSvc.CreateCommunityComment(context.Background(), user.ID, 999999, "x"); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected missing post, got %v", err)
	}

	c1, err := env.wargameSvc.CreateCommunityComment(context.Background(), user.ID, post.ID, " first ")
	if err != nil || c1.Content != "first" {
		t.Fatalf("create comment failed row=%+v err=%v", c1, err)
	}

	if _, err := env.wargameSvc.UpdateCommunityComment(context.Background(), user.ID, c1.ID, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected empty update request, got %v", err)
	}

	if _, err := env.wargameSvc.UpdateCommunityComment(context.Background(), other.ID, c1.ID, strPtr("hack")); !errors.Is(err, ErrCommunityCommentForbidden) {
		t.Fatalf("expected forbidden update, got %v", err)
	}

	updated, err := env.wargameSvc.UpdateCommunityComment(context.Background(), user.ID, c1.ID, strPtr(" updated "))
	if err != nil || updated.Content != "updated" {
		t.Fatalf("update comment failed row=%+v err=%v", updated, err)
	}

	if err := env.wargameSvc.DeleteCommunityComment(context.Background(), other.ID, c1.ID); !errors.Is(err, ErrCommunityCommentForbidden) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}

	if _, err := env.wargameSvc.CreateCommunityComment(context.Background(), other.ID, post.ID, "second"); err != nil {
		t.Fatalf("create second: %v", err)
	}

	rows, pag, err := env.wargameSvc.CommunityCommentsPage(context.Background(), post.ID, 1, 1)
	if err != nil || len(rows) != 1 || pag.TotalCount != 2 {
		t.Fatalf("comments page mismatch rows=%d pag=%+v err=%v", len(rows), pag, err)
	}

	postDetail, err := env.wargameSvc.CommunityPostByID(context.Background(), post.ID, user.ID, false)
	if err != nil || postDetail.CommentCount != 2 {
		t.Fatalf("expected post comment_count=2 row=%+v err=%v", postDetail, err)
	}

	if err := env.wargameSvc.DeleteCommunityComment(context.Background(), user.ID, c1.ID); err != nil {
		t.Fatalf("delete own comment: %v", err)
	}

	if err := env.wargameSvc.DeleteCommunityComment(context.Background(), user.ID, c1.ID); !errors.Is(err, ErrCommunityCommentNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}

	if _, _, err := env.wargameSvc.CommunityCommentsPage(context.Background(), 0, 1, 20); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid id page, got %v", err)
	}

	if _, _, err := env.wargameSvc.CommunityCommentsPage(context.Background(), 999999, 1, 20); !errors.Is(err, ErrCommunityPostNotFound) {
		t.Fatalf("expected missing post page, got %v", err)
	}
}

func strPtr(v string) *string { return &v }
func intPtr(v int) *int       { return &v }
func toString(v int) string {
	return fmt.Sprintf("%d", v)
}
