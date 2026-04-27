package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"
	"wargame/internal/models"
)

func TestListUsers(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "user1@example.com", "user1", "pass1", models.UserRole)
	_ = createUser(t, env, "user2@example.com", "user2", "pass2", models.UserRole)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "pass3", models.AdminRole)

	reason := "policy"
	bio := "security enthusiast"
	user1.BlockedReason = &reason
	user1.Bio = &bio
	now := time.Now().UTC()
	user1.BlockedAt = &now
	if err := env.userRepo.Update(context.Background(), user1); err != nil {
		t.Fatalf("update user: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/users", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Users []struct {
			ID            int64   `json:"id"`
			Username      string  `json:"username"`
			Role          string  `json:"role"`
			Affiliation   *string `json:"affiliation"`
			Bio           *string `json:"bio"`
			BlockedReason *string `json:"blocked_reason"`
		} `json:"users"`
		Pagination struct {
			Page       int  `json:"page"`
			PageSize   int  `json:"page_size"`
			TotalCount int  `json:"total_count"`
			HasNext    bool `json:"has_next"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(resp.Users))
	}

	if resp.Users[0].Username != models.AdminRole || resp.Users[1].Username != "user2" || resp.Users[2].Username != "user1" {
		t.Fatalf("unexpected response: %+v", resp.Users)
	}

	if resp.Users[2].BlockedReason == nil {
		t.Fatalf("expected blocked reason for user1")
	}
	if resp.Users[2].Bio == nil || *resp.Users[2].Bio != bio {
		t.Fatalf("expected bio for user1, got %+v", resp.Users[2].Bio)
	}

	if resp.Pagination.Page != 1 || resp.Pagination.PageSize != 20 || resp.Pagination.TotalCount != 3 || resp.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", resp.Pagination)
	}
}

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", models.UserRole)
		reason := "policy"
		bio := "security enthusiast"
		user.BlockedReason = &reason
		user.Bio = &bio
		now := time.Now().UTC()
		user.BlockedAt = &now
		if err := env.userRepo.Update(context.Background(), user); err != nil {
			t.Fatalf("update user: %v", err)
		}

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID), nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			ID            int64   `json:"id"`
			Username      string  `json:"username"`
			Role          string  `json:"role"`
			Bio           *string `json:"bio"`
			BlockedReason *string `json:"blocked_reason"`
		}
		decodeJSON(t, rec, &resp)

		if resp.ID != user.ID || resp.Username != "user1" || resp.Role != models.UserRole || resp.BlockedReason == nil {
			t.Fatalf("unexpected response: %+v", resp)
		}
		if resp.Bio == nil || *resp.Bio != bio {
			t.Fatalf("unexpected bio: %+v", resp.Bio)
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/invalid", nil, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestListUsersPaginationAndSearch(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "alpha@example.com", "alpha", "pass1", models.UserRole)
	_ = createUser(t, env, "beta@example.com", "beta", "pass2", models.UserRole)
	_ = createUser(t, env, "admin@example.com", "admin", "pass3", models.AdminRole)

	rec := doRequest(t, env.router, http.MethodGet, "/api/users?page=2&page_size=1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var pagedResp struct {
		Users []struct {
			Username string `json:"username"`
		} `json:"users"`
		Pagination struct {
			Page       int  `json:"page"`
			PageSize   int  `json:"page_size"`
			TotalCount int  `json:"total_count"`
			HasPrev    bool `json:"has_prev"`
			HasNext    bool `json:"has_next"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &pagedResp)
	if len(pagedResp.Users) != 1 || pagedResp.Users[0].Username != "beta" {
		t.Fatalf("unexpected paged users: %+v", pagedResp.Users)
	}
	if pagedResp.Pagination.Page != 2 || pagedResp.Pagination.PageSize != 1 || pagedResp.Pagination.TotalCount != 3 || !pagedResp.Pagination.HasPrev || !pagedResp.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagedResp.Pagination)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users/search?q=alp&page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec.Code, rec.Body.String())
	}
	var searchResp struct {
		Users []struct {
			Username string `json:"username"`
		} `json:"users"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &searchResp)
	if len(searchResp.Users) != 1 || searchResp.Users[0].Username != "alpha" || searchResp.Pagination.TotalCount != 1 {
		t.Fatalf("unexpected search response: %+v", searchResp)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users/search?q=&page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for empty q, got %d", rec.Code)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users?page=-1", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid page, got %d", rec.Code)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/users?page=1&page_size=101", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for oversized page_size, got %d", rec.Code)
	}
}

func TestGetUserSolved(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", models.UserRole)
		challenge1 := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
		challenge2 := createChallenge(t, env, "Warmup 2", 150, "flag{ok2}", true)
		now := time.Now().UTC()
		createSubmission(t, env, user.ID, challenge1.ID, true, now.Add(-2*time.Minute))
		createSubmission(t, env, user.ID, challenge2.ID, true, now.Add(-time.Minute))

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Solved []struct {
				ChallengeID int64  `json:"challenge_id"`
				Title       string `json:"title"`
				Points      int    `json:"points"`
				SolvedAt    string `json:"solved_at"`
			} `json:"solved"`
			Pagination struct {
				Page       int `json:"page"`
				TotalCount int `json:"total_count"`
			} `json:"pagination"`
		}
		decodeJSON(t, rec, &resp)

		if len(resp.Solved) != 2 {
			t.Fatalf("expected 2 solved challenges, got %d", len(resp.Solved))
		}

		if resp.Solved[0].ChallengeID != challenge2.ID || resp.Solved[0].Title != "Warmup 2" || resp.Solved[0].Points != 150 {
			t.Fatalf("unexpected response: %+v", resp)
		}
		if resp.Pagination.Page != 1 || resp.Pagination.TotalCount != 2 {
			t.Fatalf("unexpected pagination: %+v", resp.Pagination)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", models.UserRole)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Solved []any `json:"solved"`
		}
		decodeJSON(t, rec, &resp)

		if len(resp.Solved) != 0 {
			t.Fatalf("expected empty list, got %d", len(resp.Solved))
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999/solved", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}
