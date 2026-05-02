package http_test

import (
	"net/http"
	"testing"

	"wargame/internal/models"
)

func TestCommunityIntegrationFlow(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := createUser(t, env, "admin-community-int@example.com", "admin-community-int", "pass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "pass")
	user := createUser(t, env, "user-community-int@example.com", "user-community-int", "pass", models.UserRole)
	userAccess, _, _ := loginUser(t, env.router, user.Email, "pass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/community", map[string]any{"category": 0, "title": "notice", "content": "body"}, authHeader(userAccess))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user notice create, got %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/community", map[string]any{"category": 1, "title": "hello", "content": "**md**"}, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create free post: %d %s", rec.Code, rec.Body.String())
	}

	var free struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &free)

	rec = doRequest(t, env.router, http.MethodPost, "/api/community", map[string]any{"category": 0, "title": "notice", "content": "body"}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create notice post: %d %s", rec.Code, rec.Body.String())
	}

	var notice struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &notice)

	rec = doRequest(t, env.router, http.MethodPatch, "/api/community/"+itoa(notice.ID), map[string]any{"title": "notice updated"}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin update notice: %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community?page=1&page_size=10&sort=latest&category=0&q=notice", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list community: %d %s", rec.Code, rec.Body.String())
	}

	var listed struct {
		Posts []struct {
			ID int64 `json:"id"`
		} `json:"posts"`
	}
	decodeJSON(t, rec, &listed)
	if len(listed.Posts) != 1 || listed.Posts[0].ID != notice.ID {
		t.Fatalf("unexpected list posts: %+v", listed)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community?page=1&page_size=10&sort=latest&exclude_notice=true", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list community exclude notice: %d %s", rec.Code, rec.Body.String())
	}

	var listedWithoutNotice struct {
		Posts []struct {
			ID       int64 `json:"id"`
			Category int   `json:"category"`
		} `json:"posts"`
	}
	decodeJSON(t, rec, &listedWithoutNotice)
	if len(listedWithoutNotice.Posts) != 1 || listedWithoutNotice.Posts[0].ID != free.ID || listedWithoutNotice.Posts[0].Category == models.CommunityCategoryNotice {
		t.Fatalf("unexpected exclude_notice result: %+v", listedWithoutNotice)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community/"+itoa(notice.ID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get notice detail: %d %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		Post struct {
			ViewCount int `json:"view_count"`
		} `json:"post"`
	}
	decodeJSON(t, rec, &detail)
	if detail.Post.ViewCount != 1 {
		t.Fatalf("expected view_count 1, got %+v", detail)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/community/"+itoa(notice.ID)+"/likes", nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle like status %d: %s", rec.Code, rec.Body.String())
	}

	var likeToggle struct {
		Liked     bool `json:"liked"`
		LikeCount int  `json:"like_count"`
	}
	decodeJSON(t, rec, &likeToggle)
	if !likeToggle.Liked || likeToggle.LikeCount != 1 {
		t.Fatalf("unexpected like toggle response: %+v", likeToggle)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community/"+itoa(notice.ID)+"/likes?page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("likes list status %d: %s", rec.Code, rec.Body.String())
	}

	var likesList struct {
		Likes []struct {
			UserID int64 `json:"user_id"`
		} `json:"likes"`
	}
	decodeJSON(t, rec, &likesList)
	if len(likesList.Likes) != 1 || likesList.Likes[0].UserID != user.ID {
		t.Fatalf("unexpected likes list %+v", likesList)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community/"+itoa(notice.ID), nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("detail with auth status %d: %s", rec.Code, rec.Body.String())
	}

	var authedDetail struct {
		Post struct {
			LikedByMe bool `json:"liked_by_me"`
			LikeCount int  `json:"like_count"`
		} `json:"post"`
	}
	decodeJSON(t, rec, &authedDetail)
	if !authedDetail.Post.LikedByMe || authedDetail.Post.LikeCount != 1 {
		t.Fatalf("unexpected authed detail %+v", authedDetail)
	}

	for i := 0; i < models.PopularPostLikeThreshold-1; i += 1 {
		u := createUser(t, env, "popular-http-"+itoa(int64(i))+"@example.com", "popular-http-"+itoa(int64(i)), "pass", models.UserRole)
		access, _, _ := loginUser(t, env.router, u.Email, "pass")
		rec = doRequest(t, env.router, http.MethodPost, "/api/community/"+itoa(notice.ID)+"/likes", nil, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("seed popular likes status %d: %s", rec.Code, rec.Body.String())
		}
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/community?page=1&page_size=10&popular_only=true", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("popular-only list status %d: %s", rec.Code, rec.Body.String())
	}

	var popularListed struct {
		Posts []struct {
			ID        int64 `json:"id"`
			LikeCount int   `json:"like_count"`
		} `json:"posts"`
	}
	decodeJSON(t, rec, &popularListed)
	if len(popularListed.Posts) != 1 || popularListed.Posts[0].ID != notice.ID || popularListed.Posts[0].LikeCount < models.PopularPostLikeThreshold {
		t.Fatalf("unexpected popular-only posts %+v", popularListed)
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/community/"+itoa(free.ID), nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete own free post: %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/community/"+itoa(notice.ID), nil, authHeader(userAccess))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden delete notice by user, got %d %s", rec.Code, rec.Body.String())
	}
}
