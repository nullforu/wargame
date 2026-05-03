package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/http/middleware"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/service"
	stackpkg "wargame/internal/stack"

	"github.com/gin-gonic/gin"
)

func TestHandlerCreateAndUpdateChallenge(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	t.Run("create challenge", func(t *testing.T) {
		body := []byte(`{"title":"Vote Challenge","description":"desc","category":"Misc","points":200,"flag":"FLAG{VOTE}","is_active":true}`)
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges", body)
		ctx.Set("userID", admin.ID)

		env.handler.CreateChallenge(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp challengeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode create response: %v", err)
		}

		if resp.Title != "Vote Challenge" || resp.Points != 200 {
			t.Fatalf("unexpected create response: %+v", resp)
		}
	})

	t.Run("update challenge", func(t *testing.T) {
		challenge := createHandlerChallenge(t, env, "Before Update", 100, "FLAG{OLD}", true)
		body := []byte(`{"title":"After Update","points":333,"is_active":false}`)
		ctx, rec := newJSONContext(t, http.MethodPut, "/api/admin/challenges/"+toStringID(challenge.ID), body)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))

		env.handler.UpdateChallenge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp challengeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode update response: %v", err)
		}

		if resp.Title != "After Update" || resp.Points != 333 || resp.IsActive {
			t.Fatalf("unexpected update response: %+v", resp)
		}
	})
}

