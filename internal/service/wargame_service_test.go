package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/stack"
	"wargame/internal/storage"
	"wargame/internal/utils"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type errorFileStore struct {
	uploadErr   error
	downloadErr error
	deleteErr   error
}

func (e errorFileStore) PresignUpload(ctx context.Context, key, contentType string) (storage.PresignedPost, error) {
	if e.uploadErr != nil {
		return storage.PresignedPost{}, e.uploadErr
	}
	return storage.PresignedPost{URL: "https://example.com/upload", Fields: map[string]string{"key": key}}, nil
}

func (e errorFileStore) PresignDownload(ctx context.Context, key, filename string) (storage.PresignedURL, error) {
	if e.downloadErr != nil {
		return storage.PresignedURL{}, e.downloadErr
	}
	return storage.PresignedURL{URL: "https://example.com/download/" + key}, nil
}

func (e errorFileStore) Delete(ctx context.Context, key string) error { return e.deleteErr }

func newClosedServiceDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := db.New(serviceCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	_ = conn.Close()
	return conn
}

func ptrString(value string) *string { return &value }

func TestWargameServiceCreateListGetChallenge(t *testing.T) {
	env := setupServiceTest(t)

	created, err := env.wargameSvc.CreateChallenge(context.Background(), "Title", "Desc", "Misc", 100, "FLAG{1}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	list, pagination, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{})
	if err != nil {
		t.Fatalf("ListChallenges: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected challenge list: %+v", list)
	}
	if pagination.TotalCount != 1 {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	found, err := env.wargameSvc.GetChallengeByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetChallengeByID: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("unexpected challenge: %+v", found)
	}
}

func TestWargameServiceSearchAndPagination(t *testing.T) {
	env := setupServiceTest(t)
	_, err := env.wargameSvc.CreateChallenge(context.Background(), "Web Warmup", "Desc", "Web", 100, "FLAG{1}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge 1: %v", err)
	}
	_, err = env.wargameSvc.CreateChallenge(context.Background(), "Web Advanced", "Desc", "Web", 100, "FLAG{2}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge 2: %v", err)
	}
	_, err = env.wargameSvc.CreateChallenge(context.Background(), "Crypto Basic", "Desc", "Crypto", 100, "FLAG{3}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge 3: %v", err)
	}

	rows, pagination, err := env.wargameSvc.SearchChallenges(context.Background(), "Web", 1, 1, ChallengeQueryFilter{})
	if err != nil {
		t.Fatalf("SearchChallenges: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if pagination.TotalCount != 2 || pagination.TotalPages != 2 || !pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	if _, _, err := env.wargameSvc.SearchChallenges(context.Background(), " ", 1, 10, ChallengeQueryFilter{}); err == nil {
		t.Fatalf("expected required query validation error")
	}
}

