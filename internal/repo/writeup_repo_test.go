package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestWriteupRepoCRUDAndLookup(t *testing.T) {
	env := setupRepoTest(t)
	writeupRepo := NewWriteupRepo(env.db)

	user := createUserForTestUserScope(t, env, "writeup-crud@example.com", "writeup_crud", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Writeup CRUD", 250, "FLAG{WCRUD}", true)

	now := time.Now().UTC().Truncate(time.Microsecond)
	row := &models.Writeup{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		Content:     "initial body",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := writeupRepo.Create(context.Background(), row); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if row.ID <= 0 {
		t.Fatalf("expected writeup id > 0, got %d", row.ID)
	}

	gotByID, err := writeupRepo.GetByID(context.Background(), row.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if gotByID.Content != "initial body" || gotByID.UserID != user.ID || gotByID.ChallengeID != challenge.ID {
		t.Fatalf("unexpected writeup from GetByID: %+v", gotByID)
	}

	gotByUserChallenge, err := writeupRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByUserAndChallenge: %v", err)
	}

	if gotByUserChallenge.ID != row.ID {
		t.Fatalf("expected same writeup id, got %d", gotByUserChallenge.ID)
	}

	row.Content = "updated body"
	row.UpdatedAt = now.Add(10 * time.Minute)
	if err := writeupRepo.Update(context.Background(), row); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := writeupRepo.GetByID(context.Background(), row.ID)
	if err != nil {
		t.Fatalf("GetByID updated: %v", err)
	}

	if updated.Content != "updated body" {
		t.Fatalf("expected updated content, got %q", updated.Content)
	}

	if err := writeupRepo.DeleteByID(context.Background(), row.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	if _, err := writeupRepo.GetByID(context.Background(), row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	if _, err := writeupRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound by user/challenge after delete, got %v", err)
	}
}

func TestWriteupRepoDetailAndPaging(t *testing.T) {
	env := setupRepoTest(t)
	writeupRepo := NewWriteupRepo(env.db)

	aff := &models.Affiliation{
		Name:      "Semyeong Security",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if _, err := env.db.NewInsert().Model(aff).Exec(context.Background()); err != nil {
		t.Fatalf("insert affiliation: %v", err)
	}

	user1 := createUserForTestUserScope(t, env, "writeup-page-u1@example.com", "writeup_u1", "pass", models.UserRole)
	user2 := createUserForTestUserScope(t, env, "writeup-page-u2@example.com", "writeup_u2", "pass", models.UserRole)

	bio := "web + rev"
	if _, err := env.db.NewUpdate().
		Model((*models.User)(nil)).
		Set("affiliation_id = ?", aff.ID).
		Set("bio = ?", bio).
		Where("id = ?", user1.ID).
		Exec(context.Background()); err != nil {
		t.Fatalf("update user profile fields: %v", err)
	}

	ch1 := createChallenge(t, env, "Writeup Challenge 1", 111, "FLAG{WP1}", true)
	ch2 := createChallenge(t, env, "Writeup Challenge 2", 222, "FLAG{WP2}", true)

	base := time.Now().UTC().Truncate(time.Microsecond)
	w1 := &models.Writeup{
		UserID:      user1.ID,
		ChallengeID: ch1.ID,
		Content:     "u1-c1-older",
		CreatedAt:   base.Add(-20 * time.Minute),
		UpdatedAt:   base.Add(-15 * time.Minute),
	}
	w2 := &models.Writeup{
		UserID:      user2.ID,
		ChallengeID: ch1.ID,
		Content:     "u2-c1-newer",
		CreatedAt:   base.Add(-10 * time.Minute),
		UpdatedAt:   base.Add(-9 * time.Minute),
	}
	w3 := &models.Writeup{
		UserID:      user1.ID,
		ChallengeID: ch2.ID,
		Content:     "u1-c2-newest-updated",
		CreatedAt:   base.Add(-30 * time.Minute),
		UpdatedAt:   base.Add(-1 * time.Minute),
	}

	for _, row := range []*models.Writeup{w1, w2, w3} {
		if err := writeupRepo.Create(context.Background(), row); err != nil {
			t.Fatalf("Create writeup row: %v", err)
		}
	}

	detail, err := writeupRepo.GetDetailByID(context.Background(), w2.ID)
	if err != nil {
		t.Fatalf("GetDetailByID: %v", err)
	}

	if detail.Username != user2.Username || detail.ChallengeTitle != ch1.Title || detail.ChallengeCategory != ch1.Category || detail.ChallengePoints != ch1.Points {
		t.Fatalf("unexpected detail fields: %+v", detail)
	}

	detailU1, err := writeupRepo.GetDetailByID(context.Background(), w1.ID)
	if err != nil {
		t.Fatalf("GetDetailByID user1: %v", err)
	}

	if detailU1.Affiliation == nil || *detailU1.Affiliation != aff.Name || detailU1.Bio == nil || *detailU1.Bio != bio {
		t.Fatalf("expected affiliation/bio in detail join, got %+v", detailU1)
	}

	challengeRows, challengeTotal, err := writeupRepo.ChallengePage(context.Background(), ch1.ID, 1, 1)
	if err != nil {
		t.Fatalf("ChallengePage: %v", err)
	}

	if challengeTotal != 2 || len(challengeRows) != 1 || challengeRows[0].ID != w2.ID {
		t.Fatalf("unexpected challenge page rows=%+v total=%d", challengeRows, challengeTotal)
	}

	challengeRowsPage2, _, err := writeupRepo.ChallengePage(context.Background(), ch1.ID, 2, 1)
	if err != nil {
		t.Fatalf("ChallengePage page2: %v", err)
	}

	if len(challengeRowsPage2) != 1 || challengeRowsPage2[0].ID != w1.ID {
		t.Fatalf("unexpected challenge page2 rows=%+v", challengeRowsPage2)
	}

	userRows, userTotal, err := writeupRepo.UserPage(context.Background(), user1.ID, 1, 1)
	if err != nil {
		t.Fatalf("UserPage: %v", err)
	}

	if userTotal != 2 || len(userRows) != 1 || userRows[0].ID != w3.ID {
		t.Fatalf("unexpected user page rows=%+v total=%d", userRows, userTotal)
	}

	userRowsPage2, _, err := writeupRepo.UserPage(context.Background(), user1.ID, 2, 1)
	if err != nil {
		t.Fatalf("UserPage page2: %v", err)
	}

	if len(userRowsPage2) != 1 || userRowsPage2[0].ID != w1.ID {
		t.Fatalf("unexpected user page2 rows=%+v", userRowsPage2)
	}
}

func TestWriteupRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	writeupRepo := NewWriteupRepo(env.db)

	if _, err := writeupRepo.GetByID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from GetByID, got %v", err)
	}

	if _, err := writeupRepo.GetByUserAndChallenge(context.Background(), 999999, 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from GetByUserAndChallenge, got %v", err)
	}

	if _, err := writeupRepo.GetDetailByID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound from GetDetailByID, got %v", err)
	}
}
