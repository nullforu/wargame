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

	if resp.CreatedByUserID == nil || *resp.CreatedByUserID != admin.ID {
		t.Fatalf("expected creator id %d, got %+v", admin.ID, resp.CreatedByUserID)
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
	env.handler = New(env.cfg, env.authSvc, env.wargameSvc, env.userSvc, env.scoreSvc, env.stackSvc, env.redis)

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
	_ = createHandlerUser(t, env, "user2@example.com", "user2", "pass", models.UserRole)
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
		ctx.Request.Header.Set("Authorization", "Bearer "+accessToken)
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
