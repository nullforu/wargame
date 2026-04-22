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
			Title             string `json:"title"`
			CreatedByUserID   int64  `json:"created_by_user_id"`
			CreatedByUsername string `json:"created_by_username"`
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
	if pagedResp.Challenges[0].CreatedByUserID <= 0 || pagedResp.Challenges[0].CreatedByUsername == "" {
		t.Fatalf("expected creator info in list response: %+v", pagedResp.Challenges[0])
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

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?page=1&page_size=10&sort=oldest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("sort oldest status %d: %s", rec.Code, rec.Body.String())
	}
	var oldestResp struct {
		Challenges []struct {
			Title string `json:"title"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &oldestResp)
	if len(oldestResp.Challenges) < 3 || oldestResp.Challenges[0].Title != "Web Warmup" {
		t.Fatalf("unexpected oldest sort response: %+v", oldestResp)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?page=1&page_size=10&sort=invalid", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid sort, got %d", rec.Code)
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
			"points":      300,
			"flag":        "flag{web3}",
			"is_active":   true,
		},
		{
			"title":       "Crypto L7",
			"description": "desc",
			"category":    "Crypto",
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
			foundSolved = item.IsSolved && item.Level == models.UnknownLevel
		}
	}

	if !foundSolved {
		t.Fatalf("expected solved challenge with unknown level in list: %+v", listResp.Challenges)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=Web&category=Web&solved=true&page=1&page_size=20", nil, authHeader(userAccess))
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
		ID                int64                   `json:"id"`
		Level             int                     `json:"level"`
		LevelVoteCounts   []models.LevelVoteCount `json:"level_vote_counts"`
		IsSolved          bool                    `json:"is_solved"`
		CreatedByUserID   int64                   `json:"created_by_user_id"`
		CreatedByUsername string                  `json:"created_by_username"`
	}
	decodeJSON(t, rec, &detail)
	if detail.ID != created[0].ID || detail.Level != models.UnknownLevel || !detail.IsSolved || len(detail.LevelVoteCounts) != 0 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.CreatedByUserID <= 0 || detail.CreatedByUsername == "" {
		t.Fatalf("expected creator info in detail response: %+v", detail)
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

func TestChallengesSortOptions(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	normal1 := createUser(t, env, "normal1@example.com", "normal1", "pass", models.UserRole)
	normal2 := createUser(t, env, "normal2@example.com", "normal2", "pass", models.UserRole)
	blocked := createUser(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)
	adminUser := createUser(t, env, "admin2@example.com", "sort_admin_user", "pass", models.AdminRole)

	createChallengeReq := func(title string) int64 {
		rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
			"title":       title,
			"description": "desc",
			"category":    "Web",
			"points":      100,
			"flag":        "flag{" + title + "}",
			"is_active":   true,
		}, authHeader(adminAccess))
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
		}
		var created struct {
			ID int64 `json:"id"`
		}
		decodeJSON(t, rec, &created)
		return created.ID
	}

	idA := createChallengeReq("Sort A")
	idB := createChallengeReq("Sort B")
	idC := createChallengeReq("Sort C")
	idD := createChallengeReq("Sort D")

	createSubmission(t, env, normal1.ID, idA, true, time.Now().UTC().Add(-5*time.Minute))
	createSubmission(t, env, normal2.ID, idA, true, time.Now().UTC().Add(-4*time.Minute))
	createSubmission(t, env, normal1.ID, idB, true, time.Now().UTC().Add(-3*time.Minute))
	createSubmission(t, env, normal2.ID, idD, true, time.Now().UTC().Add(-150*time.Second))
	createSubmission(t, env, blocked.ID, idC, true, time.Now().UTC().Add(-2*time.Minute))
	createSubmission(t, env, adminUser.ID, idC, true, time.Now().UTC().Add(-1*time.Minute))

	type challengeItem struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	parseIDs := func(path string) []int64 {
		rec := doRequest(t, env.router, http.MethodGet, path, nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d for %s: %s", rec.Code, path, rec.Body.String())
		}
		var resp struct {
			Challenges []challengeItem `json:"challenges"`
		}
		decodeJSON(t, rec, &resp)
		ids := make([]int64, 0, len(resp.Challenges))
		for _, c := range resp.Challenges {
			if c.ID == idA || c.ID == idB || c.ID == idC || c.ID == idD {
				ids = append(ids, c.ID)
			}
		}
		return ids
	}

	assertOrder := func(got []int64, want []int64) {
		if len(got) < len(want) {
			t.Fatalf("got too short order: got=%v want-prefix=%v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("unexpected order at %d: got=%v want-prefix=%v", i, got, want)
			}
		}
	}

	assertOrder(parseIDs("/api/challenges?page=1&page_size=50&sort=latest"), []int64{idD, idC, idB, idA})
	assertOrder(parseIDs("/api/challenges?page=1&page_size=50&sort=oldest"), []int64{idA, idB, idC, idD})
	assertOrder(parseIDs("/api/challenges?page=1&page_size=50&sort=most_solved"), []int64{idA, idD, idC, idB})
	assertOrder(parseIDs("/api/challenges?page=1&page_size=50&sort=least_solved"), []int64{idD, idC, idB, idA})

	assertOrder(parseIDs("/api/challenges/search?q=Sort&page=1&page_size=50&sort=latest"), []int64{idD, idC, idB, idA})
	assertOrder(parseIDs("/api/challenges/search?q=Sort&page=1&page_size=50&sort=oldest"), []int64{idA, idB, idC, idD})
	assertOrder(parseIDs("/api/challenges/search?q=Sort&page=1&page_size=50&sort=most_solved"), []int64{idA, idD, idC, idB})
	assertOrder(parseIDs("/api/challenges/search?q=Sort&page=1&page_size=50&sort=least_solved"), []int64{idD, idC, idB, idA})

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges?sort=bad", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid sort to return 400, got %d", rec.Code)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=Sort&sort=bad", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid sort(search) to return 400, got %d", rec.Code)
	}
}

func TestChallengeLevelVotes(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")
	user1 := createUser(t, env, "vote1@example.com", "vote1", "pass", models.UserRole)
	user2 := createUser(t, env, "vote2@example.com", "vote2", "pass", models.UserRole)
	user1Access, _, _ := loginUser(t, env.router, user1.Email, "pass")
	user2Access, _, _ := loginUser(t, env.router, user2.Email, "pass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Vote Target",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{vote}",
		"is_active":   true,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID    int64 `json:"id"`
		Level int   `json:"level"`
	}
	decodeJSON(t, rec, &created)
	if created.Level != models.UnknownLevel {
		t.Fatalf("expected unknown on create, got %d", created.Level)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(created.ID)+"/vote", map[string]any{
		"level": 6,
	}, authHeader(user1Access))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden before solving, got %d", rec.Code)
	}

	for _, u := range []struct {
		id     int64
		access string
	}{
		{user1.ID, user1Access},
		{user2.ID, user2Access},
	} {
		createSubmission(t, env, u.id, created.ID, true, time.Now().UTC())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(created.ID)+"/vote", map[string]any{
		"level": 6,
	}, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote1 status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(created.ID)+"/vote", map[string]any{
		"level": 7,
	}, authHeader(user2Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote2 status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(created.ID)+"/vote", map[string]any{
		"level": 6,
	}, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote1 update status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created.ID), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status %d: %s", rec.Code, rec.Body.String())
	}

	var detail struct {
		Level           int                     `json:"level"`
		LevelVoteCounts []models.LevelVoteCount `json:"level_vote_counts"`
	}
	decodeJSON(t, rec, &detail)
	if detail.Level != 6 {
		t.Fatalf("expected representative level 6, got %d", detail.Level)
	}

	if len(detail.LevelVoteCounts) != 2 {
		t.Fatalf("expected two level count entries, got %+v", detail.LevelVoteCounts)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created.ID)+"/votes?page=1&page_size=1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("votes status %d: %s", rec.Code, rec.Body.String())
	}

	var votes struct {
		Votes []struct {
			UserID int64 `json:"user_id"`
			Level  int   `json:"level"`
		} `json:"votes"`
		Pagination struct {
			TotalCount int  `json:"total_count"`
			HasNext    bool `json:"has_next"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &votes)
	if votes.Pagination.TotalCount != 2 || !votes.Pagination.HasNext || len(votes.Votes) != 1 {
		t.Fatalf("unexpected votes pagination: %+v", votes.Pagination)
	}
}

