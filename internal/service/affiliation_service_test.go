package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestAffiliationServiceCreateListGet(t *testing.T) {
	env := setupServiceTest(t)

	created, err := env.affiliationSvc.Create(context.Background(), "Blue Team")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if created.ID <= 0 || created.Name != "Blue Team" {
		t.Fatalf("unexpected created row: %+v", created)
	}

	_, err = env.affiliationSvc.Create(context.Background(), "blue team")
	if err == nil {
		t.Fatalf("expected duplicate name error")
	}

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected validation error, got %v", err)
	}

	got, err := env.affiliationSvc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.Name != "Blue Team" {
		t.Fatalf("unexpected get row: %+v", got)
	}

	another := &models.Affiliation{Name: "Alpha", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := env.affiliationRepo.Create(context.Background(), another); err != nil {
		t.Fatalf("seed affiliation: %v", err)
	}

	rows, pagination, err := env.affiliationSvc.List(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(rows) != 1 || rows[0].Name != "Alpha" {
		t.Fatalf("unexpected rows: %+v", rows)
	}

	if pagination.TotalCount != 2 || !pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	filteredRows, filteredPagination, err := env.affiliationSvc.Search(context.Background(), "blue", 1, 10)
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}

	if len(filteredRows) != 1 || filteredRows[0].Name != "Blue Team" {
		t.Fatalf("unexpected filtered rows: %+v", filteredRows)
	}

	if filteredPagination.TotalCount != 1 {
		t.Fatalf("unexpected filtered pagination: %+v", filteredPagination)
	}
}

func TestAffiliationServiceValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.affiliationSvc.Create(context.Background(), " "); err == nil {
		t.Fatalf("expected name required validation")
	}

	if _, err := env.affiliationSvc.GetByID(context.Background(), 0); err == nil {
		t.Fatalf("expected invalid id validation")
	}

	if _, err := env.affiliationSvc.GetByID(context.Background(), 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}

	if _, _, err := env.affiliationSvc.Search(context.Background(), " ", 1, 10); err == nil {
		t.Fatalf("expected required query validation for search")
	}

	if _, _, err := env.affiliationSvc.List(context.Background(), -1, 10); err == nil {
		t.Fatalf("expected invalid pagination validation for list")
	}
}
