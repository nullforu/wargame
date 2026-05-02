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

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community/"+toStringID(notice.ID)+"/comments", []byte(`{"content":"first comment"}`))
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.CreateCommunityComment(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for comment create, got %d body=%s", rec.Code, rec.Body.String())
	}

	var createdComment communityCommentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &createdComment); err != nil {
		t.Fatalf("decode created comment: %v", err)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/"+toStringID(notice.ID)+"/comments?page=1&page_size=20", nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(notice.ID)))
	env.handler.CommunityComments(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for comments list, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodPatch, "/api/community/comments/"+toStringID(createdComment.ID), []byte(`{"content":"updated comment"}`))
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(createdComment.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.UpdateCommunityComment(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for comment update, got %d body=%s", rec.Code, rec.Body.String())
	}

	ctx, rec = newJSONContext(t, http.MethodDelete, "/api/community/comments/"+toStringID(createdComment.ID), nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(createdComment.ID)))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.DeleteCommunityComment(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for comment delete, got %d body=%s", rec.Code, rec.Body.String())
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

func TestHandlerCommunityValidationCases(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user-community-validation-handler@example.com", "user-community-validation-handler", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/community?category=bad", nil)
	env.handler.ListCommunityPosts(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid category, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community?page=abc", nil)
	env.handler.ListCommunityPosts(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid page, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community", []byte(`{"category":"x"}`))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.CreateCommunityPost(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 create bind error, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPatch, "/api/community/1", []byte(`{"title":null}`))
	ctx.Params = append(ctx.Params, ginParam("id", "1"))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.UpdateCommunityPost(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 update null field, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/bad", nil)
	ctx.Params = append(ctx.Params, ginParam("id", "bad"))
	env.handler.GetCommunityPost(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid id, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community/999999/likes", nil)
	ctx.Params = append(ctx.Params, ginParam("id", "999999"))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.ToggleCommunityPostLike(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 missing post like toggle, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/1/likes?page=bad", nil)
	ctx.Params = append(ctx.Params, ginParam("id", "1"))
	env.handler.CommunityPostLikes(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid likes pagination, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodGet, "/api/community/1/comments?page=bad", nil)
	ctx.Params = append(ctx.Params, ginParam("id", "1"))
	env.handler.CommunityComments(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid comments pagination, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPost, "/api/community/1/comments", []byte(`{"content":123}`))
	ctx.Params = append(ctx.Params, ginParam("id", "1"))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.CreateCommunityComment(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 create comment bind error, got %d", rec.Code)
	}

	ctx, rec = newJSONContext(t, http.MethodPatch, "/api/community/comments/1", []byte(`{"content":null}`))
	ctx.Params = append(ctx.Params, ginParam("id", "1"))
	ctx.Set("userID", user.ID)
	ctx.Set("role", models.UserRole)
	env.handler.UpdateCommunityComment(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 update comment null field, got %d", rec.Code)
	}
}
