package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/models"
	"wargame/internal/service"

	"github.com/gin-gonic/gin"
)

func TestParseIDParam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges/12", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "12"))
		id, ok := parseIDParam(ctx, "id")
		if !ok || id != 12 {
			t.Fatalf("unexpected parse result id=%d ok=%v", id, ok)
		}
	})

	t.Run("missing", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		if _, ok := parseIDParam(ctx, "id"); ok {
			t.Fatalf("expected failure")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges/abc", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "abc"))
		if _, ok := parseIDParam(ctx, "id"); ok {
			t.Fatalf("expected failure")
		}
	})
}

func TestParseIDParamOrError(t *testing.T) {
	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/abc", nil)
	ctx.Params = append(ctx.Params, ginParam("id", "abc"))
	if _, ok := parseIDParamOrError(ctx, "id"); ok {
		t.Fatalf("expected failure")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestOptionalUserID(t *testing.T) {
	env := setupHandlerTest(t)

	t.Run("no authorization header", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		if got := env.handler.optionalUserID(ctx); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("invalid bearer format", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		ctx.Request.Header.Set("Authorization", "Bad token")
		if got := env.handler.optionalUserID(ctx); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("valid access token", func(t *testing.T) {
		token, err := auth.GenerateAccessToken(env.cfg.JWT, 777, models.UserRole)
		if err != nil {
			t.Fatalf("GenerateAccessToken: %v", err)
		}

		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		ctx.Request.Header.Set("Authorization", "Bearer "+token)
		if got := env.handler.optionalUserID(ctx); got != 777 {
			t.Fatalf("expected 777, got %d", got)
		}
	})
}

func TestHandlerGetChallengeAndSolvers(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "detail@example.com", "detail-user", "pass", models.UserRole)

	prev := createHandlerChallenge(t, env, "Detail Prev", 100, "FLAG{PREV}", true)
	locked := createHandlerChallenge(t, env, "Detail Locked", 200, "FLAG{LOCKED}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	token, _, _, err := env.authSvc.Login(context.Background(), user.Email, "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Run("get challenge invalid id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/abc", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "abc"))
		env.handler.GetChallenge(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("get challenge locked response", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(locked.ID), nil)
		ctx.Request.Header.Set("Authorization", "Bearer "+token)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(locked.ID)))
		env.handler.GetChallenge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp lockedChallengeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !resp.IsLocked {
			t.Fatalf("expected locked response")
		}
		if resp.PreviousChallengeID == nil || *resp.PreviousChallengeID != prev.ID {
			t.Fatalf("unexpected previous challenge in response: %+v", resp)
		}
	})

	t.Run("challenge solvers invalid id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/abc/solvers", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "abc"))
		env.handler.ChallengeSolvers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("challenge solvers success", func(t *testing.T) {
		now := time.Now().UTC()
		createHandlerSubmission(t, env, user.ID, prev.ID, true, now.Add(-time.Minute))

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(prev.ID)+"/solvers?page=1&page_size=10", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(prev.ID)))
		env.handler.ChallengeSolvers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengeSolversResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Solvers) != 1 || resp.Solvers[0].UserID != user.ID {
			t.Fatalf("unexpected solvers response: %+v", resp)
		}
	})
}

func TestHandlerGetUserSolved(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "solved@example.com", "solved-user", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Solved Challenge", 100, "FLAG{SOLVED}", true)
	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	t.Run("invalid user id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/abc/solved", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "abc"))
		env.handler.GetUserSolved(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/"+toStringID(user.ID)+"/solved?page=1&page_size=10", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(user.ID)))
		env.handler.GetUserSolved(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp userSolvedListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Solved) != 1 || resp.Solved[0].ChallengeID != challenge.ID {
			t.Fatalf("unexpected solved response: %+v", resp)
		}
	})
}

func TestRequireNonNullOptionalStringAndPointer(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		val, err := requireNonNullOptionalString("title", optionalString{})
		if err != nil || val != nil {
			t.Fatalf("unexpected result val=%v err=%v", val, err)
		}
	})

	t.Run("set null invalid", func(t *testing.T) {
		_, err := requireNonNullOptionalString("title", optionalString{Set: true, Value: nil})
		if !errorsIsInvalidInput(err) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("optionalStringToPointer", func(t *testing.T) {
		if optionalStringToPointer(optionalString{}) != nil {
			t.Fatalf("expected nil for unset")
		}
		empty := optionalStringToPointer(optionalString{Set: true, Value: nil})
		if empty == nil || *empty != "" {
			t.Fatalf("expected empty pointer, got %+v", empty)
		}
		value := "ok"
		got := optionalStringToPointer(optionalString{Set: true, Value: &value})
		if got == nil || *got != value {
			t.Fatalf("expected value pointer, got %+v", got)
		}
	})
}

func ginParam(key, value string) gin.Param {
	return gin.Param{Key: key, Value: value}
}

func toStringID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func errorsIsInvalidInput(err error) bool {
	var ve *service.ValidationError
	return err != nil && errors.As(err, &ve)
}
