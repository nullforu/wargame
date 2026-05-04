package http_test

import (
	"context"
	"net/http"
	"strings"
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
		CategoryCounts []struct {
			Category string `json:"category"`
			Count    int    `json:"count"`
		} `json:"category_counts"`
		LevelCounts []struct {
			Level int `json:"level"`
			Count int `json:"count"`
		} `json:"level_counts"`
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

	if len(pagedResp.CategoryCounts) != 2 || pagedResp.CategoryCounts[0].Category != "Crypto" || pagedResp.CategoryCounts[0].Count != 1 || pagedResp.CategoryCounts[1].Category != "Web" || pagedResp.CategoryCounts[1].Count != 2 {
		t.Fatalf("unexpected category counts: %+v", pagedResp.CategoryCounts)
	}

	if len(pagedResp.LevelCounts) != 1 || pagedResp.LevelCounts[0].Level != models.UnknownLevel || pagedResp.LevelCounts[0].Count != 3 {
		t.Fatalf("unexpected level counts: %+v", pagedResp.LevelCounts)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=web&page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec.Code, rec.Body.String())
	}

	var searchResp struct {
		Challenges []struct {
			Title string `json:"title"`
		} `json:"challenges"`
		CategoryCounts []struct {
			Category string `json:"category"`
			Count    int    `json:"count"`
		} `json:"category_counts"`
		LevelCounts []struct {
			Level int `json:"level"`
			Count int `json:"count"`
		} `json:"level_counts"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	decodeJSON(t, rec, &searchResp)
	if len(searchResp.Challenges) != 2 || searchResp.Pagination.TotalCount != 2 {
		t.Fatalf("unexpected search response: %+v", searchResp)
	}

	if len(searchResp.CategoryCounts) != 2 || searchResp.CategoryCounts[0].Category != "Crypto" || searchResp.CategoryCounts[1].Category != "Web" {
		t.Fatalf("unexpected search category counts: %+v", searchResp.CategoryCounts)
	}

	if len(searchResp.LevelCounts) != 1 || searchResp.LevelCounts[0].Level != models.UnknownLevel || searchResp.LevelCounts[0].Count != 3 {
		t.Fatalf("unexpected search level counts: %+v", searchResp.LevelCounts)
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
	authorAffiliation := createAffiliation(t, env, "Challenge Authors")
	adminUser, err := env.userRepo.GetByEmail(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("get admin user: %v", err)
	}
	authorBio := "challenge author bio"
	adminUser.AffiliationID = &authorAffiliation.ID
	adminUser.Bio = &authorBio
	if err := env.userRepo.Update(context.Background(), adminUser); err != nil {
		t.Fatalf("update admin affiliation: %v", err)
	}
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
		ID              int64                   `json:"id"`
		CreatedAt       string                  `json:"created_at"`
		Level           int                     `json:"level"`
		LevelVoteCounts []models.LevelVoteCount `json:"level_vote_counts"`
		IsSolved        bool                    `json:"is_solved"`
		FirstBlood      *struct {
			UserID       int64  `json:"user_id"`
			Username     string `json:"username"`
			IsFirstBlood bool   `json:"is_first_blood"`
		} `json:"first_blood"`
		CreatedBy struct {
			UserID        int64   `json:"user_id"`
			Username      string  `json:"username"`
			AffiliationID *int64  `json:"affiliation_id"`
			Affiliation   *string `json:"affiliation"`
			Bio           *string `json:"bio"`
		} `json:"created_by"`
	}
	decodeJSON(t, rec, &detail)
	if detail.ID != created[0].ID || detail.Level != models.UnknownLevel || !detail.IsSolved || len(detail.LevelVoteCounts) != 0 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.FirstBlood == nil || detail.FirstBlood.UserID != user.ID || !detail.FirstBlood.IsFirstBlood {
		t.Fatalf("expected first_blood in detail response: %+v", detail)
	}
	if detail.CreatedBy.UserID <= 0 || detail.CreatedBy.Username == "" {
		t.Fatalf("expected creator info in detail response: %+v", detail)
	}
	if detail.CreatedAt == "" {
		t.Fatalf("expected created_at in detail response: %+v", detail)
	}
	if detail.CreatedBy.AffiliationID == nil || *detail.CreatedBy.AffiliationID != authorAffiliation.ID {
		t.Fatalf("expected creator affiliation id in detail response: %+v", detail)
	}
	if detail.CreatedBy.Affiliation == nil || *detail.CreatedBy.Affiliation != authorAffiliation.Name {
		t.Fatalf("expected creator affiliation in detail response: %+v", detail)
	}
	if detail.CreatedBy.Bio == nil || *detail.CreatedBy.Bio != authorBio {
		t.Fatalf("expected creator bio in detail response: %+v", detail)
	}

	other1 := createUser(t, env, "solver1@example.com", "solver1", "pass", models.UserRole)
	other2 := createUser(t, env, "solver2@example.com", "solver2", "pass", models.UserRole)
	solver1Bio := "solver1 bio"
	solver2Bio := "solver2 bio"
	other1.Bio = &solver1Bio
	other2.Bio = &solver2Bio
	if err := env.userRepo.Update(context.Background(), other1); err != nil {
		t.Fatalf("update solver1 bio: %v", err)
	}
	if err := env.userRepo.Update(context.Background(), other2); err != nil {
		t.Fatalf("update solver2 bio: %v", err)
	}
	createSubmission(t, env, other1.ID, created[0].ID, true, time.Now().UTC().Add(1*time.Minute))
	createSubmission(t, env, other2.ID, created[0].ID, true, time.Now().UTC().Add(2*time.Minute))
	createSubmission(t, env, user.ID, created[0].ID, true, time.Now().UTC().Add(3*time.Minute))

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created[0].ID)+"/solvers?page=1&page_size=2", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("solvers status %d: %s", rec.Code, rec.Body.String())
	}

	var solvers struct {
		Solvers []struct {
			UserID int64   `json:"user_id"`
			Bio    *string `json:"bio"`
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
	if solvers.Solvers[0].UserID != other2.ID || solvers.Solvers[1].UserID != other1.ID {
		t.Fatalf("expected latest solvers first, got %+v", solvers.Solvers)
	}
	foundExpectedBio := false
	for _, solver := range solvers.Solvers {
		if solver.Bio != nil && (*solver.Bio == solver1Bio || *solver.Bio == solver2Bio) {
			foundExpectedBio = true
			break
		}
	}
	if !foundExpectedBio {
		t.Fatalf("expected solver bio in response page: %+v", solvers.Solvers)
	}
}

func TestChallengesFilterByUnknownLevel(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	_ = createUser(t, env, "player@example.com", "player", "playerpass", models.UserRole)
	userAccess, _, _ := loginUser(t, env.router, "player@example.com", "playerpass")

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

	unknownID := createChallengeReq("Unknown Only", "flag{unknown}")
	votedID := createChallengeReq("Known Level", "flag{known}")

	// Vote one challenge to give it a non-unknown representative level.
	submitRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(votedID)+"/submit", map[string]string{"flag": "flag{known}"}, authHeader(userAccess))
	if submitRec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", submitRec.Code, submitRec.Body.String())
	}
	voteRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(votedID)+"/vote", map[string]int{"level": 7}, authHeader(userAccess))
	if voteRec.Code != http.StatusOK {
		t.Fatalf("vote status %d: %s", voteRec.Code, voteRec.Body.String())
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges?level=0&page=1&page_size=20", nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("list unknown level status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
	}
	decodeJSON(t, rec, &listResp)

	foundUnknown := false
	foundVoted := false
	for _, item := range listResp.Challenges {
		if item.ID == unknownID {
			foundUnknown = true
		}
		if item.ID == votedID {
			foundVoted = true
		}
	}
	if !foundUnknown {
		t.Fatalf("expected unknown-level challenge %d in response: %+v", unknownID, listResp.Challenges)
	}
	if foundVoted {
		t.Fatalf("did not expect voted challenge %d in unknown-level response: %+v", votedID, listResp.Challenges)
	}

	searchRec := doRequest(t, env.router, http.MethodGet, "/api/challenges/search?q=Unknown&level=0&page=1&page_size=20", nil, authHeader(userAccess))
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search unknown level status %d: %s", searchRec.Code, searchRec.Body.String())
	}

	var searchResp struct {
		Challenges []struct {
			ID int64 `json:"id"`
		} `json:"challenges"`
	}
	decodeJSON(t, searchRec, &searchResp)
	if len(searchResp.Challenges) != 1 || searchResp.Challenges[0].ID != unknownID {
		t.Fatalf("unexpected unknown-level search response: %+v", searchResp.Challenges)
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

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created.ID)+"/my-vote", nil, authHeader(user1Access))
	if rec.Code != http.StatusOK {
		t.Fatalf("my-vote status %d: %s", rec.Code, rec.Body.String())
	}

	var myVote struct {
		Level *int `json:"level"`
	}
	decodeJSON(t, rec, &myVote)
	if myVote.Level == nil || *myVote.Level != 6 {
		t.Fatalf("expected my vote level 6, got %+v", myVote.Level)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(created.ID)+"/my-vote", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", rec.Code)
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

func TestChallengeCommentsIntegration(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := createUser(t, env, "admin-cmt@example.com", "admincmt", "pass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "pass")
	user := createUser(t, env, "user-cmt@example.com", "usercmt", "pass", models.UserRole)
	userAccess, _, _ := loginUser(t, env.router, user.Email, "pass")
	other := createUser(t, env, "other-cmt@example.com", "othercmt", "pass", models.UserRole)
	otherAccess, _, _ := loginUser(t, env.router, other.Email, "pass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{"title": "Comment Int", "description": "d", "category": "Misc", "points": 100, "flag": "flag{CINT}", "is_active": true}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create challenge: %d %s", rec.Code, rec.Body.String())
	}

	var challenge struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &challenge)

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(challenge.ID)+"/challenge-comments?page=1&page_size=10", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list comments: %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/challenge-comments", map[string]any{"content": "first comment"}, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create comment: %d %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &created)

	rec = doRequest(t, env.router, http.MethodPatch, "/api/challenges/challenge-comments/"+itoa(created.ID), map[string]any{"content": "updated comment"}, authHeader(otherAccess))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden update, got %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPatch, "/api/challenges/challenge-comments/"+itoa(created.ID), map[string]any{"content": "updated comment"}, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("update comment: %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/challenges/challenge-comments/"+itoa(created.ID), nil, authHeader(otherAccess))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden delete, got %d %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/challenges/challenge-comments/"+itoa(created.ID), nil, authHeader(userAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete comment: %d %s", rec.Code, rec.Body.String())
	}
}

func TestChallengeCommentsKoreanLengthLimit(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := createUser(t, env, "admin-ko-comment@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, admin.Email, "adminpass")
	user := createUser(t, env, "user-ko-comment@example.com", "ko_comment_user", "pass", models.UserRole)
	userAccess, _, _ := loginUser(t, env.router, user.Email, "pass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "KO Comment Length",
		"description": "desc",
		"category":    "Misc",
		"points":      100,
		"flag":        "flag{KOC}",
		"is_active":   true,
	}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create challenge status %d: %s", rec.Code, rec.Body.String())
	}

	var challenge struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &challenge)

	ok500 := strings.Repeat("가", 500)
	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/challenge-comments", map[string]any{"content": ok500}, authHeader(userAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for 500 korean chars, got %d: %s", rec.Code, rec.Body.String())
	}

	tooLong501 := strings.Repeat("가", 501)
	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/challenge-comments", map[string]any{"content": tooLong501}, authHeader(userAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for 501 korean chars, got %d: %s", rec.Code, rec.Body.String())
	}
}
