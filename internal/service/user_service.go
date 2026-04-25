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

type UserService struct {
	userRepo        *repo.UserRepo
	affiliationRepo *repo.AffiliationRepo
}

func NewUserService(userRepo *repo.UserRepo, affiliationRepo *repo.AffiliationRepo) *UserService {
	return &UserService{userRepo: userRepo, affiliationRepo: affiliationRepo}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.GetByID: %w", err)
	}

	return user, nil
}

func (s *UserService) List(ctx context.Context) ([]models.User, models.Pagination, error) {
	params, err := NormalizePagination(0, 0)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	return s.ListPage(ctx, params.Page, params.PageSize)
}

func (s *UserService) ListPage(ctx context.Context, page, pageSize int) ([]models.User, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	users, totalCount, err := s.userRepo.List(ctx, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("user.ListPage: %w", err)
	}

	return users, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *UserService) Search(ctx context.Context, query string, page, pageSize int) ([]models.User, models.Pagination, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, models.Pagination{}, NewValidationError(FieldError{Field: "q", Reason: "required"})
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	users, totalCount, err := s.userRepo.Search(ctx, query, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("user.Search: %w", err)
	}

	return users, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int64, username *string, affiliationID *int64, affiliationSet bool) (*models.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != nil {
		user.Username = *username
	}

	if affiliationSet {
		if affiliationID == nil {
			user.AffiliationID = nil
		} else {
			if s.affiliationRepo == nil {
				return nil, NewValidationError(FieldError{Field: "affiliation_id", Reason: "invalid"})
			}

			exists, err := s.affiliationRepo.ExistsByID(ctx, *affiliationID)
			if err != nil {
				return nil, fmt.Errorf("user.UpdateProfile affiliation exists: %w", err)
			}

			if !exists {
				return nil, NewValidationError(FieldError{Field: "affiliation_id", Reason: "invalid"})
			}

			user.AffiliationID = affiliationID
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.UpdateProfile reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) BlockUser(ctx context.Context, userID int64, reason string) (*models.User, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, NewValidationError(FieldError{Field: "reason", Reason: "required"})
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.BlockUser lookup: %w", err)
	}

	if user.Role == models.AdminRole {
		return nil, NewValidationError(FieldError{Field: "user_id", Reason: "admin_blocked"})
	}

	user.Role = models.BlockedRole
	user.BlockedReason = &reason
	blockedAt := time.Now().UTC()
	user.BlockedAt = &blockedAt
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.BlockUser update: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.BlockUser reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) ListByAffiliation(ctx context.Context, affiliationID int64, page, pageSize int) ([]models.User, models.Pagination, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", affiliationID)
	if err := validator.Error(); err != nil {
		return nil, models.Pagination{}, err
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.userRepo.ListByAffiliation(ctx, affiliationID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("user.ListByAffiliation: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *UserService) UnblockUser(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.UnblockUser lookup: %w", err)
	}

	if user.Role == models.AdminRole {
		return nil, NewValidationError(FieldError{Field: "user_id", Reason: "admin_blocked"})
	}

	user.Role = models.UserRole
	user.BlockedReason = nil
	user.BlockedAt = nil
	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.UnblockUser update: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.UnblockUser reload: %w", err)
	}

	return updated, nil
}
