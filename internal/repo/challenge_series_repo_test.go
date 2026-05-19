package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/db"
	"wargame/internal/models"
)

func TestChallengeSeriesRepoCRUDAndList(t *testing.T) {
	env := setupRepoTest(t)
	seriesRepo := NewChallengeSeriesRepo(env.db)
	user := createUser(t, env, "series-repo@example.com", "seriesrepo", "pass", models.AdminRole)
	now := time.Now().UTC()

	series := &models.ChallengeSeries{
		Title:           "Repo Series",
		Description:     "desc",
		CreatedByUserID: &user.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := seriesRepo.Create(context.Background(), series); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := seriesRepo.GetByID(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Title != "Repo Series" {
		t.Fatalf("unexpected title: %s", got.Title)
	}

	list, total, err := seriesRepo.List(context.Background(), 1, 20, "latest")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if total != 1 || len(list) != 1 {
		t.Fatalf("unexpected list result: total=%d len=%d", total, len(list))
	}

	series2 := &models.ChallengeSeries{
		Title:           "Repo Series 2",
		Description:     "desc2",
		CreatedByUserID: &user.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := seriesRepo.Create(context.Background(), series2); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	latest, _, err := seriesRepo.List(context.Background(), 1, 20, "latest")
	if err != nil {
		t.Fatalf("List latest: %v", err)
	}

	if len(latest) < 2 || latest[0].ID != series2.ID {
		t.Fatalf("expected latest first id %d, got %+v", series2.ID, latest)
	}

	oldest, _, err := seriesRepo.List(context.Background(), 1, 20, "oldest")
	if err != nil {
		t.Fatalf("List oldest: %v", err)
	}

	if len(oldest) < 2 || oldest[0].ID != series.ID {
		t.Fatalf("expected oldest first id %d, got %+v", series.ID, oldest)
	}

	got.Title = "Repo Series Updated"
	if err := seriesRepo.Update(context.Background(), got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := seriesRepo.GetByID(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("GetByID updated: %v", err)
	}

	if updated.Title != "Repo Series Updated" {
		t.Fatalf("unexpected updated title: %s", updated.Title)
	}

	if err := seriesRepo.DeleteByID(context.Background(), series.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	if _, err := seriesRepo.GetByID(context.Background(), series.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestChallengeSeriesRepoReplaceChallengesAndRollback(t *testing.T) {
	env := setupRepoTest(t)
	seriesRepo := NewChallengeSeriesRepo(env.db)
	ch1 := createChallenge(t, env, "S-1", 100, "FLAG{S1}", true)
	ch2 := createChallenge(t, env, "S-2", 100, "FLAG{S2}", true)

	series := &models.ChallengeSeries{Title: "Replace Series", Description: "desc", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := seriesRepo.Create(context.Background(), series); err != nil {
		t.Fatalf("Create series: %v", err)
	}

	if err := seriesRepo.ReplaceChallenges(context.Background(), series.ID, []int64{ch1.ID, ch2.ID}); err != nil {
		t.Fatalf("ReplaceChallenges: %v", err)
	}

	detail, err := seriesRepo.DetailChallenges(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("DetailChallenges: %v", err)
	}

	if len(detail) != 2 || detail[0].Challenge.ID != ch1.ID || detail[1].Challenge.ID != ch2.ID {
		t.Fatalf("unexpected ordered detail: %+v", detail)
	}

	if err := seriesRepo.ReplaceChallenges(context.Background(), series.ID, []int64{ch1.ID, ch1.ID}); err == nil {
		t.Fatalf("expected unique violation for duplicate challenge id")
	}

	rollbackDetail, err := seriesRepo.DetailChallenges(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("DetailChallenges after rollback: %v", err)
	}

	if len(rollbackDetail) != 2 || rollbackDetail[0].Challenge.ID != ch1.ID || rollbackDetail[1].Challenge.ID != ch2.ID {
		t.Fatalf("expected transaction rollback preserving previous rows, got %+v", rollbackDetail)
	}
}

func TestChallengeSeriesRepoExistsByTitle(t *testing.T) {
	env := setupRepoTest(t)
	seriesRepo := NewChallengeSeriesRepo(env.db)
	now := time.Now().UTC()

	s1 := &models.ChallengeSeries{Title: "Unique Series", Description: "d1", CreatedAt: now, UpdatedAt: now}
	s2 := &models.ChallengeSeries{Title: "Other Series", Description: "d2", CreatedAt: now, UpdatedAt: now}
	if err := seriesRepo.Create(context.Background(), s1); err != nil {
		t.Fatalf("create s1: %v", err)
	}

	if err := seriesRepo.Create(context.Background(), s2); err != nil {
		t.Fatalf("create s2: %v", err)
	}

	exists, err := seriesRepo.ExistsByTitle(context.Background(), "Unique Series", nil)
	if err != nil || !exists {
		t.Fatalf("expected title exists, exists=%v err=%v", exists, err)
	}

	exists, err = seriesRepo.ExistsByTitle(context.Background(), "Unique Series", &s1.ID)
	if err != nil {
		t.Fatalf("ExistsByTitle exclude self: %v", err)
	}

	if exists {
		t.Fatalf("expected false when excluding self")
	}

	exists, err = seriesRepo.ExistsByTitle(context.Background(), "Unique Series", &s2.ID)
	if err != nil {
		t.Fatalf("ExistsByTitle exclude other: %v", err)
	}

	if !exists {
		t.Fatalf("expected true when excluding other")
	}
}

func TestChallengeSeriesRepoErrorPaths(t *testing.T) {
	conn, err := db.New(repoCfg.DB, "test")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	_ = conn.Close()
	repo := NewChallengeSeriesRepo(conn)

	series := &models.ChallengeSeries{ID: 1, Title: "x", Description: "y", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), series); err == nil {
		t.Fatalf("expected create error on closed db")
	}

	if err := repo.Update(context.Background(), series); err == nil {
		t.Fatalf("expected update error on closed db")
	}

	if err := repo.DeleteByID(context.Background(), 1); err == nil {
		t.Fatalf("expected delete error on closed db")
	}

	if _, err := repo.GetByID(context.Background(), 1); err == nil {
		t.Fatalf("expected get error on closed db")
	}

	if _, _, err := repo.List(context.Background(), 1, 20, "latest"); err == nil {
		t.Fatalf("expected list error on closed db")
	}

	if err := repo.ReplaceChallenges(context.Background(), 1, []int64{1}); err == nil {
		t.Fatalf("expected replace error on closed db")
	}

	if _, err := repo.DetailChallenges(context.Background(), 1); err == nil {
		t.Fatalf("expected detail error on closed db")
	}

	if _, err := repo.ExistsByTitle(context.Background(), "x", nil); err == nil {
		t.Fatalf("expected exists error on closed db")
	}
}