func TestWargameServiceUpdateAndDeleteChallenge(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Old", 50, "FLAG{2}", true)

	title := "New"
	desc := "New Desc"
	category := "Crypto"
	points := 120
	active := false

	updated, err := env.wargameSvc.UpdateChallenge(context.Background(), challenge.ID, &title, &desc, &category, &points, nil, &active, nil, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("UpdateChallenge: %v", err)
	}
	if updated.Title != title || updated.Category != category || updated.Points != points || updated.IsActive != active {
		t.Fatalf("unexpected updated challenge: %+v", updated)
	}

	if err := env.wargameSvc.DeleteChallenge(context.Background(), challenge.ID); err != nil {
		t.Fatalf("DeleteChallenge: %v", err)
	}
	if err := env.wargameSvc.DeleteChallenge(context.Background(), challenge.ID); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestWargameServiceSubmitFlagAndSolvedQueries(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Solve", 100, "FLAG{SOLVE}", true)

	correct, err := env.wargameSvc.SubmitFlag(context.Background(), user.ID, challenge.ID, "WRONG")
	if err != nil {
		t.Fatalf("SubmitFlag wrong: %v", err)
	}
	if correct {
		t.Fatalf("expected wrong submission to be false")
	}

	correct, err = env.wargameSvc.SubmitFlag(context.Background(), user.ID, challenge.ID, "FLAG{SOLVE}")
	if err != nil {
		t.Fatalf("SubmitFlag correct: %v", err)
	}
	if !correct {
		t.Fatalf("expected correct submission")
	}

	ids, err := env.wargameSvc.SolvedChallengeIDs(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("SolvedChallengeIDs: %v", err)
	}
	if _, ok := ids[challenge.ID]; !ok {
		t.Fatalf("expected challenge id in solved set")
	}

	solvedPageRows, solvedPagination, err := env.wargameSvc.SolvedChallengesPage(context.Background(), user.ID, 1, 10)
	if err != nil {
		t.Fatalf("SolvedChallengesPage: %v", err)
	}
	if len(solvedPageRows) != 1 || solvedPageRows[0].ChallengeID != challenge.ID {
		t.Fatalf("unexpected solved page rows: %+v", solvedPageRows)
	}
	if solvedPagination.TotalCount != 1 {
		t.Fatalf("unexpected solved pagination: %+v", solvedPagination)
	}
}

func TestWargameServiceChallengeFiltersAndPagedSolvedAndSolvers(t *testing.T) {
	env := setupServiceTest(t)
	user1 := createUser(t, env, "f1@example.com", "f1", "pass", models.UserRole)
	user2 := createUser(t, env, "f2@example.com", "f2", "pass", models.UserRole)

	chWeb, err := env.wargameSvc.CreateChallenge(context.Background(), "Web SQL", "Desc", "Web", 200, "FLAG{WEB}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge web: %v", err)
	}
	chCrypto, err := env.wargameSvc.CreateChallenge(context.Background(), "Crypto RSA", "Desc", "Crypto", 150, "FLAG{CRYPTO}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge crypto: %v", err)
	}

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, chWeb.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, chWeb.ID, true, now.Add(-1*time.Minute))

	solvedTrue := true
	rows, pagination, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{
		Category: "Web",
		Solved:   &solvedTrue,
		UserID:   user1.ID,
	})
	if err != nil {
		t.Fatalf("ListChallenges solved=true filter: %v", err)
	}

	if len(rows) != 1 || rows[0].ID != chWeb.ID {
		t.Fatalf("unexpected solved=true filtered rows: %+v", rows)
	}

	if pagination.TotalCount != 1 {
		t.Fatalf("unexpected solved=true pagination: %+v", pagination)
	}

	solvedFalse := false
	rows, pagination, err = env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{
		Solved: &solvedFalse,
		UserID: user1.ID,
	})
	if err != nil {
		t.Fatalf("ListChallenges solved=false filter: %v", err)
	}

	if len(rows) != 1 || rows[0].ID != chCrypto.ID {
		t.Fatalf("unexpected solved=false filtered rows: %+v", rows)
	}

	if pagination.TotalCount != 1 {
		t.Fatalf("unexpected solved=false pagination: %+v", pagination)
	}

	if _, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{Solved: &solvedTrue}); err == nil {
		t.Fatalf("expected solved filter auth_required validation error")
	}

	searchRows, _, err := env.wargameSvc.SearchChallenges(context.Background(), "Web", 1, 20, ChallengeQueryFilter{
		Category: "Web",
		Solved:   &solvedTrue,
		UserID:   user1.ID,
	})
	if err != nil {
		t.Fatalf("SearchChallenges with filters: %v", err)
	}

	if len(searchRows) != 1 || searchRows[0].ID != chWeb.ID {
		t.Fatalf("unexpected search rows: %+v", searchRows)
	}

	solvedPageRows, solvedPagination, err := env.wargameSvc.SolvedChallengesPage(context.Background(), user1.ID, 1, 1)
	if err != nil {
		t.Fatalf("SolvedChallengesPage: %v", err)
	}

	if len(solvedPageRows) != 1 || solvedPageRows[0].ChallengeID != chWeb.ID {
		t.Fatalf("unexpected solved page rows: %+v", solvedPageRows)
	}

	if solvedPagination.TotalCount != 1 || solvedPagination.TotalPages != 1 {
		t.Fatalf("unexpected solved pagination: %+v", solvedPagination)
	}

	solversPage1, solversPagination1, err := env.wargameSvc.ChallengeSolversPage(context.Background(), chWeb.ID, 1, 1)
	if err != nil {
		t.Fatalf("ChallengeSolversPage page1: %v", err)
	}

	if len(solversPage1) != 1 || solversPagination1.TotalCount != 2 || !solversPagination1.HasNext {
		t.Fatalf("unexpected solvers page1: rows=%+v pagination=%+v", solversPage1, solversPagination1)
	}

	solversPage2, solversPagination2, err := env.wargameSvc.ChallengeSolversPage(context.Background(), chWeb.ID, 2, 1)
	if err != nil {
		t.Fatalf("ChallengeSolversPage page2: %v", err)
	}

	if len(solversPage2) != 1 || solversPagination2.TotalCount != 2 || solversPagination2.HasNext {
		t.Fatalf("unexpected solvers page2: rows=%+v pagination=%+v", solversPage2, solversPagination2)
	}

	if _, _, err := env.wargameSvc.ChallengeSolversPage(context.Background(), 0, 1, 20); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	if _, _, err := env.wargameSvc.ChallengeSolversPage(context.Background(), 999999, 1, 20); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestWargameServiceLevelFilterValidation(t *testing.T) {
	env := setupServiceTest(t)

	level := 11
	_, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{Level: &level})
	if err == nil {
		t.Fatalf("expected validation error for invalid level filter")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error type, got %v", err)
	}
}

