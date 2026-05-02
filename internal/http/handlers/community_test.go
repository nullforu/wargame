package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"wargame/internal/models"
)

func TestHandlerCommunityHandlers(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin-community-handler@example.com", "admin-community-handler", "pass", models.AdminRole)
	user := createHandlerUser(t, env, "user-community-handler@example.com", "user-community-handler", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/community", []byte(`{"category":0,"title":"n","content":"c"}`))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.CreateCommunityPost(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user notice create, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community", []byte(`{"category":1,"title":"free","content":"body"}`))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.CreateCommunityPost(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for free create, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created communityPostResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created free: %v", err)
	}

	ctx, rec = newJSONContext(t, http.MethodPatch, "/api/community/"+toStringID(created.ID), []byte(`{"category":0}`))
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(created.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.UpdateCommunityPost(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for promote to notice by user, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPatch, "/api/community/"+toStringID(created.ID), []byte(`{"title":"free updated"}`))
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(created.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.UpdateCommunityPost(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner update, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community", []byte(`{"category":0,"title":"notice","content":"body"}`))
	ctx.Set("userID", admin.ID)
	ctx.Set("role", models.AdminRole)
	env.handler.CreateCommunityPost(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin notice create, got %d body=%s", rec.Code, rec.Body.String())
	}
	var notice communityPostResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &notice); err != nil {
		t.Fatalf("decode notice: %v", err)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community?page=1&page_size=20&sort=popular", nil)
	env.handler.ListCommunityPosts(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/"+toStringID(notice.ID), nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	env.handler.GetCommunityPost(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for get detail, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community/"+toStringID(notice.ID)+"/likes", nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.ToggleCommunityPostLike(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for like toggle, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/"+toStringID(notice.ID)+"/likes?page=1&page_size=20", nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	env.handler.CommunityPostLikes(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for likes list, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/community/"+toStringID(notice.ID), nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.DeleteCommunityPost(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user notice delete, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/community/"+toStringID(notice.ID), nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	ctx.Set("userID", admin.ID)
	ctx.Set("role", models.AdminRole)
	env.handler.DeleteCommunityPost(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin notice delete, got %d", rec.Code)
	}
}