func TestHandlerChallengeLevelVote(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "solver@example.com", "solver", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Vote Target", 100, "FLAG{TARGET}", true)

	t.Run("invalid body", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/vote", []byte(`{"level":"bad"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)

		env.handler.VoteChallengeLevel(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("unsolved challenge cannot vote", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/vote", []byte(`{"level":7}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)

		env.handler.VoteChallengeLevel(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	t.Run("vote and list logs", func(t *testing.T) {
		voteCtx, voteRec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/vote", []byte(`{"level":8}`))
		voteCtx.Params = append(voteCtx.Params, ginParam("id", toStringID(challenge.ID)))
		voteCtx.Set("userID", user.ID)

		env.handler.VoteChallengeLevel(voteCtx)
		if voteRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", voteRec.Code, voteRec.Body.String())
		}

		votesCtx, votesRec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/votes?page=1&page_size=10", nil)
		votesCtx.Params = append(votesCtx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.ChallengeVotes(votesCtx)
		if votesRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", votesRec.Code, votesRec.Body.String())
		}

		var resp challengeVotesResponse
		if err := json.Unmarshal(votesRec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode votes response: %v", err)
		}

		if len(resp.Votes) != 1 || resp.Votes[0].Level != 8 || resp.Votes[0].UserID != user.ID {
			t.Fatalf("unexpected votes response: %+v", resp)
		}
	})
}

func TestHandlerChallengeVotesValidation(t *testing.T) {
	env := setupHandlerTest(t)

	t.Run("invalid challenge id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/bad/votes", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))

		env.handler.ChallengeVotes(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		challenge := createHandlerChallenge(t, env, "Vote Validation", 100, "FLAG{VV}", true)
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/votes?page=abc", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))

		env.handler.ChallengeVotes(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found challenge", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/999999/votes", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "999999"))

		env.handler.ChallengeVotes(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerChallengeMyVote(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "myvote-handler@example.com", "myvote-handler", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "My Vote Challenge", 100, "FLAG{HMV}", true)

	t.Run("no vote yet", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/my-vote", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)

		env.handler.ChallengeMyVote(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp challengeMyVoteResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if resp.Level != nil {
			t.Fatalf("expected nil level, got %+v", resp.Level)
		}
	})

	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())
	if err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, challenge.ID, 9); err != nil {
		t.Fatalf("seed vote: %v", err)
	}

	t.Run("returns own vote", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/my-vote", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)

		env.handler.ChallengeMyVote(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp challengeMyVoteResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if resp.Level == nil || *resp.Level != 9 {
			t.Fatalf("expected level=9, got %+v", resp.Level)
		}
	})
}

func TestHandlerVoteChallengeLevelNotFound(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "user@example.com", "user", "pass", models.UserRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/999999/vote", []byte(`{"level":7}`))
	ctx.Params = append(ctx.Params, ginParam("id", "999999"))
	ctx.Set("userID", user.ID)

	env.handler.VoteChallengeLevel(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateChallengeSetsCreator(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "creator@example.com", "creator", "pass", models.AdminRole)

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges", []byte(`{"title":"Creator","description":"desc","category":"Misc","points":100,"flag":"FLAG{C}"}`))
	ctx.Set("userID", admin.ID)

	env.handler.CreateChallenge(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp challengeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.CreatedBy == nil || resp.CreatedBy.UserID == nil || *resp.CreatedBy.UserID != admin.ID {
		t.Fatalf("expected creator id %d, got %+v", admin.ID, resp.CreatedBy)
	}

	challenge, err := env.challengeRepo.GetByID(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("fetch created challenge: %v", err)
	}

	if challenge.CreatedByUserID == nil || *challenge.CreatedByUserID != admin.ID {
		t.Fatalf("creator not persisted: %+v", challenge.CreatedByUserID)
	}

	voteRepo := repo.NewChallengeVoteRepo(env.db)
	levels, err := voteRepo.RepresentativeLevelsByChallengeIDs(context.Background(), []int64{resp.ID})
	if err != nil {
		t.Fatalf("RepresentativeLevelsByChallengeIDs: %v", err)
	}

	if levels[resp.ID] != models.UnknownLevel {
		t.Fatalf("expected unknown level for no votes, got %d", levels[resp.ID])
	}
}

func TestHandlerAffiliationsAndRankings(t *testing.T) {
	env := setupHandlerTest(t)

	admin := createHandlerUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	user1 := createHandlerUser(t, env, "user1@example.com", "user1", "pass", models.UserRole)
	user2 := createHandlerUser(t, env, "user2@example.com", "user2", "pass", models.UserRole)

	createCtx, createRec := newJSONContext(t, http.MethodPost, "/api/admin/affiliations", []byte(`{"name":"Blue Team"}`))
	createCtx.Set("userID", admin.ID)
	env.handler.AdminCreateAffiliation(createCtx)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var created affiliationResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create affiliation: %v", err)
	}

	if created.ID <= 0 || created.Name != "Blue Team" {
		t.Fatalf("unexpected created affiliation: %+v", created)
	}

	t.Run("create affiliation validation", func(t *testing.T) {
		invalidCtx, invalidRec := newJSONContext(t, http.MethodPost, "/api/admin/affiliations", []byte(`{"name":123}`))
		invalidCtx.Set("userID", admin.ID)
		env.handler.AdminCreateAffiliation(invalidCtx)
		if invalidRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid body, got %d", invalidRec.Code)
		}

		dupCtx, dupRec := newJSONContext(t, http.MethodPost, "/api/admin/affiliations", []byte(`{"name":"blue team"}`))
		dupCtx.Set("userID", admin.ID)
		env.handler.AdminCreateAffiliation(dupCtx)
		if dupRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for duplicate affiliation, got %d", dupRec.Code)
		}
	})

	user1.AffiliationID = &created.ID
	if err := env.userRepo.Update(context.Background(), user1); err != nil {
		t.Fatalf("update user1 affiliation: %v", err)
	}
	user2.AffiliationID = &created.ID
	if err := env.userRepo.Update(context.Background(), user2); err != nil {
		t.Fatalf("update user2 affiliation: %v", err)
	}

	ch1 := createHandlerChallenge(t, env, "Ch1", 100, "FLAG{1}", true)
	ch2 := createHandlerChallenge(t, env, "Ch2", 200, "FLAG{2}", true)
	createHandlerSubmission(t, env, user1.ID, ch2.ID, true, time.Now().UTC())
	createHandlerSubmission(t, env, user2.ID, ch1.ID, true, time.Now().UTC())

	t.Run("list affiliations", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/affiliations?page=1&page_size=20", nil)
		env.handler.ListAffiliations(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp affiliationsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Affiliations) != 1 || resp.Affiliations[0].ID != created.ID {
			t.Fatalf("unexpected affiliations list: %+v", resp.Affiliations)
		}
	})

	t.Run("search affiliations", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/affiliations/search?q=blue&page=1&page_size=20", nil)
		env.handler.SearchAffiliations(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp affiliationsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Affiliations) != 1 || resp.Affiliations[0].ID != created.ID {
			t.Fatalf("unexpected filtered affiliations list: %+v", resp.Affiliations)
		}
	})

	t.Run("search affiliations validation", func(t *testing.T) {
		ctxMissingQ, recMissingQ := newJSONContext(t, http.MethodGet, "/api/affiliations/search?page=1&page_size=20", nil)
		env.handler.SearchAffiliations(ctxMissingQ)
		if recMissingQ.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing q, got %d", recMissingQ.Code)
		}

		ctxBadPage, recBadPage := newJSONContext(t, http.MethodGet, "/api/affiliations/search?q=blue&page=bad&page_size=20", nil)
		env.handler.SearchAffiliations(ctxBadPage)
		if recBadPage.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad page, got %d", recBadPage.Code)
		}
	})

	t.Run("list affiliation users", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/affiliations/"+toStringID(created.ID)+"/users?page=1&page_size=20", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(created.ID)))
		env.handler.ListAffiliationUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp usersListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Users) != 2 || resp.Users[0].ID != user2.ID || resp.Users[1].ID != user1.ID {
			t.Fatalf("unexpected affiliation users: %+v", resp.Users)
		}
	})

	t.Run("list affiliation users validation", func(t *testing.T) {
		ctxInvalidID, recInvalidID := newJSONContext(t, http.MethodGet, "/api/affiliations/bad/users?page=1&page_size=20", nil)
		ctxInvalidID.Params = append(ctxInvalidID.Params, ginParam("id", "bad"))
		env.handler.ListAffiliationUsers(ctxInvalidID)
		if recInvalidID.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid affiliation id, got %d", recInvalidID.Code)
		}

		ctxMissingAff, recMissingAff := newJSONContext(t, http.MethodGet, "/api/affiliations/999999/users?page=1&page_size=20", nil)
		ctxMissingAff.Params = append(ctxMissingAff.Params, ginParam("id", "999999"))
		env.handler.ListAffiliationUsers(ctxMissingAff)
		if recMissingAff.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for missing affiliation, got %d", recMissingAff.Code)
		}
	})

	t.Run("ranking users", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/rankings/users?page=1&page_size=20", nil)
		env.handler.RankingUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp userRankingListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Entries) < 2 || resp.Entries[0].UserID != user1.ID {
			t.Fatalf("unexpected ranking users: %+v", resp.Entries)
		}
	})

	t.Run("ranking users invalid pagination", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/rankings/users?page=bad&page_size=20", nil)
		env.handler.RankingUsers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("ranking affiliations", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/rankings/affiliations?page=1&page_size=20", nil)
		env.handler.RankingAffiliations(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp affiliationRankingListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Entries) != 1 || resp.Entries[0].AffiliationID != created.ID {
			t.Fatalf("unexpected ranking affiliations: %+v", resp.Entries)
		}
	})

	t.Run("ranking affiliations invalid pagination", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/rankings/affiliations?page=1&page_size=bad", nil)
		env.handler.RankingAffiliations(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("ranking affiliation users", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/rankings/affiliations/"+toStringID(created.ID)+"/users?page=1&page_size=20", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(created.ID)))
		env.handler.RankingAffiliationUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp userRankingListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Entries) != 2 || resp.Entries[0].UserID != user1.ID || resp.Entries[1].UserID != user2.ID {
			t.Fatalf("unexpected ranking affiliation users: %+v", resp.Entries)
		}
	})

	t.Run("ranking affiliation users validation", func(t *testing.T) {
		ctxInvalidID, recInvalidID := newJSONContext(t, http.MethodGet, "/api/rankings/affiliations/bad/users?page=1&page_size=20", nil)
		ctxInvalidID.Params = append(ctxInvalidID.Params, ginParam("id", "bad"))
		env.handler.RankingAffiliationUsers(ctxInvalidID)
		if recInvalidID.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recInvalidID.Code)
		}

		ctxMissingAff, recMissingAff := newJSONContext(t, http.MethodGet, "/api/rankings/affiliations/999999/users?page=1&page_size=20", nil)
		ctxMissingAff.Params = append(ctxMissingAff.Params, ginParam("id", "999999"))
		env.handler.RankingAffiliationUsers(ctxMissingAff)
		if recMissingAff.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", recMissingAff.Code)
		}
	})
}
func TestHandlerSubmitFlagFlow(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "flag-user@example.com", "flag-user", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Flag Target", 100, "FLAG{OK}", true)

	t.Run("invalid challenge id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/bad/submit", []byte(`{"flag":"FLAG{OK}"}`))
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		ctx.Set("userID", user.ID)

		env.handler.SubmitFlag(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/submit", []byte(`{"flag":1}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)

		env.handler.SubmitFlag(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("wrong then correct then already solved", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/submit", []byte(`{"flag":"FLAG{WRONG}"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.SubmitFlag(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("wrong submit expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var wrongResp map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &wrongResp); err != nil {
			t.Fatalf("decode wrong resp: %v", err)
		}

		if wrongResp["correct"] {
			t.Fatalf("expected incorrect submission")
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/submit", []byte(`{"flag":"FLAG{OK}"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.SubmitFlag(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("correct submit expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var okResp map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &okResp); err != nil {
			t.Fatalf("decode correct resp: %v", err)
		}

		if !okResp["correct"] {
			t.Fatalf("expected correct submission")
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/submit", []byte(`{"flag":"FLAG{OK}"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.SubmitFlag(ctx)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 for already solved, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerStackEndpoints(t *testing.T) {
	env := setupHandlerTest(t)
	provisioner := stackpkg.NewProvisionerMock()
	env.stackSvc = service.NewStackService(env.cfg.Stack, env.stackRepo, env.challengeRepo, env.submissionRepo, provisioner.Client(), env.redis)
	env.handler = New(env.cfg, env.authSvc, env.wargameSvc, env.userSvc, env.affiliationSvc, env.scoreSvc, env.stackSvc, env.redis)

	user := createHandlerUser(t, env, "stack-user@example.com", "stack-user", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Stack Target", 100, "FLAG{STACK}", true)
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge.StackEnabled = true
	challenge.StackTargetPorts = stackpkg.TargetPortSpecs{{ContainerPort: 80, Protocol: "TCP"}}
	challenge.StackPodSpec = &podSpec
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("enable stack on challenge: %v", err)
	}

	t.Run("stack disabled", func(t *testing.T) {
		hcopy := *env.handler
		hcopy.stacks = nil

		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/stack", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		hcopy.CreateStack(ctx)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
	})

	var created stackResponse
	t.Run("create stack", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/stack", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.CreateStack(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode create stack response: %v", err)
		}

		if created.StackID == "" || created.ChallengeID != challenge.ID {
			t.Fatalf("unexpected create stack response: %+v", created)
		}
	})

	t.Run("get stack", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/stack", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.GetStack(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list user stacks", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/stacks", nil)
		ctx.Set("userID", user.ID)
		env.handler.ListStacks(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin list/get/delete stack", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks", nil)
		env.handler.AdminListStacks(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/admin/stacks/"+created.StackID, nil)
		ctx.Params = append(ctx.Params, ginParam("stack_id", created.StackID))
		env.handler.AdminGetStack(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/admin/stacks/"+created.StackID, nil)
		ctx.Params = append(ctx.Params, ginParam("stack_id", created.StackID))
		env.handler.AdminDeleteStack(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete stack not found", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodDelete, "/api/challenges/"+toStringID(challenge.ID)+"/stack", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.DeleteStack(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerChallengeAdminAndFileEndpoints(t *testing.T) {
	env := setupHandlerTest(t)
	admin := createHandlerUser(t, env, "admin2@example.com", "admin2", "pass", models.AdminRole)
	user := createHandlerUser(t, env, "file-user@example.com", "file-user", "pass", models.UserRole)

	challenge := createHandlerChallenge(t, env, "File Target", 150, "FLAG{FILE}", true)
	challenge.CreatedByUserID = &admin.ID
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("set creator: %v", err)
	}

	t.Run("admin get challenge", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/challenges/"+toStringID(challenge.ID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.AdminGetChallenge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("request file upload and download then delete", func(t *testing.T) {
		uploadBody := []byte(`{"filename":"challenge.zip"}`)
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/challenges/"+toStringID(challenge.ID)+"/file/upload", uploadBody)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.RequestChallengeFileUpload(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("upload expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var uploadResp challengeFileUploadResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &uploadResp); err != nil {
			t.Fatalf("decode upload response: %v", err)
		}

		if uploadResp.Upload.URL == "" || uploadResp.Challenge.ID != challenge.ID {
			t.Fatalf("unexpected upload response: %+v", uploadResp)
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/file/download", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", user.ID)
		env.handler.RequestChallengeFileDownload(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("download expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/admin/challenges/"+toStringID(challenge.ID)+"/file", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.DeleteChallengeFile(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete file expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete challenge", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/challenges/"+toStringID(challenge.ID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.DeleteChallenge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete challenge expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerAdminBlockAndUnblockUser(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "blocked@example.com", "blocked", "pass", models.UserRole)

	t.Run("block user", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/"+toStringID(user.ID)+"/block", []byte(`{"reason":"abuse"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(user.ID)))
		env.handler.AdminBlockUser(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("block expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unblock user", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/"+toStringID(user.ID)+"/unblock", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(user.ID)))
		env.handler.AdminUnblockUser(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("unblock expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("block invalid body", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/"+toStringID(user.ID)+"/block", []byte(`{"reason":1}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(user.ID)))
		env.handler.AdminBlockUser(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unblock invalid id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/admin/users/bad/unblock", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.AdminUnblockUser(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerStackAdminValidation(t *testing.T) {
	env := setupHandlerTest(t)

	t.Run("admin stack id required", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodDelete, "/api/admin/stacks/", nil)
		ctx.Params = append(ctx.Params, ginParam("stack_id", ""))
		env.handler.AdminDeleteStack(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin get stack id required", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/admin/stacks/", nil)
		ctx.Params = append(ctx.Params, ginParam("stack_id", ""))
		env.handler.AdminGetStack(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerChallengeVotesUpdatedAtUTC(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "utc-user@example.com", "utc-user", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "UTC Vote", 100, "FLAG{UTC}", true)
	createHandlerSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	voteCtx, voteRec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/vote", []byte(`{"level":9}`))
	voteCtx.Params = append(voteCtx.Params, ginParam("id", toStringID(challenge.ID)))
	voteCtx.Set("userID", user.ID)
	env.handler.VoteChallengeLevel(voteCtx)
	if voteRec.Code != http.StatusOK {
		t.Fatalf("vote expected 200, got %d body=%s", voteRec.Code, voteRec.Body.String())
	}

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/votes?page=1&page_size=10", nil)
	ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
	env.handler.ChallengeVotes(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp challengeVotesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode votes response: %v", err)
	}

	if len(resp.Votes) != 1 || resp.Votes[0].UpdatedAt.Location().String() != "UTC" {
		t.Fatalf("expected utc updated_at, got %+v", resp.Votes)
	}
}
func TestParsePaginationParams(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/users", nil)
		page, pageSize, ok := parsePaginationParams(ctx)
		if !ok {
			t.Fatalf("expected ok")
		}

		if page != 0 || pageSize != 0 {
			t.Fatalf("expected zero values before normalization, got page=%d pageSize=%d", page, pageSize)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=abc", nil)
		page, pageSize, ok := parsePaginationParams(ctx)
		if ok || page != 0 || pageSize != 0 {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid page_size", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page_size=abc", nil)
		_, _, ok := parsePaginationParams(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestParseSearchQuery(t *testing.T) {
	t.Run("success with trim", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=%20web%20", nil)
		q, ok := parseSearchQuery(ctx)
		if !ok || q != "web" {
			t.Fatalf("unexpected q=%q ok=%v", q, ok)
		}
	})

	t.Run("required", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=", nil)
		_, ok := parseSearchQuery(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestParseChallengeFilters(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges?category=Web&level=7&solved=true&sort=most_solved", nil)
		filters, ok := parseChallengeFilters(ctx)
		if !ok || filters.Category != "Web" || filters.Level == nil || *filters.Level != 7 || filters.Solved == nil || !*filters.Solved || filters.Sort != "most_solved" {
			t.Fatalf("unexpected filters: ok=%v filters=%+v", ok, filters)
		}
	})

	t.Run("success unknown level", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges?level=0", nil)
		filters, ok := parseChallengeFilters(ctx)
		if !ok || filters.Level == nil || *filters.Level != 0 {
			t.Fatalf("unexpected filters for unknown level: ok=%v filters=%+v", ok, filters)
		}
	})

	t.Run("invalid level", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?level=abc", nil)
		_, ok := parseChallengeFilters(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid solved", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?solved=maybe", nil)
		_, ok := parseChallengeFilters(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid sort", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?sort=random", nil)
		_, ok := parseChallengeFilters(ctx)
		if ok {
			t.Fatalf("expected parse failure")
		}

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestPreviousChallengeForResponse(t *testing.T) {
	env := setupHandlerTest(t)
	prev := createHandlerChallenge(t, env, "Prev", 100, "FLAG{P}", true)
	other := createHandlerChallenge(t, env, "Other", 100, "FLAG{O}", true)

	t.Run("from current page map", func(t *testing.T) {
		byID := map[int64]*models.Challenge{prev.ID: prev}
		got := env.handler.previousChallengeForResponse(context.Background(), byID, &prev.ID)
		if got == nil || got.ID != prev.ID {
			t.Fatalf("expected previous from map, got %+v", got)
		}
	})

	t.Run("from repository fallback", func(t *testing.T) {
		byID := map[int64]*models.Challenge{}
		got := env.handler.previousChallengeForResponse(context.Background(), byID, &other.ID)
		if got == nil || got.ID != other.ID {
			t.Fatalf("expected previous from fallback, got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		missingID := int64(999999)
		got := env.handler.previousChallengeForResponse(context.Background(), map[int64]*models.Challenge{}, &missingID)
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})
}

func TestHandlerSearchChallengesAndUsers(t *testing.T) {
	env := setupHandlerTest(t)
	_ = createHandlerUser(t, env, "alpha@example.com", "alpha", "pass", models.UserRole)
	_ = createHandlerUser(t, env, "beta@example.com", "beta", "pass", models.UserRole)

	prev := createHandlerChallenge(t, env, "Web Prev", 100, "FLAG{1}", true)
	locked := createHandlerChallenge(t, env, "Web Locked", 200, "FLAG{2}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update challenge prerequisite: %v", err)
	}

	t.Run("search users success", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/search?q=alp&page=1&page_size=10", nil)
		env.handler.SearchUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp usersListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Users) != 1 || resp.Users[0].Username != "alpha" {
			t.Fatalf("unexpected users response: %+v", resp)
		}
	})

	t.Run("search users invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/search?q=alpha&page=abc", nil)
		env.handler.SearchUsers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("search challenges success with fallback previous challenge", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=Locked&page=1&page_size=1", nil)
		env.handler.SearchChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Challenges []lockedChallengeResponse `json:"challenges"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Challenges) != 1 {
			t.Fatalf("expected one challenge, got %d", len(resp.Challenges))
		}

		if resp.Challenges[0].PreviousChallengeTitle == nil || *resp.Challenges[0].PreviousChallengeTitle != prev.Title {
			t.Fatalf("expected previous challenge title fallback, got %+v", resp.Challenges[0])
		}
	})

	t.Run("search challenges missing query", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/search?q=", nil)
		env.handler.SearchChallenges(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandlerListChallengesAndUsers(t *testing.T) {
	env := setupHandlerTest(t)
	user1 := createHandlerUser(t, env, "user1@example.com", "user1", "pass", models.UserRole)
	user2 := createHandlerUser(t, env, "user2@example.com", "user2", "pass", models.UserRole)
	_ = createHandlerChallenge(t, env, "Challenge 1", 100, "FLAG{1}", true)
	_ = createHandlerChallenge(t, env, "Challenge 2", 200, "FLAG{2}", true)

	t.Run("list users", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=1&page_size=1", nil)
		env.handler.ListUsers(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp usersListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Users) != 1 || resp.Pagination.TotalCount != 2 {
			t.Fatalf("unexpected users list response: %+v", resp)
		}
		if resp.Users[0].ID != user2.ID || resp.Users[0].ID == user1.ID {
			t.Fatalf("expected newest user first, got %+v", resp.Users[0])
		}
	})

	t.Run("list users invalid page", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users?page=bad", nil)
		env.handler.ListUsers(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list challenges", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page=1&page_size=1", nil)
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengesListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if len(resp.Challenges) != 1 || resp.Pagination.TotalCount != 2 {
			t.Fatalf("unexpected challenges list response: %+v", resp)
		}
	})

	t.Run("list challenges with auth and prerequisite", func(t *testing.T) {
		prev := createHandlerChallenge(t, env, "Prev Auth", 100, "FLAG{P}", true)
		next := createHandlerChallenge(t, env, "Next Auth", 200, "FLAG{N}", true)
		next.PreviousChallengeID = &prev.ID
		if err := env.challengeRepo.Update(context.Background(), next); err != nil {
			t.Fatalf("update next challenge prerequisite: %v", err)
		}

		accessToken, _, _, err := env.authSvc.Login(context.Background(), user1.Email, "pass")
		if err != nil {
			t.Fatalf("login for auth list challenge: %v", err)
		}

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page=1&page_size=10", nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengesListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.Pagination.TotalCount < 4 {
			t.Fatalf("expected expanded challenge count, got %+v", resp.Pagination)
		}
	})

	t.Run("list challenges invalid page_size", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges?page_size=bad", nil)
		env.handler.ListChallenges(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

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

	t.Run("no access token cookie", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		if got := env.handler.optionalUserID(ctx); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("invalid access token cookie", func(t *testing.T) {
		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: "not-a-jwt"})
		if got := env.handler.optionalUserID(ctx); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("valid access token cookie", func(t *testing.T) {
		token, err := auth.GenerateAccessToken(env.cfg.JWT, 777, models.UserRole)
		if err != nil {
			t.Fatalf("GenerateAccessToken: %v", err)
		}

		ctx, _ := newJSONContext(t, http.MethodGet, "/api/challenges", nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
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
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
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

	t.Run("get challenge includes first blood", func(t *testing.T) {
		now := time.Now().UTC()
		sub := &models.Submission{UserID: user.ID, ChallengeID: prev.ID, Provided: "FLAG{PREV}", Correct: true, SubmittedAt: now.Add(-2 * time.Minute)}
		inserted, err := env.submissionRepo.CreateCorrectIfNotSolvedByUser(context.Background(), sub)
		if err != nil || !inserted {
			t.Fatalf("seed first blood: inserted=%v err=%v", inserted, err)
		}

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(prev.ID), nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(prev.ID)))
		env.handler.GetChallenge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp challengeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.FirstBlood == nil || resp.FirstBlood.UserID != user.ID || !resp.FirstBlood.IsFirstBlood {
			t.Fatalf("expected first blood in detail response, got %+v", resp.FirstBlood)
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
	challenge1 := createHandlerChallenge(t, env, "Solved Challenge 1", 100, "FLAG{SOLVED1}", true)
	challenge2 := createHandlerChallenge(t, env, "Solved Challenge 2", 200, "FLAG{SOLVED2}", true)
	now := time.Now().UTC()
	createHandlerSubmission(t, env, user.ID, challenge1.ID, true, now.Add(-2*time.Minute))
	createHandlerSubmission(t, env, user.ID, challenge2.ID, true, now.Add(-time.Minute))

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

		if len(resp.Solved) != 2 || resp.Solved[0].ChallengeID != challenge2.ID || resp.Solved[1].ChallengeID != challenge1.ID {
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

		setCookies := rec.Header().Values("Set-Cookie")
		if len(setCookies) == 0 {
			t.Fatalf("expected auth cookies on login")
		}

		refreshToken := ""
		csrfToken := ""
		for _, c := range setCookies {
			if after, ok := strings.CutPrefix(c, "refresh_token="); ok {
				refreshToken = strings.SplitN(after, ";", 2)[0]
			}

			if after, ok := strings.CutPrefix(c, "csrf_token="); ok {
				csrfToken = strings.SplitN(after, ";", 2)[0]
			}
		}

		if refreshToken == "" || csrfToken == "" {
			t.Fatalf("expected refresh/csrf cookies on login")
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/refresh", nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
		ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
		ctx.Request.Header.Set("X-CSRF-Token", csrfToken)
		env.handler.Refresh(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("refresh status %d: %s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/logout", nil)
		ctx.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
		ctx.Request.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
		ctx.Request.Header.Set("X-CSRF-Token", csrfToken)
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

		uploadBody := []byte(`{"filename":"avatar.png"}`)
		ctx, rec = newJSONContext(t, http.MethodPost, "/api/me/profile-image/upload", uploadBody)
		ctx.Set("userID", user.ID)
		env.handler.RequestProfileImageUpload(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("profile image upload status %d: %s", rec.Code, rec.Body.String())
		}

		var uploadResp profileImageUploadResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &uploadResp); err != nil {
			t.Fatalf("decode profile image upload: %v", err)
		}

		if uploadResp.Upload.Method != "POST" || uploadResp.Upload.URL == "" {
			t.Fatalf("unexpected upload response: %+v", uploadResp.Upload)
		}
		if uploadResp.User.ProfileImage == nil || *uploadResp.User.ProfileImage == "" {
			t.Fatalf("expected profile image key in response, got %+v", uploadResp.User)
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/me/profile-image/upload", []byte(`{"filename":"avatar.gif"}`))
		ctx.Set("userID", user.ID)
		env.handler.RequestProfileImageUpload(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 invalid extension, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/me/profile-image", nil)
		ctx.Set("userID", user.ID)
		env.handler.DeleteProfileImage(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("profile image delete status %d: %s", rec.Code, rec.Body.String())
		}

		var deletedResp userMeResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &deletedResp); err != nil {
			t.Fatalf("decode profile image delete: %v", err)
		}
		if deletedResp.ProfileImage != nil {
			t.Fatalf("expected nil profile image after delete, got %+v", deletedResp.ProfileImage)
		}
	})
}

func TestHandlerGetUser(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "get-user@example.com", "get-user", "pass", models.UserRole)

	t.Run("invalid id", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.GetUser(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/999999", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "999999"))
		env.handler.GetUser(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/users/"+toStringID(user.ID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(user.ID)))
		env.handler.GetUser(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp userDetailResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.ID != user.ID || resp.Username != user.Username {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})
}

func TestHandlerWriteupHandlers(t *testing.T) {
	env := setupHandlerTest(t)
	writer := createHandlerUser(t, env, "hwriter@example.com", "hwriter", "pass", models.UserRole)
	viewer := createHandlerUser(t, env, "hviewer@example.com", "hviewer", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Handler Writeup", 250, "flag{hwriteup}", true)
	viewerAccess, err := auth.GenerateAccessToken(env.cfg.JWT, viewer.ID, viewer.Role)
	if err != nil {
		t.Fatalf("GenerateAccessToken viewer: %v", err)
	}

	t.Run("challenge writeups validation", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/bad/writeups", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.ChallengeWriteups(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad challenge id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/writeups?page=bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.ChallengeWriteups(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad pagination, got %d", rec.Code)
		}
	})

	t.Run("create writeup flow", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/writeups", []byte(`{"content":123}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", writer.ID)
		env.handler.CreateWriteup(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid create payload, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/writeups", []byte(`{"content":"first body"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", writer.ID)
		env.handler.CreateWriteup(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 before solve, got %d body=%s", rec.Code, rec.Body.String())
		}

		createHandlerSubmission(t, env, writer.ID, challenge.ID, true, time.Now().UTC())

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/writeups", []byte(`{"content":"first body"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", writer.ID)
		env.handler.CreateWriteup(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 create, got %d body=%s", rec.Code, rec.Body.String())
		}

		var created writeupResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode created writeup: %v", err)
		}

		if created.ID <= 0 || created.Content == nil || *created.Content != "first body" {
			t.Fatalf("unexpected create response: %+v", created)
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/writeups", []byte(`{"content":"dup"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", writer.ID)
		env.handler.CreateWriteup(ctx)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 duplicate create, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("challenge writeups visibility", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/writeups?page=1&page_size=10", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: viewerAccess})
		env.handler.ChallengeWriteups(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 list unsolved viewer, got %d body=%s", rec.Code, rec.Body.String())
		}

		var unsolved writeupsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &unsolved); err != nil {
			t.Fatalf("decode unsolved writeups list: %v", err)
		}

		if unsolved.CanViewContent || len(unsolved.Writeups) != 1 || unsolved.Writeups[0].Content != nil {
			t.Fatalf("unexpected unsolved writeups list: %+v", unsolved)
		}
	})

	var writeupID int64
	t.Run("get/update/delete and user/my writeups", func(t *testing.T) {
		rows, _, _, err := env.wargameSvc.ChallengeWriteupsPage(context.Background(), challenge.ID, writer.ID, 1, 10)
		if err != nil || len(rows) != 1 {
			t.Fatalf("seed fetch created writeup failed rows=%+v err=%v", rows, err)
		}
		writeupID = rows[0].ID

		ctx, rec := newJSONContext(t, http.MethodGet, "/api/writeups/bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.GetWriteup(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 bad writeup id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/writeups/"+toStringID(writeupID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: viewerAccess})
		env.handler.GetWriteup(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 get unsolved viewer, got %d body=%s", rec.Code, rec.Body.String())
		}

		var unsolvedDetail writeupDetailResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &unsolvedDetail); err != nil {
			t.Fatalf("decode unsolved detail: %v", err)
		}

		if unsolvedDetail.CanViewContent || unsolvedDetail.Writeup.Content != nil {
			t.Fatalf("unexpected unsolved detail response: %+v", unsolvedDetail)
		}

		createHandlerSubmission(t, env, viewer.ID, challenge.ID, true, time.Now().UTC())

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/writeups/"+toStringID(writeupID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: viewerAccess})
		env.handler.GetWriteup(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 get solved viewer, got %d body=%s", rec.Code, rec.Body.String())
		}

		var solvedDetail writeupDetailResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &solvedDetail); err != nil {
			t.Fatalf("decode solved detail: %v", err)
		}

		if !solvedDetail.CanViewContent || solvedDetail.Writeup.Content == nil || *solvedDetail.Writeup.Content == "" {
			t.Fatalf("unexpected solved detail response: %+v", solvedDetail)
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/writeups/"+toStringID(writeupID), []byte(`{"content":123}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", writer.ID)
		env.handler.UpdateWriteup(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 invalid patch body, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/writeups/"+toStringID(writeupID), []byte(`{}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", writer.ID)
		env.handler.UpdateWriteup(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 empty patch body, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/writeups/"+toStringID(writeupID), []byte(`{"content":"hacked"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", viewer.ID)
		env.handler.UpdateWriteup(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden patch, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/writeups/"+toStringID(writeupID), []byte(`{"content":"updated by owner"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", writer.ID)
		env.handler.UpdateWriteup(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 owner patch, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/me/writeups?page=bad", nil)
		ctx.Set("userID", writer.ID)
		env.handler.MyWriteups(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 my writeups bad page, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/me/writeups?page=1&page_size=10", nil)
		ctx.Set("userID", writer.ID)
		env.handler.MyWriteups(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 my writeups, got %d body=%s", rec.Code, rec.Body.String())
		}

		var myResp writeupsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &myResp); err != nil {
			t.Fatalf("decode my writeups: %v", err)
		}

		if !myResp.CanViewContent || len(myResp.Writeups) != 1 || myResp.Writeups[0].Content == nil {
			t.Fatalf("unexpected my writeups response: %+v", myResp)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/bad/writeups", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.GetUserWriteups(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 user writeups bad id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/999999/writeups", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "999999"))
		env.handler.GetUserWriteups(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 user writeups not found user, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/users/"+toStringID(writer.ID)+"/writeups?page=1&page_size=10", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writer.ID)))
		ctx.Request.AddCookie(&http.Cookie{Name: "access_token", Value: viewerAccess})
		env.handler.GetUserWriteups(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 user writeups, got %d body=%s", rec.Code, rec.Body.String())
		}

		var userWriteupsResp writeupsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &userWriteupsResp); err != nil {
			t.Fatalf("decode user writeups: %v", err)
		}

		if !userWriteupsResp.CanViewContent || len(userWriteupsResp.Writeups) != 1 || userWriteupsResp.Writeups[0].Content == nil {
			t.Fatalf("unexpected user writeups response: %+v", userWriteupsResp)
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/writeups/bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		ctx.Set("userID", writer.ID)
		env.handler.DeleteWriteup(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 bad delete id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/writeups/"+toStringID(writeupID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", viewer.ID)
		env.handler.DeleteWriteup(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden delete, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/writeups/"+toStringID(writeupID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", writer.ID)
		env.handler.DeleteWriteup(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 owner delete, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/writeups/"+toStringID(writeupID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(writeupID)))
		ctx.Set("userID", writer.ID)
		env.handler.GetWriteup(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerChallengeCommentHandlers(t *testing.T) {
	env := setupHandlerTest(t)
	owner := createHandlerUser(t, env, "comment-owner@example.com", "comment-owner", "pass", models.UserRole)
	other := createHandlerUser(t, env, "comment-other@example.com", "comment-other", "pass", models.UserRole)
	challenge := createHandlerChallenge(t, env, "Handler Comment", 150, "FLAG{HCMT}", true)

	t.Run("challenge comments validation", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/bad/challenge-comments", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		env.handler.ChallengeComments(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad challenge id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments?page=bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.ChallengeComments(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad pagination, got %d", rec.Code)
		}
	})

	t.Run("create comment flow", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments", []byte(`{"content":123}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", owner.ID)
		env.handler.CreateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments", []byte(`{"content":"   "}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", owner.ID)
		env.handler.CreateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for blank content, got %d body=%s", rec.Code, rec.Body.String())
		}

		tooLong := strings.Repeat("가", 501)
		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments", []byte(`{"content":"`+tooLong+`"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", owner.ID)
		env.handler.CreateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for too long content, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments", []byte(`{"content":"첫 댓글"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", owner.ID)
		env.handler.CreateChallengeCommentItem(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 create first, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPost, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments", []byte(`{"content":"둘째 댓글"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		ctx.Set("userID", other.ID)
		env.handler.CreateChallengeCommentItem(ctx)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 create second, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	var latestCommentID int64
	t.Run("list comments latest first", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodGet, "/api/challenges/"+toStringID(challenge.ID)+"/challenge-comments?page=1&page_size=10", nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(challenge.ID)))
		env.handler.ChallengeComments(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 list, got %d body=%s", rec.Code, rec.Body.String())
		}

		var resp challengeCommentsListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode comments list: %v", err)
		}

		if len(resp.Comments) != 2 {
			t.Fatalf("expected 2 comments, got %d", len(resp.Comments))
		}

		if resp.Comments[0].CreatedAt.Before(resp.Comments[1].CreatedAt) {
			t.Fatalf("expected latest-first order, got first=%s second=%s", resp.Comments[0].CreatedAt, resp.Comments[1].CreatedAt)
		}

		if resp.Comments[0].Content != "둘째 댓글" || resp.Comments[1].Content != "첫 댓글" {
			t.Fatalf("unexpected order/content: %+v", resp.Comments)
		}

		latestCommentID = resp.Comments[0].ID
	})

	t.Run("update comment validation and permission", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":123}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid patch payload, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for empty patch payload, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":"   "}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for blank content, got %d body=%s", rec.Code, rec.Body.String())
		}

		tooLong := strings.Repeat("가", 501)
		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":"`+tooLong+`"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for too long patch content, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":"권한 없는 수정 시도"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", owner.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden update, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":"수정 완료"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 owner update, got %d body=%s", rec.Code, rec.Body.String())
		}

		var updated challengeCommentResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
			t.Fatalf("decode updated comment: %v", err)
		}

		if updated.Content != "수정 완료" {
			t.Fatalf("unexpected updated content: %+v", updated)
		}
	})

	t.Run("delete comment validation and permission", func(t *testing.T) {
		ctx, rec := newJSONContext(t, http.MethodDelete, "/api/challenges/challenge-comments/bad", nil)
		ctx.Params = append(ctx.Params, ginParam("id", "bad"))
		ctx.Set("userID", other.ID)
		env.handler.DeleteChallengeCommentItem(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad delete id, got %d", rec.Code)
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", owner.ID)
		env.handler.DeleteChallengeCommentItem(ctx)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden delete, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodDelete, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), nil)
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.DeleteChallengeCommentItem(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 owner delete, got %d body=%s", rec.Code, rec.Body.String())
		}

		ctx, rec = newJSONContext(t, http.MethodPatch, "/api/challenges/challenge-comments/"+toStringID(latestCommentID), []byte(`{"content":"삭제 후 수정"}`))
		ctx.Params = append(ctx.Params, ginParam("id", toStringID(latestCommentID)))
		ctx.Set("userID", other.ID)
		env.handler.UpdateChallengeCommentItem(ctx)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}