func TestWargameServiceChallengeSortValidationAndOrder(t *testing.T) {
	env := setupServiceTest(t)
	user1 := createUser(t, env, "sort1@example.com", "sort1", "pass", models.UserRole)
	user2 := createUser(t, env, "sort2@example.com", "sort2", "pass", models.UserRole)
	blocked := createUser(t, env, "sortblocked@example.com", "sortblocked", "pass", models.BlockedRole)
	adminUser := createUser(t, env, "sortadmin@example.com", "sortadmin", "pass", models.AdminRole)

	chA, err := env.wargameSvc.CreateChallenge(context.Background(), "Sort A", "Desc", "Web", 100, "FLAG{A}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge A: %v", err)
	}

	chB, err := env.wargameSvc.CreateChallenge(context.Background(), "Sort B", "Desc", "Web", 100, "FLAG{B}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge B: %v", err)
	}

	chC, err := env.wargameSvc.CreateChallenge(context.Background(), "Sort C", "Desc", "Web", 100, "FLAG{C}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge C: %v", err)
	}

	chD, err := env.wargameSvc.CreateChallenge(context.Background(), "Sort D", "Desc", "Web", 100, "FLAG{D}", true, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateChallenge D: %v", err)
	}

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, chA.ID, true, now.Add(-5*time.Minute))
	createSubmission(t, env, user2.ID, chA.ID, true, now.Add(-4*time.Minute))
	createSubmission(t, env, user1.ID, chB.ID, true, now.Add(-3*time.Minute))
	createSubmission(t, env, user2.ID, chD.ID, true, now.Add(-150*time.Second))
	createSubmission(t, env, blocked.ID, chC.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, adminUser.ID, chC.ID, true, now.Add(-1*time.Minute))

	assertOrder := func(rows []models.Challenge, want []int64) {
		if len(rows) < len(want) {
			t.Fatalf("rows too short: got=%d want=%d", len(rows), len(want))
		}

		for i := range want {
			if rows[i].ID != want[i] {
				t.Fatalf("unexpected order at %d: got=%d want=%d", i, rows[i].ID, want[i])
			}
		}
	}

	latest, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{Sort: "latest"})
	if err != nil {
		t.Fatalf("ListChallenges latest: %v", err)
	}
	assertOrder(latest, []int64{chD.ID, chC.ID, chB.ID, chA.ID})

	oldest, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{Sort: "oldest"})
	if err != nil {
		t.Fatalf("ListChallenges oldest: %v", err)
	}
	assertOrder(oldest, []int64{chA.ID, chB.ID, chC.ID, chD.ID})

	most, _, err := env.wargameSvc.SearchChallenges(context.Background(), "Sort", 1, 20, ChallengeQueryFilter{Sort: "most_solved"})
	if err != nil {
		t.Fatalf("SearchChallenges most_solved: %v", err)
	}
	assertOrder(most, []int64{chA.ID, chD.ID, chC.ID, chB.ID})

	least, _, err := env.wargameSvc.SearchChallenges(context.Background(), "Sort", 1, 20, ChallengeQueryFilter{Sort: "least_solved"})
	if err != nil {
		t.Fatalf("SearchChallenges least_solved: %v", err)
	}
	assertOrder(least, []int64{chD.ID, chC.ID, chB.ID, chA.ID})

	if _, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{Sort: "invalid"}); err == nil {
		t.Fatalf("expected invalid sort validation error")
	}
}

