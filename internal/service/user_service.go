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
	userRepo *repo.UserRepo
}

func NewUserService(userRepo *repo.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
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

func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("user.List: %w", err)
	}

	return users, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int64, username *string) (*models.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != nil {
		user.Username = *username
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	return user, nil
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