func TestChallengesLevelFilterIntegration(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")
	user1 := createUser(t, env, "lv-user1@example.com", "lv-user1", "pass", models.UserRole)
	user2 := createUser(t, env, "lv-user2@example.com", "lv-user2", "pass", models.UserRole)
	user3 := createUser(t, env, "lv-user3@example.com", "lv-user3", "pass", models.UserRole)
	user1Access, _, _ := loginUser(t, env.router, user1.Email, "pass")
	user2Access, _, _ := loginUser(t, env.router, user2.Email, "pass")
	user3Access, _, _ := loginUser(t, env.router, user3.Email, "pass")

	createChallengeReq := func(title, flag string) int64 {
		rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
			"title":       title,
			"description": "desc",
			"category":    "Web",
			"points":      100,
			"flag":        flag,
			"is_active":   true,
		}, authHeader(adminAccess))
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
		}

		var created struct {
			ID int64 `json:"id"`
		}
		decodeJSON(t, rec, &created)

		return created.ID
	}

	unknownID := createChallengeReq("Unknown Challenge", "flag{unknown}")
	level7ID := createChallengeReq("Level Seven Challenge", "flag{lv7}")
	level6ID := createChallengeReq("Level Six Challenge", "flag{lv6}")

	createSubmission(t, env, user1.ID, level7ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, level7ID, true, time.Now().UTC())
	createSubmission(t, env, user3.ID, level7ID, true, time.Now().UTC())
	createSubmission(t, env, user1.ID, level6ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, level6ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level7ID)+"/vote", map[string]any{"level": 7}, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level7ID)+"/vote", map[string]any{"level": 7}, authHeader(user2Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level7ID)+"/vote", map[string]any{"level": 6}, authHeader(user3Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level6ID)+"/vote", map[string]any{"level": 6}, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level6ID)+"/vote", map[string]any{"level": 7}, authHeader(user2Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(level6ID)+"/vote", map[string]any{"level": 6}, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", rec.Code, rec.Body.String())
	}

	assertOnlyChallenge := func(path string, wantID int64) {
		rec := doRequest(t, env.router, http.MethodGet, path, nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("filter status %d for %s: %s", rec.Code, path, rec.Body.String())
		}

		var resp struct {
			Challenges []struct {
				ID int64 `json:"id"`
			} `json:"challenges"`
		}
		decodeJSON(t, rec, &resp)
		if len(resp.Challenges) != 1 || resp.Challenges[0].ID != wantID {
			t.Fatalf("unexpected filter result for %s: %+v", path, resp.Challenges)
		}
	}

	assertOnlyChallenge("/api/challenges?level=0&page=1&page_size=50", unknownID)
	assertOnlyChallenge("/api/challenges?level=7&page=1&page_size=50", level7ID)
	assertOnlyChallenge("/api/challenges?level=6&page=1&page_size=50", level6ID)
	assertOnlyChallenge("/api/challenges/search?q=Challenge&level=7&page=1&page_size=50", level7ID)

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges?level=abc&page=1&page_size=20", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid level, got %d", rec.Code)
	}
}