func TestWargameServiceUpdateChallengeCreator(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Creator Challenge", 100, "FLAG{CREATOR}", true)
	user := createUser(t, env, "creator@example.com", "creator-user", "pass", models.UserRole)

	t.Run("invalid input", func(t *testing.T) {
		if err := env.wargameSvc.UpdateChallengeCreator(context.Background(), 0, user.ID); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if err := env.wargameSvc.UpdateChallengeCreator(context.Background(), 999999, user.ID); !errors.Is(err, ErrChallengeNotFound) {
			t.Fatalf("expected ErrChallengeNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		if err := env.wargameSvc.UpdateChallengeCreator(context.Background(), challenge.ID, user.ID); err != nil {
			t.Fatalf("UpdateChallengeCreator: %v", err)
		}

		updated, err := env.wargameSvc.GetChallengeByID(context.Background(), challenge.ID)
		if err != nil {
			t.Fatalf("GetChallengeByID: %v", err)
		}
		if updated.CreatedByUserID == nil || *updated.CreatedByUserID != user.ID {
			t.Fatalf("expected created_by_user_id=%d, got %+v", user.ID, updated.CreatedByUserID)
		}
	})
}

func TestWargameServiceListAllSubmissions(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "sub@example.com", "sub", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Sub", 100, "FLAG{SUB}", true)

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC().Add(-time.Minute))
	createSubmission(t, env, user.ID, challenge.ID, false, time.Now().UTC())

	subs, err := env.wargameSvc.ListAllSubmissions(context.Background())
	if err != nil {
		t.Fatalf("ListAllSubmissions: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(subs))
	}
}

func TestWargameServiceValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.wargameSvc.GetChallengeByID(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if _, err := env.wargameSvc.GetChallengeByID(context.Background(), 999999); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}

	if _, err := env.wargameSvc.CreateChallenge(context.Background(), "", "", "Nope", -1, "", true, false, nil, nil, nil); err == nil {
		t.Fatalf("expected create validation error")
	}
	badPts := 50
	challenge := createChallenge(t, env, "X", 100, "FLAG{X}", true)
	if _, err := env.wargameSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, &badPts, nil, nil, nil, nil, nil, nil, false); err != nil {
		t.Fatalf("unexpected update error in fixed scoring mode: %v", err)
	}
}

