package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"wargame/internal/models"
	"wargame/internal/service"
)

func TestListChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createChallenge(t, env, "Active 1", 100, "flag{1}", true)
	_ = createChallenge(t, env, "Inactive", 50, "flag{2}", false)
	_ = createChallenge(t, env, "Active 2", 200, "flag{3}", true)

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		WargameState string           `json:"wargame_state"`
		Challenges   []map[string]any `json:"challenges"`
	}
	decodeJSON(t, rec, &resp)

	if resp.WargameState != string(service.WargameStateActive) {
		t.Fatalf("expected wargame_state active, got %s", resp.WargameState)
	}

	if len(resp.Challenges) != 3 {
		t.Fatalf("expected 3 challenges, got %d", len(resp.Challenges))
	}

	expectedTitles := []string{"Active 1", "Inactive", "Active 2"}
	expectedActive := []bool{true, false, true}
	expectedCategories := []string{"Misc", "Misc", "Misc"}

	for i, row := range resp.Challenges {
		if row["title"] != expectedTitles[i] {
			t.Fatalf("expected title %q, got %q", expectedTitles[i], row["title"])
		}

		if row["category"] != expectedCategories[i] {
			t.Fatalf("expected category %q, got %q", expectedCategories[i], row["category"])
		}

		if isActive, ok := row["is_active"].(bool); !ok || isActive != expectedActive[i] {
			t.Fatalf("expected is_active to be %v for %q, got %v", expectedActive[i], row["title"], isActive)
		}
	}
}

func TestChallengesLockedFlow(t *testing.T) {
	env := setupTest(t, testCfg)
	access, _, userID := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
	prev := createChallenge(t, env, "Prev", 50, "flag{prev}", true)
	locked := createChallenge(t, env, "Locked", 100, "flag{lock}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Challenges []map[string]any `json:"challenges"`
	}
	decodeJSON(t, rec, &listResp)

	var lockedRow map[string]any
	for _, row := range listResp.Challenges {
		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			lockedRow = row
		}
	}

	if lockedRow == nil {
		t.Fatalf("expected locked challenge in list")
	}

	if lockedRow["is_locked"] != true {
		t.Fatalf("expected is_locked true, got %v", lockedRow["is_locked"])
	}

	if lockedRow["category"] != locked.Category {
		t.Fatalf("expected locked category %q, got %v", locked.Category, lockedRow["category"])
	}

	if lockedRow["initial_points"] == nil || lockedRow["minimum_points"] == nil || lockedRow["solve_count"] == nil {
		t.Fatalf("expected locked response to include points metadata")
	}

	if lockedRow["is_active"] != locked.IsActive {
		t.Fatalf("expected is_active %v, got %v", locked.IsActive, lockedRow["is_active"])
	}

	if prevID, ok := lockedRow["previous_challenge_id"].(float64); !ok || int64(prevID) != prev.ID {
		t.Fatalf("expected previous_challenge_id %d, got %v", prev.ID, lockedRow["previous_challenge_id"])
	}

	if lockedRow["previous_challenge_title"] != prev.Title {
		t.Fatalf("expected previous_challenge_title %q, got %v", prev.Title, lockedRow["previous_challenge_title"])
	}

	if lockedRow["previous_challenge_category"] != prev.Category {
		t.Fatalf(
			"expected previous_challenge_category %q, got %v",
			prev.Category,
			lockedRow["previous_challenge_category"],
		)
	}

	if _, ok := lockedRow["description"]; ok {
		t.Fatalf("expected description to be omitted for locked challenge")
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("list auth status %d: %s", rec.Code, rec.Body.String())
	}

	listResp = struct {
		Challenges []map[string]any `json:"challenges"`
	}{}
	decodeJSON(t, rec, &listResp)

	lockedRow = nil
	for _, row := range listResp.Challenges {
		if id, ok := row["id"].(float64); ok && int64(id) == locked.ID {
			lockedRow = row
		}
	}

	if lockedRow == nil || lockedRow["is_locked"] != true {
		t.Fatalf("expected locked challenge for unsolved user")
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(locked.ID)+"/submit", map[string]string{"flag": "flag{lock}"}, authHeader(access))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("submit locked status %d: %s", rec.Code, rec.Body.String())
	}

	createSubmission(t, env, userID, prev.ID, true, time.Now().UTC())

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(locked.ID)+"/submit", map[string]string{"flag": "flag{lock}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit unlocked status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitFlag(t *testing.T) {
	t.Run("missing auth", func(t *testing.T) {
		env := setupTest(t, testCfg)
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/abc/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidInput.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{}, authHeader(access))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{"flag": "required"})
	})

	t.Run("challenge not found", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/999/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrChallengeNotFound.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("inactive challenge", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", false)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("correct and wrong", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var wrongResp struct {
			Correct bool `json:"correct"`
		}
		decodeJSON(t, rec, &wrongResp)

		if wrongResp.Correct {
			t.Fatalf("expected incorrect flag")
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var correctResp struct {
			Correct bool `json:"correct"`
		}
		decodeJSON(t, rec, &correctResp)

		if !correctResp.Correct {
			t.Fatalf("expected correct flag")
		}
	})

	t.Run("already solved", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrAlreadySolved.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		for i := 0; i < env.cfg.Security.SubmissionMax; i++ {
			rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d at attempt %d: %s", rec.Code, i+1, rec.Body.String())
			}
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrRateLimited.Error() || resp.RateLimit == nil {
			t.Fatalf("unexpected rate limit response: %+v", resp)
		}

		if resp.RateLimit.Limit != env.cfg.Security.SubmissionMax || resp.RateLimit.Remaining != 0 {
			t.Fatalf("unexpected rate limit info: %+v", resp.RateLimit)
		}

		if rec.Header().Get("X-RateLimit-Limit") == "" || rec.Header().Get("X-RateLimit-Remaining") == "" || rec.Header().Get("X-RateLimit-Reset") == "" {
			t.Fatalf("missing rate limit headers")
		}
	})
}

