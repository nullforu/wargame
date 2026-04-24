package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"wargame/internal/models"
	"wargame/internal/repo"
)

type AffiliationService struct {
	repo *repo.AffiliationRepo
}

func NewAffiliationService(affiliationRepo *repo.AffiliationRepo) *AffiliationService {
	return &AffiliationService{repo: affiliationRepo}
}

func (s *AffiliationService) Create(ctx context.Context, name string) (*models.Affiliation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError(FieldError{Field: "name", Reason: "required"})
	}

	exists, err := s.repo.ExistsByNameCI(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("affiliation.Create exists: %w", err)
	}

	if exists {
		return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
	}

	affiliation := &models.Affiliation{
		Name:      name,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, affiliation); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "idx_affiliations_name_unique_lower") || strings.Contains(msg, "duplicate key") {
			return nil, NewValidationError(FieldError{Field: "name", Reason: "duplicate"})
		}

		return nil, fmt.Errorf("affiliation.Create: %w", err)
	}

	return affiliation, nil
}

func (s *AffiliationService) List(ctx context.Context, page, pageSize int) ([]models.Affiliation, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.List(ctx, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("affiliation.List: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *AffiliationService) Search(ctx context.Context, query string, page, pageSize int) ([]models.Affiliation, models.Pagination, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, models.Pagination{}, NewValidationError(FieldError{Field: "q", Reason: "required"})
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.Search(ctx, query, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("affiliation.Search: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *AffiliationService) GetByID(ctx context.Context, id int64) (*models.Affiliation, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("affiliation.GetByID: %w", err)
	}

	return row, nil
}