func TestWargameServiceSubmitFlagLockedAndInactive(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "lock@example.com", "lock", "pass", models.UserRole)
	prev := createChallenge(t, env, "Prev", 50, "FLAG{PREV}", true)
	locked := createChallenge(t, env, "Locked", 100, "FLAG{LOCK}", true)
	locked.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), locked); err != nil {
		t.Fatalf("update locked challenge: %v", err)
	}

	if _, err := env.wargameSvc.SubmitFlag(context.Background(), user.ID, locked.ID, "FLAG{LOCK}"); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("expected ErrChallengeLocked, got %v", err)
	}

	createSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())
	correct, err := env.wargameSvc.SubmitFlag(context.Background(), user.ID, locked.ID, "FLAG{LOCK}")
	if err != nil || !correct {
		t.Fatalf("expected unlocked solve, correct=%v err=%v", correct, err)
	}

	inactive := createChallenge(t, env, "Inactive", 50, "FLAG{I}", false)
	if _, err := env.wargameSvc.SubmitFlag(context.Background(), user.ID, inactive.ID, "FLAG{I}"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestWargameServiceFileUploadDownloadDeleteFlow(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Zip", 100, "FLAG{ZIP}", true)

	updated, upload, err := env.wargameSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("RequestChallengeFileUpload: %v", err)
	}
	if upload.URL == "" || updated.FileKey == nil || updated.FileName == nil {
		t.Fatalf("expected upload metadata, got challenge=%+v upload=%+v", updated, upload)
	}

	download, err := env.wargameSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID)
	if err != nil {
		t.Fatalf("RequestChallengeFileDownload: %v", err)
	}
	if download.URL == "" {
		t.Fatalf("expected download url")
	}

	cleared, err := env.wargameSvc.DeleteChallengeFile(context.Background(), challenge.ID)
	if err != nil {
		t.Fatalf("DeleteChallengeFile: %v", err)
	}
	if cleared.FileKey != nil || cleared.FileName != nil || cleared.FileUploadedAt != nil {
		t.Fatalf("expected file fields cleared, got %+v", cleared)
	}
}

func TestWargameServiceFileErrorPaths(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Zip", 100, "FLAG{ZIP}", true)

	if _, _, err := env.wargameSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "file.txt"); err == nil {
		t.Fatalf("expected filename validation error")
	}

	voteRepo := repo.NewChallengeVoteRepo(serviceDB)
	if _, _, err := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, nil).RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}

	if _, _, err := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, errorFileStore{uploadErr: errors.New("presign fail")}).RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip"); err == nil || !strings.Contains(err.Error(), "presign") {
		t.Fatalf("expected presign error, got %v", err)
	}

	if _, err := env.wargameSvc.RequestChallengeFileDownload(context.Background(), 0, challenge.ID); !errors.Is(err, ErrChallengeFileNotFound) {
		t.Fatalf("expected ErrChallengeFileNotFound, got %v", err)
	}

	_, _, err := env.wargameSvc.RequestChallengeFileUpload(context.Background(), challenge.ID, "bundle.zip")
	if err != nil {
		t.Fatalf("upload seed: %v", err)
	}

	if _, err := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, errorFileStore{downloadErr: errors.New("download fail")}).RequestChallengeFileDownload(context.Background(), 0, challenge.ID); err == nil || !strings.Contains(err.Error(), "presign") {
		t.Fatalf("expected download presign error, got %v", err)
	}

	if _, err := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, nil).RequestChallengeFileDownload(context.Background(), 0, challenge.ID); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}

	if _, err := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, errorFileStore{deleteErr: errors.New("delete fail")}).DeleteChallengeFile(context.Background(), challenge.ID); err == nil || !strings.Contains(err.Error(), "delete") {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestWargameServiceStackValidationAndSolvedIDsEdge(t *testing.T) {
	env := setupServiceTest(t)
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"

	if _, err := env.wargameSvc.CreateChallenge(context.Background(), "StackBad", "Desc", "Web", 100, "FLAG{S}", true, true, nil, &podSpec, nil); err == nil {
		t.Fatalf("expected missing stack_target_ports validation error")
	}

	badPorts := stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "ICMP"}}
	if _, err := env.wargameSvc.CreateChallenge(context.Background(), "StackBad2", "Desc", "Web", 100, "FLAG{S2}", true, true, badPorts, &podSpec, nil); err == nil {
		t.Fatalf("expected invalid protocol validation error")
	}

	ids, err := env.wargameSvc.SolvedChallengeIDs(context.Background(), 0)
	if err != nil || len(ids) != 0 {
		t.Fatalf("expected empty solved ids for user 0, ids=%v err=%v", ids, err)
	}

	voteRepo := repo.NewChallengeVoteRepo(serviceDB)
	nilRepoSvc := NewWargameService(env.cfg, env.challengeRepo, nil, voteRepo, env.redis, storage.NewMemoryChallengeFileStore(10*time.Minute))
	ids, err = nilRepoSvc.SolvedChallengeIDs(context.Background(), 1)
	if err != nil || len(ids) != 0 {
		t.Fatalf("expected empty solved ids with nil repo, ids=%v err=%v", ids, err)
	}
}