func TestChallengesBeforeStart(t *testing.T) {
	env := setupTest(t, testCfg)
	start := time.Now().Add(2 * time.Hour)
	end := time.Now().Add(4 * time.Hour)
	setWargameWindow(t, env, &start, &end)

	challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
	access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		WargameState string `json:"wargame_state"`
	}
	decodeJSON(t, rec, &listResp)
	if listResp.WargameState != string(service.WargameStateNotStarted) {
		t.Fatalf("expected wargame_state not_started, got %s", listResp.WargameState)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	var submitResp map[string]any
	decodeJSON(t, rec, &submitResp)
	if submitResp["wargame_state"] != string(service.WargameStateNotStarted) {
		t.Fatalf("expected wargame_state not_started, got %v", submitResp["wargame_state"])
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/file/download", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", rec.Code, rec.Body.String())
	}

	var downloadResp map[string]any
	decodeJSON(t, rec, &downloadResp)
	if downloadResp["wargame_state"] != string(service.WargameStateNotStarted) {
		t.Fatalf("expected wargame_state not_started, got %v", downloadResp["wargame_state"])
	}
}

func TestChallengesAfterEnd(t *testing.T) {
	env := setupTest(t, testCfg)
	end := time.Now().Add(-2 * time.Hour)
	setWargameWindow(t, env, nil, &end)

	challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
	access, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		WargameState string `json:"wargame_state"`
		Challenges   []any  `json:"challenges"`
	}
	decodeJSON(t, rec, &listResp)
	if listResp.WargameState != string(service.WargameStateEnded) {
		t.Fatalf("expected wargame_state ended, got %s", listResp.WargameState)
	}
	if len(listResp.Challenges) != 1 {
		t.Fatalf("expected challenges after end, got %d", len(listResp.Challenges))
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	var submitResp map[string]any
	decodeJSON(t, rec, &submitResp)
	if submitResp["wargame_state"] != string(service.WargameStateEnded) {
		t.Fatalf("expected wargame_state ended, got %v", submitResp["wargame_state"])
	}
}

