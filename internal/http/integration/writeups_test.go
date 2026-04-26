package http_test

import (
	"net/http"
	"testing"
)

func TestWriteupFlowAndVisibility(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	writer := createUser(t, env, "writer@example.com", "writer", "pass", "user")
	viewer := createUser(t, env, "viewer@example.com", "viewer", "pass", "user")

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	writerAccess, _, _ := loginUser(t, env.router, writer.Email, "pass")
	viewerAccess, _, _ := loginUser(t, env.router, viewer.Email, "pass")

	createChallengeRec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]any{
		"title":       "Writeup Target",
		"description": "desc",
		"category":    "Web",
		"points":      400,
		"flag":        "flag{writeup-target}",
		"is_active":   true,
	}, authHeader(adminAccess))
	if createChallengeRec.Code != http.StatusCreated {
		t.Fatalf("create challenge status %d: %s", createChallengeRec.Code, createChallengeRec.Body.String())
	}

	var createdChallenge struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, createChallengeRec, &createdChallenge)

	createBeforeSolve := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(createdChallenge.ID)+"/writeups", map[string]any{
		"content": "Nope",
	}, authHeader(writerAccess))
	if createBeforeSolve.Code != http.StatusForbidden {
		t.Fatalf("expected 403 before solve, got %d body=%s", createBeforeSolve.Code, createBeforeSolve.Body.String())
	}

	submitRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(createdChallenge.ID)+"/submit", map[string]any{
		"flag": "flag{writeup-target}",
	}, authHeader(writerAccess))
	if submitRec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", submitRec.Code, submitRec.Body.String())
	}

	createWriteupRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(createdChallenge.ID)+"/writeups", map[string]any{
		"content": "# Secret\n\nVery secret content",
	}, authHeader(writerAccess))
	if createWriteupRec.Code != http.StatusCreated {
		t.Fatalf("create writeup status %d: %s", createWriteupRec.Code, createWriteupRec.Body.String())
	}

	var createdWriteup struct {
		ID      int64   `json:"id"`
		Content *string `json:"content"`
	}
	decodeJSON(t, createWriteupRec, &createdWriteup)
	if createdWriteup.ID <= 0 || createdWriteup.Content == nil || *createdWriteup.Content == "" {
		t.Fatalf("unexpected created writeup response: %+v", createdWriteup)
	}

	createDuplicateRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(createdChallenge.ID)+"/writeups", map[string]any{
		"content": "duplicate",
	}, authHeader(writerAccess))
	if createDuplicateRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 duplicate writeup, got %d body=%s", createDuplicateRec.Code, createDuplicateRec.Body.String())
	}

	listUnsolvedRec := doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(createdChallenge.ID)+"/writeups?page=1&page_size=10", nil, authHeader(viewerAccess))
	if listUnsolvedRec.Code != http.StatusOK {
		t.Fatalf("list unsolved status %d: %s", listUnsolvedRec.Code, listUnsolvedRec.Body.String())
	}
	var unsolvedList struct {
		Writeups []struct {
			ID      int64   `json:"id"`
			Content *string `json:"content"`
		} `json:"writeups"`
		CanViewContent bool `json:"can_view_content"`
	}
	decodeJSON(t, listUnsolvedRec, &unsolvedList)
	if unsolvedList.CanViewContent || len(unsolvedList.Writeups) != 1 || unsolvedList.Writeups[0].Content != nil {
		t.Fatalf("unexpected unsolved list response: %+v", unsolvedList)
	}

	userWriteupsUnsolvedRec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(writer.ID)+"/writeups?page=1&page_size=10", nil, authHeader(viewerAccess))
	if userWriteupsUnsolvedRec.Code != http.StatusOK {
		t.Fatalf("user writeups unsolved status %d: %s", userWriteupsUnsolvedRec.Code, userWriteupsUnsolvedRec.Body.String())
	}

	var userWriteupsUnsolved struct {
		Writeups []struct {
			Content *string `json:"content"`
		} `json:"writeups"`
	}
	decodeJSON(t, userWriteupsUnsolvedRec, &userWriteupsUnsolved)
	if len(userWriteupsUnsolved.Writeups) != 1 || userWriteupsUnsolved.Writeups[0].Content != nil {
		t.Fatalf("unexpected user writeups unsolved response: %+v", userWriteupsUnsolved)
	}

	detailUnsolvedRec := doRequest(t, env.router, http.MethodGet, "/api/writeups/"+itoa(createdWriteup.ID), nil, authHeader(viewerAccess))
	if detailUnsolvedRec.Code != http.StatusOK {
		t.Fatalf("detail unsolved status %d: %s", detailUnsolvedRec.Code, detailUnsolvedRec.Body.String())
	}

	var unsolvedDetail struct {
		CanViewContent bool `json:"can_view_content"`
		Writeup        struct {
			Content *string `json:"content"`
		} `json:"writeup"`
	}
	decodeJSON(t, detailUnsolvedRec, &unsolvedDetail)
	if unsolvedDetail.CanViewContent || unsolvedDetail.Writeup.Content != nil {
		t.Fatalf("unexpected unsolved detail response: %+v", unsolvedDetail)
	}

	submitViewerRec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(createdChallenge.ID)+"/submit", map[string]any{
		"flag": "flag{writeup-target}",
	}, authHeader(viewerAccess))
	if submitViewerRec.Code != http.StatusOK {
		t.Fatalf("viewer submit status %d: %s", submitViewerRec.Code, submitViewerRec.Body.String())
	}

	listSolvedRec := doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(createdChallenge.ID)+"/writeups?page=1&page_size=10", nil, authHeader(viewerAccess))
	if listSolvedRec.Code != http.StatusOK {
		t.Fatalf("list solved status %d: %s", listSolvedRec.Code, listSolvedRec.Body.String())
	}

	var solvedList struct {
		Writeups []struct {
			ID      int64   `json:"id"`
			Content *string `json:"content"`
		} `json:"writeups"`
		CanViewContent bool `json:"can_view_content"`
	}
	decodeJSON(t, listSolvedRec, &solvedList)
	if !solvedList.CanViewContent || len(solvedList.Writeups) != 1 || solvedList.Writeups[0].Content == nil || *solvedList.Writeups[0].Content == "" {
		t.Fatalf("unexpected solved list response: %+v", solvedList)
	}

	userWriteupsSolvedRec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(writer.ID)+"/writeups?page=1&page_size=10", nil, authHeader(viewerAccess))
	if userWriteupsSolvedRec.Code != http.StatusOK {
		t.Fatalf("user writeups solved status %d: %s", userWriteupsSolvedRec.Code, userWriteupsSolvedRec.Body.String())
	}

	var userWriteupsSolved struct {
		Writeups []struct {
			Content *string `json:"content"`
		} `json:"writeups"`
	}
	decodeJSON(t, userWriteupsSolvedRec, &userWriteupsSolved)
	if len(userWriteupsSolved.Writeups) != 1 || userWriteupsSolved.Writeups[0].Content == nil || *userWriteupsSolved.Writeups[0].Content == "" {
		t.Fatalf("unexpected user writeups solved response: %+v", userWriteupsSolved)
	}

	detailSolvedRec := doRequest(t, env.router, http.MethodGet, "/api/writeups/"+itoa(createdWriteup.ID), nil, authHeader(viewerAccess))
	if detailSolvedRec.Code != http.StatusOK {
		t.Fatalf("detail solved status %d: %s", detailSolvedRec.Code, detailSolvedRec.Body.String())
	}

	var solvedDetail struct {
		CanViewContent bool `json:"can_view_content"`
		Writeup        struct {
			Content *string `json:"content"`
		} `json:"writeup"`
	}
	decodeJSON(t, detailSolvedRec, &solvedDetail)
	if !solvedDetail.CanViewContent || solvedDetail.Writeup.Content == nil || *solvedDetail.Writeup.Content == "" {
		t.Fatalf("unexpected solved detail response: %+v", solvedDetail)
	}

	updateByOtherRec := doRequest(t, env.router, http.MethodPatch, "/api/writeups/"+itoa(createdWriteup.ID), map[string]any{
		"content": "hacked",
	}, authHeader(viewerAccess))
	if updateByOtherRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner update, got %d body=%s", updateByOtherRec.Code, updateByOtherRec.Body.String())
	}

	updateByOwnerRec := doRequest(t, env.router, http.MethodPatch, "/api/writeups/"+itoa(createdWriteup.ID), map[string]any{
		"content": "updated body",
	}, authHeader(writerAccess))
	if updateByOwnerRec.Code != http.StatusOK {
		t.Fatalf("owner update status %d: %s", updateByOwnerRec.Code, updateByOwnerRec.Body.String())
	}

	var updated struct {
		Content *string `json:"content"`
	}
	decodeJSON(t, updateByOwnerRec, &updated)
	if updated.Content == nil || *updated.Content != "updated body" {
		t.Fatalf("unexpected owner update response: %+v", updated)
	}

	myWriteupsRec := doRequest(t, env.router, http.MethodGet, "/api/me/writeups?page=1&page_size=1", nil, authHeader(writerAccess))
	if myWriteupsRec.Code != http.StatusOK {
		t.Fatalf("my writeups status %d: %s", myWriteupsRec.Code, myWriteupsRec.Body.String())
	}

	var myWriteups struct {
		Writeups []struct {
			ID int64 `json:"id"`
		} `json:"writeups"`
		Pagination struct {
			PageSize   int `json:"page_size"`
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	decodeJSON(t, myWriteupsRec, &myWriteups)
	if len(myWriteups.Writeups) != 1 || myWriteups.Pagination.PageSize != 1 || myWriteups.Pagination.TotalCount != 1 {
		t.Fatalf("unexpected my writeups response: %+v", myWriteups)
	}

	deleteByOtherRec := doRequest(t, env.router, http.MethodDelete, "/api/writeups/"+itoa(createdWriteup.ID), nil, authHeader(viewerAccess))
	if deleteByOtherRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner delete, got %d body=%s", deleteByOtherRec.Code, deleteByOtherRec.Body.String())
	}

	deleteByOwnerRec := doRequest(t, env.router, http.MethodDelete, "/api/writeups/"+itoa(createdWriteup.ID), nil, authHeader(writerAccess))
	if deleteByOwnerRec.Code != http.StatusOK {
		t.Fatalf("owner delete status %d: %s", deleteByOwnerRec.Code, deleteByOwnerRec.Body.String())
	}

	deletedDetailRec := doRequest(t, env.router, http.MethodGet, "/api/writeups/"+itoa(createdWriteup.ID), nil, authHeader(writerAccess))
	if deletedDetailRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted writeup, got %d body=%s", deletedDetailRec.Code, deletedDetailRec.Body.String())
	}
}