func TestWargameServiceErrorPathsWithClosedDB(t *testing.T) {
	if skipServiceEnv {
		t.Skip("integration tests disabled via WARGAME_SKIP_INTEGRATION")
	}

	closedDB := newClosedServiceDB(t)
	challengeRepo := repo.NewChallengeRepo(closedDB)
	submissionRepo := repo.NewSubmissionRepo(closedDB)
	voteRepo := repo.NewChallengeVoteRepo(closedDB)
	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)
	wargameSvc := NewWargameService(serviceCfg, challengeRepo, submissionRepo, voteRepo, serviceRedis, fileStore)

	if _, _, err := wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{}); err == nil {
		t.Fatalf("expected ListChallenges error")
	}
	if _, err := wargameSvc.SubmitFlag(context.Background(), 1, 1, "flag{err}"); err == nil {
		t.Fatalf("expected SubmitFlag error")
	}
}

func TestWargameServiceUpdateChallengeFlagHashAndPrevious(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Old", 100, "FLAG{OLD}", true)
	prev := createChallenge(t, env, "Prev", 50, "FLAG{PREV}", true)
	newFlag := "FLAG{NEW}"

	updated, err := env.wargameSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, &newFlag, nil, nil, nil, nil, &prev.ID, true)
	if err != nil {
		t.Fatalf("UpdateChallenge with flag/previous: %v", err)
	}
	if updated.PreviousChallengeID == nil || *updated.PreviousChallengeID != prev.ID {
		t.Fatalf("expected previous challenge set, got %+v", updated.PreviousChallengeID)
	}
	ok, err := utils.CheckFlag(updated.FlagHash, newFlag)
	if err != nil || !ok {
		t.Fatalf("expected updated flag hash")
	}

	selfID := challenge.ID
	if _, err := env.wargameSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, &selfID, true); err == nil {
		t.Fatalf("expected self previous_challenge_id validation error")
	}

	nilPrev := (*int64)(nil)
	if _, err := env.wargameSvc.UpdateChallenge(context.Background(), challenge.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nilPrev, true); err != nil {
		t.Fatalf("expected previous_challenge_id clear, got %v", err)
	}
}

