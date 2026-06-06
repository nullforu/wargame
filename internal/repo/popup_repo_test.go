package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestPopupRepoCRUDAndActiveList(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewPopupRepo(env.db)
	user := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	oldTime := time.Now().UTC().Add(-time.Hour)
	key := "popups/first.png"
	name := "first.png"
	link := "https://example.com/first"
	inactiveKey := "popups/inactive.png"
	rows := []*models.Popup{
		{Title: "First", ImageKey: &key, ImageName: &name, LinkURL: &link, IsActive: true, CreatedByUserID: &user.ID, CreatedAt: oldTime, UpdatedAt: oldTime},
		{Title: "Draft", IsActive: false, CreatedByUserID: &user.ID, CreatedAt: oldTime.Add(time.Minute), UpdatedAt: oldTime.Add(time.Minute)},
		{Title: "Inactive", ImageKey: &inactiveKey, IsActive: false, CreatedByUserID: &user.ID, CreatedAt: oldTime.Add(2 * time.Minute), UpdatedAt: oldTime.Add(2 * time.Minute)},
	}

	for _, row := range rows {
		if err := repo.Create(context.Background(), row); err != nil {
			t.Fatalf("create popup: %v", err)
		}
	}

	got, err := repo.GetByID(context.Background(), rows[0].ID)
	if err != nil {
		t.Fatalf("get popup: %v", err)
	}

	if got.Title != "First" || got.ImageKey == nil || *got.ImageKey != key || got.LinkURL == nil || *got.LinkURL != link {
		t.Fatalf("unexpected popup: %+v", got)
	}

	all, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list popups: %v", err)
	}

	if len(all) != 3 || all[0].Title != "Inactive" || all[2].Title != "First" {
		t.Fatalf("expected newest first list, got %+v", all)
	}

	latestKey := "popups/latest.webp"
	rows[1].ImageKey = &latestKey
	rows[1].IsActive = true
	rows[1].UpdatedAt = time.Now().UTC()
	if err := repo.Update(context.Background(), rows[1]); err != nil {
		t.Fatalf("update popup: %v", err)
	}

	active, err := repo.ListActiveWithImages(context.Background())
	if err != nil {
		t.Fatalf("list active popups: %v", err)
	}

	if len(active) != 2 || active[0].Title != "Draft" || active[1].Title != "First" {
		t.Fatalf("expected active popups with images newest first, got %+v", active)
	}

	if err := repo.Delete(context.Background(), rows[0]); err != nil {
		t.Fatalf("delete popup: %v", err)
	}

	if _, err := repo.GetByID(context.Background(), rows[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestPopupRepoGetNotFound(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewPopupRepo(env.db)

	if _, err := repo.GetByID(context.Background(), 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
