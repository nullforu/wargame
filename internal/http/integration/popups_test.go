package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestPopupPublicAndAdminEndpoints(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	rec := doRequest(t, env.router, http.MethodGet, "/api/popups/active", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("active empty status %d: %s", rec.Code, rec.Body.String())
	}

	var activeEmpty struct {
		Popups []struct {
			ID int64 `json:"id"`
		} `json:"popups"`
	}
	decodeJSON(t, rec, &activeEmpty)
	if len(activeEmpty.Popups) != 0 {
		t.Fatalf("expected no active popups, got %+v", activeEmpty)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice 1", "is_active": true}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized create, got %d: %s", rec.Code, rec.Body.String())
	}

	userAccess, _, _ := registerAndLogin(t, env, "user@example.com", "user", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice 1", "is_active": true}, authHeader(userAccess))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden create, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice 1", "is_active": true}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected active create without image to fail, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice 1", "link_url": "https://example.com/notice-1", "is_active": false}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}

	var created1 struct {
		ID       int64   `json:"id"`
		Title    string  `json:"title"`
		LinkURL  *string `json:"link_url"`
		IsActive bool    `json:"is_active"`
	}
	decodeJSON(t, rec, &created1)
	if created1.ID <= 0 || created1.Title != "Notice 1" || created1.LinkURL == nil || *created1.LinkURL != "https://example.com/notice-1" || created1.IsActive {
		t.Fatalf("unexpected created popup: %+v", created1)
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice 2", "is_active": false}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create second status %d: %s", rec.Code, rec.Body.String())
	}

	var created2 struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, rec, &created2)

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups/"+itoa(created2.ID)+"/image/upload", map[string]string{"filename": "notice.webp"}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", rec.Code, rec.Body.String())
	}

	var uploadResp struct {
		Upload struct {
			Fields map[string]string `json:"fields"`
		} `json:"upload"`
	}
	decodeJSON(t, rec, &uploadResp)
	key2 := uploadResp.Upload.Fields["key"]
	if key2 == "" {
		t.Fatalf("expected upload key, got %+v", uploadResp)
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/popups/"+itoa(created2.ID)+"/image", map[string]string{"key": key2, "filename": "notice.webp"}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("finalize status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/popups/"+itoa(created2.ID), map[string]any{"is_active": true}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("activate status %d: %s", rec.Code, rec.Body.String())
	}

	key1 := "popups/manual.png"
	name1 := "manual.png"
	row1, err := env.popupRepo.GetByID(context.Background(), created1.ID)
	if err != nil {
		t.Fatalf("get popup1: %v", err)
	}

	row1.ImageKey = &key1
	row1.ImageName = &name1
	row1.IsActive = true
	row1.CreatedAt = time.Now().UTC().Add(-time.Hour)
	if err := env.popupRepo.Update(context.Background(), row1); err != nil {
		t.Fatalf("update popup1 image: %v", err)
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/popups/active", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("active status %d: %s", rec.Code, rec.Body.String())
	}

	var activeResp struct {
		Popups []struct {
			ID       int64   `json:"id"`
			ImageKey *string `json:"image_key"`
		} `json:"popups"`
	}
	decodeJSON(t, rec, &activeResp)
	if len(activeResp.Popups) != 2 || activeResp.Popups[0].ID != created2.ID || activeResp.Popups[1].ID != created1.ID {
		t.Fatalf("expected active popups newest first, got %+v", activeResp.Popups)
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/admin/popups/"+itoa(created1.ID), map[string]any{"title": "Updated", "link_url": "https://example.com/updated", "is_active": false}, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/admin/popups/"+itoa(created2.ID), nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPopupAdminValidation(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": " "}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for blank title, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups", map[string]any{"title": "Notice", "link_url": "ftp://example.com"}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid link, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/popups/999999/image/upload", map[string]string{"filename": "notice.gif"}, authHeader(adminAccess))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found for missing popup, got %d: %s", rec.Code, rec.Body.String())
	}
}
