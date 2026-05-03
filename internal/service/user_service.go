package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/storage"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo        *repo.UserRepo
	affiliationRepo *repo.AffiliationRepo
	profileStore    storage.ProfileImageStore
}

func NewUserService(userRepo *repo.UserRepo, affiliationRepo *repo.AffiliationRepo, profileStore storage.ProfileImageStore) *UserService {
	return &UserService{userRepo: userRepo, affiliationRepo: affiliationRepo, profileStore: profileStore}
}

const maxProfileImageBytes int64 = 100 * 1024

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

func (s *UserService) UpdateProfile(ctx context.Context, userID int64, username *string, affiliationID *int64, affiliationSet bool, bio *string, bioSet bool) (*models.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != nil {
		normalizedUsername := normalizeTrim(*username)
		if normalizedUsername == "" {
			return nil, NewValidationError(FieldError{Field: "username", Reason: "required"})
		}

		exists, err := s.userRepo.ExistsByUsername(ctx, normalizedUsername, &userID)
		if err != nil {
			return nil, fmt.Errorf("user.UpdateProfile username exists: %w", err)
		}

		if exists {
			return nil, ErrUserExists
		}

		user.Username = normalizedUsername
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

	if bioSet {
		user.Bio = bio
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrUserExists
		}

		return nil, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.UpdateProfile reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) RequestProfileImageUpload(ctx context.Context, userID int64, filename string) (*models.User, storage.PresignedUpload, error) {
	filename = normalizeTrim(filename)
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.Required("filename", filename)
	if err := validator.Error(); err != nil {
		return nil, storage.PresignedUpload{}, err
	}

	if s.profileStore == nil {
		return nil, storage.PresignedUpload{}, ErrStorageUnavailable
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, storage.PresignedUpload{}, ErrNotFound
		}
		return nil, storage.PresignedUpload{}, fmt.Errorf("user.RequestProfileImageUpload lookup: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := ""
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	default:
		return nil, storage.PresignedUpload{}, NewValidationError(FieldError{Field: "filename", Reason: "must be a .png, .jpg, or .jpeg file"})
	}

	key := "profiles/" + uuid.NewString() + ext
	upload, err := s.profileStore.PresignUpload(ctx, key, contentType, maxProfileImageBytes)
	if err != nil {
		return nil, storage.PresignedUpload{}, fmt.Errorf("user.RequestProfileImageUpload presign: %w", err)
	}

	return user, upload, nil
}

func (s *UserService) FinalizeProfileImageUpload(ctx context.Context, userID int64, key string) (*models.User, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.Required("key", normalizeTrim(key))
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if s.profileStore == nil {
		return nil, ErrStorageUnavailable
	}

	normalizedKey, err := normalizeProfileImageKey(key)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user.FinalizeProfileImageUpload lookup: %w", err)
	}

	oldKey := ""
	if user.ProfileImage != nil {
		oldKey = strings.TrimSpace(*user.ProfileImage)
	}

	user.ProfileImage = &normalizedKey
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.FinalizeProfileImageUpload update: %w", err)
	}

	if oldKey != "" && oldKey != normalizedKey {
		_ = s.profileStore.Delete(ctx, oldKey)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.FinalizeProfileImageUpload reload: %w", err)
	}

	return updated, nil
}

func (s *UserService) DeleteProfileImage(ctx context.Context, userID int64) (*models.User, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("user.DeleteProfileImage lookup: %w", err)
	}

	if s.profileStore == nil {
		return nil, ErrStorageUnavailable
	}

	oldKey := ""
	if user.ProfileImage != nil {
		oldKey = strings.TrimSpace(*user.ProfileImage)
	}

	user.ProfileImage = nil
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("user.DeleteProfileImage update: %w", err)
	}

	if oldKey != "" {
		_ = s.profileStore.Delete(ctx, oldKey)
	}

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.DeleteProfileImage reload: %w", err)
	}

	return updated, nil
}

func normalizeProfileImageKey(key string) (string, error) {
	k := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(key), "/"))
	if !strings.HasPrefix(k, "profiles/") {
		return "", NewValidationError(FieldError{Field: "key", Reason: "must start with profiles/"})
	}

	filename := strings.TrimPrefix(k, "profiles/")
	if filename == "" || strings.Contains(filename, "/") {
		return "", NewValidationError(FieldError{Field: "key", Reason: "invalid profile image key"})
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return "", NewValidationError(FieldError{Field: "key", Reason: "must end with .png, .jpg, or .jpeg"})
	}

	id := strings.TrimSuffix(filename, ext)
	if _, err := uuid.Parse(id); err != nil {
		return "", NewValidationError(FieldError{Field: "key", Reason: "must include UUID filename"})
	}

	return k, nil
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
