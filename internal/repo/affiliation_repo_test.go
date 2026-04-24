package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestAffiliationRepoCreateListGetExists(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewAffiliationRepo(env.db)

	row1 := &models.Affiliation{Name: "Blue Team", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	row2 := &models.Affiliation{Name: "Alpha Club", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), row1); err != nil {
		t.Fatalf("create row1: %v", err)
	}

	if err := repo.Create(context.Background(), row2); err != nil {
		t.Fatalf("create row2: %v", err)
	}

	exists, err := repo.ExistsByID(context.Background(), row1.ID)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}

	if !exists {
		t.Fatalf("expected affiliation to exist")
	}

	got, err := repo.GetByID(context.Background(), row1.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.Name != "Blue Team" {
		t.Fatalf("unexpected row: %+v", got)
	}

	rows, totalCount, err := repo.List(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if totalCount != 2 {
		t.Fatalf("expected total_count 2, got %d", totalCount)
	}

	if len(rows) != 1 || rows[0].Name != "Alpha Club" {
		t.Fatalf("expected alphabetical list order, got %+v", rows)
	}

	searchRows, searchTotal, err := repo.Search(context.Background(), "blue", 1, 10)
	if err != nil {
		t.Fatalf("search list: %v", err)
	}

	if searchTotal != 1 || len(searchRows) != 1 || searchRows[0].Name != "Blue Team" {
		t.Fatalf("unexpected search result: total=%d rows=%+v", searchTotal, searchRows)
	}
}

func TestAffiliationRepoGetNotFound(t *testing.T) {
	env := setupRepoTest(t)
	repo := NewAffiliationRepo(env.db)

	if _, err := repo.GetByID(context.Background(), 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
