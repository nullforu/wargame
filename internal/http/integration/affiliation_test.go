package http_test

import (
	"net/http"
	"testing"
	"wargame/internal/models"
)

func TestAffiliationsAndProfileUpdate(t *testing.T) {
	env := setupTest(t, testCfg)

	admin := ensureAdminUser(t, env)
	adminToken, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	createRec := doRequest(t, env.router, http.MethodPost, "/api/admin/affiliations", map[string]string{"name": "Blue Team"}, authHeader(adminToken))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create affiliation status %d: %s", createRec.Code, createRec.Body.String())
	}

	var created struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	decodeJSON(t, createRec, &created)
	if created.ID <= 0 || created.Name != "Blue Team" {
		t.Fatalf("unexpected create response: %+v", created)
	}

	dupRec := doRequest(t, env.router, http.MethodPost, "/api/admin/affiliations", map[string]string{"name": "blue team"}, authHeader(adminToken))
	if dupRec.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate create failure, got %d", dupRec.Code)
	}

	listRec := doRequest(t, env.router, http.MethodGet, "/api/affiliations?page=1&page_size=10", nil, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list affiliations status %d: %s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Affiliations []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"affiliations"`
	}
	decodeJSON(t, listRec, &listResp)
	if len(listResp.Affiliations) != 1 || listResp.Affiliations[0].Name != "Blue Team" {
		t.Fatalf("unexpected affiliation list: %+v", listResp.Affiliations)
	}

	searchRec := doRequest(t, env.router, http.MethodGet, "/api/affiliations/search?q=blue&page=1&page_size=10", nil, nil)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search affiliations status %d: %s", searchRec.Code, searchRec.Body.String())
	}

	var searchResp struct {
		Affiliations []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"affiliations"`
	}
	decodeJSON(t, searchRec, &searchResp)
	if len(searchResp.Affiliations) != 1 || searchResp.Affiliations[0].Name != "Blue Team" {
		t.Fatalf("unexpected search affiliation list: %+v", searchResp.Affiliations)
	}

	searchMissingQRec := doRequest(t, env.router, http.MethodGet, "/api/affiliations/search?page=1&page_size=10", nil, nil)
	if searchMissingQRec.Code != http.StatusBadRequest {
		t.Fatalf("search missing q status %d: %s", searchMissingQRec.Code, searchMissingQRec.Body.String())
	}

	searchBadPageRec := doRequest(t, env.router, http.MethodGet, "/api/affiliations/search?q=blue&page=bad&page_size=10", nil, nil)
	if searchBadPageRec.Code != http.StatusBadRequest {
		t.Fatalf("search bad page status %d: %s", searchBadPageRec.Code, searchBadPageRec.Body.String())
	}

	user := createUser(t, env, "aff-user@example.com", "aff-user", "pass", models.UserRole)
	userToken, _, _ := loginUser(t, env.router, user.Email, "pass")

	updateRec := doRequest(t, env.router, http.MethodPut, "/api/me", map[string]any{"affiliation_id": created.ID}, authHeader(userToken))
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update affiliation status %d: %s", updateRec.Code, updateRec.Body.String())
	}

	var meResp struct {
		AffiliationID *int64  `json:"affiliation_id"`
		Affiliation   *string `json:"affiliation"`
	}
	decodeJSON(t, updateRec, &meResp)
	if meResp.AffiliationID == nil || *meResp.AffiliationID != created.ID {
		t.Fatalf("unexpected updated affiliation id: %+v", meResp)
	}

	if meResp.Affiliation == nil || *meResp.Affiliation != "Blue Team" {
		t.Fatalf("unexpected updated affiliation name: %+v", meResp)
	}

	usersRec := doRequest(t, env.router, http.MethodGet, "/api/affiliations/"+itoa(created.ID)+"/users?page=1&page_size=10", nil, nil)
	if usersRec.Code != http.StatusOK {
		t.Fatalf("affiliation users status %d: %s", usersRec.Code, usersRec.Body.String())
	}

	var usersResp struct {
		Users []struct {
			ID int64 `json:"id"`
		} `json:"users"`
	}
	decodeJSON(t, usersRec, &usersResp)
	if len(usersResp.Users) != 1 || usersResp.Users[0].ID != user.ID {
		t.Fatalf("unexpected affiliation users: %+v", usersResp.Users)
	}

	clearRec := doRequest(t, env.router, http.MethodPut, "/api/me", map[string]any{"affiliation_id": nil}, authHeader(userToken))
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear affiliation status %d: %s", clearRec.Code, clearRec.Body.String())
	}

	var clearResp struct {
		AffiliationID *int64 `json:"affiliation_id"`
	}
	decodeJSON(t, clearRec, &clearResp)
	if clearResp.AffiliationID != nil {
		t.Fatalf("expected nil affiliation after clear, got %+v", clearResp)
	}
}
