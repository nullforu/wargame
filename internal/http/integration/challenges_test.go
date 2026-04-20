package http_test

import (
	"net/http"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestListChallengesPaginationAndSearch(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	for _, body := range []map[string]any{
		{
			"title":       "Web Warmup",
			"description": "desc",
			"category":    "Web",
			"points":      100,
			"flag":        "flag{1}",
			"is_active":   true,
		},
		{
			"title":       "Web Advanced",
			"description": "desc",
			"category":    "Web",
			"points":      200,
			"flag":        "flag{2}",
			"is_active":   true,
		},
		{
			"title":       "Crypto Basic",
			"description": "desc",
			"category":    "Crypto",
			"points":      150,
			"flag":        "flag{3}",
			"is_active":   true,
		},
	} {
		rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", body, authHeader(adminAccess))
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
		}
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges?page=2&page_size=1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var pagedResp struct {
		Challenges []struct {
			Title string `json:"title"`
		} `json:"challenges"`
		Pagination struct {
			Page       int  `json:"page"`
			PageSize   int  `json:"page_size"`
			TotalCount int  `json:"total_count"`
			HasPrev    bool `json:"has_prev"`
			HasNext    bool `json:"has_next"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &pagedResp)
	if len(pagedResp.Challenges) != 1 || pagedResp.Challenges[0].Title != "Web Advanced" {
		t.Fatalf("unexpected paged challenges: %+v", pagedResp.Challenges)
	}

	if pagedResp.Pagination.Page != 2 || pagedResp.Pagination.PageSize != 1 || pagedResp.Pagination.TotalCount != 3 || !pagedResp.Pagination.HasPrev || !pagedResp.Pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagedResp.Pagination)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=web&page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec.Code, rec.Body.String())
	}

	var searchResp struct {
		Challenges []struct {
			Title string `json:"title"`
		} `json:"challenges"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &searchResp)
	if len(searchResp.Challenges) != 2 || searchResp.Pagination.TotalCount != 2 {
		t.Fatalf("unexpected search response: %+v", searchResp)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for empty q, got %d", rec.Code)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?page=abc", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid page, got %d", rec.Code)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?page=1&page_size=101", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for oversized page_size, got %d", rec.Code)
	}
}

func TestChallengeDetailFiltersSolvedAndSolvers(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	user := createUser(t, env, "player@example.com", "player", "playerpass", models.UserRole)
	userAccess, _, _ := loginUser(t, env.router, "player@example.com", "playerpass")

	var created []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	for _, body := range []map[string]any{
		{
			"title":       "Web L3",
			"description": "desc",
			"category":    "Web",
			"level":       3,
			"points":      300,
			"flag":        "flag{web3}",
			"is_active":   true,
		},
		{
			"title":       "Crypto L7",
			"description": "desc",
			"category":    "Crypto",
			"level":       7,
			"points":      700,
			"flag":        "flag{crypto7}",
			"is_active":   true,
		},
	} {
		rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", body, authHeader(adminAccess))
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		}
		decodeJSON(t, rec, &resp)
		created = append(created, resp)
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(created[0].ID)+"/submit", map[string]string{"flag": "flag{web3}"}, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?page=1&page_size=20", nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Challenges []struct {
			ID       int64 `json:"id"`
			Level    int   `json:"level"`
			IsSolved bool  `json:"is_solved"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &listResp)
	if len(listResp.Challenges) < 2 {
		t.Fatalf("expected challenges, got %+v", listResp)
	}

	foundSolved := false
	for _, item := range listResp.Challenges {
		if item.ID == created[0].ID {
			foundSolved = item.IsSolved && item.Level == 3
		}
	}

	if !foundSolved {
		t.Fatalf("expected solved challenge with level in list: %+v", listResp.Challenges)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=Web&category=Web&level=3&solved=true&page=1&page_size=20", nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec.Code, rec.Body.String())
	}

	var filtered struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &filtered)
	if len(filtered.Challenges) != 1 || filtered.Challenges[0].ID != created[0].ID {
		t.Fatalf("unexpected filtered response: %+v", filtered)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created[0].ID), nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status %d: %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		ID       int64 `json:"id"`
		Level    int   `json:"level"`
		IsSolved bool  `json:"is_solved"`
	}
	decodeJSON(t, rec, &detail)
	if detail.ID != created[0].ID || detail.Level != 3 || !detail.IsSolved {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	other1 := createUser(t, env, "solver1@example.com", "solver1", "pass", models.UserRole)
	other2 := createUser(t, env, "solver2@example.com", "solver2", "pass", models.UserRole)
	createSubmission(t, env, other1.ID, created[0].ID, true, time.Now().UTC().Add(1*time.Minute))
	createSubmission(t, env, other2.ID, created[0].ID, true, time.Now().UTC().Add(2*time.Minute))
	createSubmission(t, env, user.ID, created[0].ID, true, time.Now().UTC().Add(3*time.Minute))

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created[0].ID)+"/solvers?page=1&page_size=2", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("solvers status %d: %s", rec.Code, rec.Body.String())
	}

	var solvers struct {
		Solvers []struct {
			UserID int64 `json:"user_id"`
		} `json:"solvers"`
		Pagination struct {
			TotalCount int  `json:"total_count"`
			HasNext    bool `json:"has_next"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &solvers)
	if len(solvers.Solvers) != 2 || solvers.Pagination.TotalCount < 3 || !solvers.Pagination.HasNext {
		t.Fatalf("unexpected solvers response: %+v", solvers)
	}
}