func TestChallengesDynamicScoring(t *testing.T) {
	env := setupTest(t, testCfg)
	userA := createUser(t, env, "usera@example.com", "usera", "pass123", models.UserRole)
	userSolo := createUser(t, env, "solo@example.com", "solo", "pass123", models.UserRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "flag{dynamic}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	login := func(email, password string) string {
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			AccessToken string `json:"access_token"`
		}
		decodeJSON(t, rec, &resp)

		return resp.AccessToken
	}

	accessA := login(userA.Email, "pass123")
	accessSolo := login(userSolo.Email, "pass123")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{dynamic}"}, authHeader(accessA))
	if rec.Code != http.StatusOK {
		t.Fatalf("first user submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{dynamic}"}, authHeader(accessSolo))
	if rec.Code != http.StatusOK {
		t.Fatalf("solo submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		WargameState string           `json:"wargame_state"`
		Challenges   []map[string]any `json:"challenges"`
	}
	decodeJSON(t, rec, &resp)

	if len(resp.Challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(resp.Challenges))
	}

	row := resp.Challenges[0]
	if row["points"].(float64) != 100 {
		t.Fatalf("expected dynamic points 100, got %v", row["points"])
	}

	if row["solve_count"].(float64) != 2 {
		t.Fatalf("expected solve_count 2, got %v", row["solve_count"])
	}
}

func TestChallengeFileUploadDownloadDelete(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	uploadRec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file/upload",
		map[string]string{"filename": "bundle.zip"},
		authHeader(access),
	)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	var uploadResp struct {
		Challenge struct {
			ID       int64   `json:"id"`
			HasFile  bool    `json:"has_file"`
			FileName *string `json:"file_name"`
		} `json:"challenge"`
		Upload struct {
			URL    string            `json:"url"`
			Fields map[string]string `json:"fields"`
		} `json:"upload"`
	}

	decodeJSON(t, uploadRec, &uploadResp)
	if !uploadResp.Challenge.HasFile {
		t.Fatalf("expected has_file true")
	}

	if uploadResp.Challenge.FileName == nil || *uploadResp.Challenge.FileName != "bundle.zip" {
		t.Fatalf("expected file_name bundle.zip, got %v", uploadResp.Challenge.FileName)
	}

	if uploadResp.Upload.URL == "" || len(uploadResp.Upload.Fields) == 0 {
		t.Fatalf("expected upload payload")
	}

	downloadRec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/challenges/"+itoa(challenge.ID)+"/file/download",
		nil,
		authHeader(access),
	)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("download status %d: %s", downloadRec.Code, downloadRec.Body.String())
	}

	deleteRec := doRequest(
		t,
		env.router,
		http.MethodDelete,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file",
		nil,
		authHeader(access),
	)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	var deleteResp struct {
		HasFile bool `json:"has_file"`
	}
	decodeJSON(t, deleteRec, &deleteResp)

	if deleteResp.HasFile {
		t.Fatalf("expected has_file false after delete")
	}
}

func TestChallengeFileUploadRejectsNonZip(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	rec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/admin/challenges/"+itoa(challenge.ID)+"/file/upload",
		map[string]string{"filename": "bundle.txt"},
		authHeader(access),
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChallengeFileDownloadMissing(t *testing.T) {
	env := setupTest(t, testCfg)
	admin := ensureAdminUser(t, env)
	access, _, _ := loginUser(t, env.router, admin.Email, "adminpass")

	challenge := createChallenge(t, env, "FileChallenge", 100, "flag{file}", true)

	rec := doRequest(
		t,
		env.router,
		http.MethodPost,
		"/api/challenges/"+itoa(challenge.ID)+"/file/download",
		nil,
		authHeader(access),
	)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
