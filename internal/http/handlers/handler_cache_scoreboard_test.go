package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"wargame/internal/http/middleware"
	"wargame/internal/models"
)

func TestHandlerCacheHelpers(t *testing.T) {
	env := setupHandlerTest(t)

	t.Run("respond from cache", func(t *testing.T) {
		key := "cache:test:respond"
		want := `{"ok":true}`
		if err := env.redis.Set(context.Background(), key, want, time.Minute).Err(); err != nil {
			t.Fatalf("seed cache: %v", err)
		}

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
		if !env.handler.respondFromCache(ctx, key) {
			t.Fatalf("expected cache hit")
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != want {
			t.Fatalf("unexpected cached body: %s", rec.Body.String())
		}
	})

	t.Run("store and invalidate cache", func(t *testing.T) {
		if err := env.redis.Set(context.Background(), "leaderboard:users:stale", "stale", time.Minute).Err(); err != nil {
			t.Fatalf("seed stale cache: %v", err)
		}

		ctx, _ := newJSONContext(t, http.MethodGet, "/api/leaderboard", nil)
		env.handler.storeCache(ctx, "leaderboard:users:fresh", map[string]bool{"ok": true}, time.Minute)
		if got, err := env.redis.Get(context.Background(), "leaderboard:users:fresh").Result(); err != nil || got == "" {
			t.Fatalf("expected stored cache, got %q err %v", got, err)
		}

		env.handler.invalidateCacheByPrefix(context.Background(), "leaderboard:users:")
		if exists, err := env.redis.Exists(context.Background(), "leaderboard:users:stale", "leaderboard:users:fresh").Result(); err != nil {
			t.Fatalf("exists check: %v", err)
		} else if exists != 0 {
			t.Fatalf("expected caches invalidated, exists=%d", exists)
		}
	})

	t.Run("notify scoreboard changed", func(t *testing.T) {
		sub := env.redis.Subscribe(context.Background(), "scoreboard.events")
		defer sub.Close()

		env.handler.notifyScoreboardChanged(context.Background(), "test")
		msg, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			t.Fatalf("receive message: %v", err)
		}

		var payload struct {
			Scope  string    `json:"scope"`
			Reason string    `json:"reason"`
			TS     time.Time `json:"ts"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Scope != "all" || payload.Reason != "test" || payload.TS.IsZero() {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	})
}

func TestHandlerLeaderboardAndTimeline(t *testing.T) {
	env := setupHandlerTest(t)

	user1 := createHandlerUser(t, env, "lb1@example.com", "lb1", "pass", models.UserRole)
	user2 := createHandlerUser(t, env, "lb2@example.com", "lb2", "pass", models.UserRole)
	ch1 := createHandlerChallenge(t, env, "LB Ch1", 100, "FLAG{LB1}", true)
	ch2 := createHandlerChallenge(t, env, "LB Ch2", 200, "FLAG{LB2}", true)

	now := time.Now().UTC()
	createHandlerSubmission(t, env, user1.ID, ch1.ID, true, now.Add(-2*time.Minute))
	createHandlerSubmission(t, env, user2.ID, ch2.ID, true, now.Add(-time.Minute))

	t.Run("leaderboard pagination response", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/leaderboard?page=1&page_size=1", nil)
		env.handler.Leaderboard(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp leaderboardListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Entries) != 1 || resp.Pagination.TotalCount != 2 || !resp.Pagination.HasNext {
			t.Fatalf("unexpected leaderboard response: %+v", resp)
		}
	})

	t.Run("timeline response", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/timeline", nil)
		env.handler.Timeline(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp timelineResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Submissions) < 1 {
			t.Fatalf("expected submissions, got %+v", resp)
		}
	})
}

func TestHandlerAuthMeUpdateFlow(t *testing.T) {
	env := setupHandlerTest(t)

	t.Run("register/login/refresh/logout", func(t *testing.T) {
		registerBody := []byte(`{"email":"flow@example.com","username":"flow-user","password":"pass1234"}`)
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/register", registerBody)
		env.handler.Register(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
		}

		loginBody := []byte(`{"email":"flow@example.com","password":"pass1234"}`)
		ctx, rec = newJSONContext(t, http.MethodPost, "/api/login", loginBody)
		env.handler.Login(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
		}

		var loginResp loginResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
			t.Fatalf("expected tokens in login response: %+v", loginResp)
		}

		refreshBody := []byte(`{"refresh_token":"` + loginResp.RefreshToken + `"}`)
		ctx, rec = newJSONContext(t, http.MethodPost, "/api/refresh", refreshBody)
		env.handler.Refresh(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("refresh status %d: %s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/logout", refreshBody)
		env.handler.Logout(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("logout status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("me and update me", func(t *testing.T) {
		user := createHandlerUser(t, env, "me@example.com", "me-user", "pass", models.UserRole)

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/me", nil)
		ctx.Set("userID", user.ID)
		env.handler.Me(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("me status %d: %s", rec.Code, rec.Body.String())
		}

		updateBody := []byte(`{"username":"me-user-updated"}`)
		ctx, rec = newJSONContext(t, http.MethodPut, "/api/me", updateBody)
		ctx.Set("userID", user.ID)
		env.handler.UpdateMe(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("update me status %d: %s", rec.Code, rec.Body.String())
		}

		updated, err := env.userRepo.GetByID(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("get updated user: %v", err)
		}
		if updated.Username != "me-user-updated" {
			t.Fatalf("expected updated username, got %q", updated.Username)
		}

		if middleware.UserID(ctx) != user.ID {
			t.Fatalf("expected middleware user id %d", user.ID)
		}
	})
}