func TestWargameServiceUploadReplacesOldFileBestEffort(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createChallenge(t, env, "Zip", 100, "FLAG{ZIP}", true)

	oldKey := uuid.NewString() + ".zip"
	now := time.Now().UTC()
	challenge.FileKey = &oldKey
	challenge.FileName = ptrString("old.zip")
	challenge.FileUploadedAt = &now
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("seed old file: %v", err)
	}

	voteRepo := repo.NewChallengeVoteRepo(serviceDB)
	svc := NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, errorFileStore{deleteErr: errors.New("best effort")})
	updated, _, err := svc.RequestChallengeFileUpload(context.Background(), challenge.ID, "new.zip")
	if err != nil {
		t.Fatalf("expected upload success despite delete failure, got %v", err)
	}
	if updated.FileKey == nil || *updated.FileKey == oldKey {
		t.Fatalf("expected file key replaced, got %+v", updated.FileKey)
	}
}
func TestWargameServiceVoteChallengeLevelAndChallengeVotesPage(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vote-user@example.com", "vote-user", "pass", models.UserRole)
	challenge := createChallenge(t, env, "level-target", 100, "FLAG{LV}", true)

	if err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, challenge.ID, 7); !errors.Is(err, ErrChallengeNotSolvedByUser) {
		t.Fatalf("expected ErrChallengeNotSolvedByUser, got %v", err)
	}

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	if err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, challenge.ID, 7); err != nil {
		t.Fatalf("VoteChallengeLevel: %v", err)
	}

	votes, pagination, err := env.wargameSvc.ChallengeVotesPage(context.Background(), challenge.ID, 1, 10)
	if err != nil {
		t.Fatalf("ChallengeVotesPage: %v", err)
	}

	if len(votes) != 1 || votes[0].Level != 7 || votes[0].UserID != user.ID {
		t.Fatalf("unexpected votes: %+v", votes)
	}

	if pagination.TotalCount != 1 {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	detail, err := env.wargameSvc.GetChallengeByID(context.Background(), challenge.ID)
	if err != nil {
		t.Fatalf("GetChallengeByID: %v", err)
	}

	if detail.Level != 7 {
		t.Fatalf("expected representative level 7, got %d", detail.Level)
	}

	if len(detail.LevelVotes) != 1 || detail.LevelVotes[0].Level != 7 || detail.LevelVotes[0].Count != 1 {
		t.Fatalf("unexpected level vote counts: %+v", detail.LevelVotes)
	}
}

func TestWargameServiceVoteChallengeLevelValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vote2@example.com", "vote2", "pass", models.UserRole)

	err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, 1, 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for level=0, got %v", err)
	}

	if err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, 999999, 3); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}

	_, _, err = env.wargameSvc.ChallengeVotesPage(context.Background(), 0, 1, 10)
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error for challenge id, got %v", err)
	}
}

func TestWargameServiceApplyChallengeLevelsUnknownByDefault(t *testing.T) {
	env := setupServiceTest(t)
	ch1 := createChallenge(t, env, "no-votes-1", 100, "FLAG{N1}", true)
	ch2 := createChallenge(t, env, "no-votes-2", 100, "FLAG{N2}", true)

	rows, _, err := env.wargameSvc.ListChallenges(context.Background(), 1, 20, ChallengeQueryFilter{})
	if err != nil {
		t.Fatalf("ListChallenges: %v", err)
	}

	levels := map[int64]int{}
	for _, row := range rows {
		levels[row.ID] = row.Level
	}

	if levels[ch1.ID] != models.UnknownLevel || levels[ch2.ID] != models.UnknownLevel {
		t.Fatalf("expected unknown levels for no-vote challenges: %+v", levels)
	}
}

func TestWargameServiceChallengeVotesPageNotFound(t *testing.T) {
	env := setupServiceTest(t)
	if _, _, err := env.wargameSvc.ChallengeVotesPage(context.Background(), 999999, 1, 10); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestWargameServiceVoteChallengeLevelRepoFailures(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "vote3@example.com", "vote3", "pass", models.UserRole)
	challenge := createChallenge(t, env, "repo-fail", 100, "FLAG{RF}", true)
	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	closedDB := newClosedServiceDB(t)
	voteRepo := repo.NewChallengeVoteRepo(closedDB)
	origSvc := env.wargameSvc
	env.wargameSvc = NewWargameService(env.cfg, env.challengeRepo, env.submissionRepo, voteRepo, env.redis, origSvc.fileStore)

	if err := env.wargameSvc.VoteChallengeLevel(context.Background(), user.ID, challenge.ID, 5); err == nil {
		t.Fatalf("expected vote repo failure")
	}
}
