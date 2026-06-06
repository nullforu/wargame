package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/storage"

	"github.com/google/uuid"
)

const maxPopupImageBytes int64 = 10 * 1024 * 1024

type PopupService struct {
	repo       *repo.PopupRepo
	mediaStore storage.ProfileImageStore
}

type PopupUpdate struct {
	Title    *string
	TitleSet bool
	LinkURL  *string
	LinkSet  bool
	IsActive *bool
}

func NewPopupService(popupRepo *repo.PopupRepo, mediaStore storage.ProfileImageStore) *PopupService {
	return &PopupService{repo: popupRepo, mediaStore: mediaStore}
}

func (s *PopupService) List(ctx context.Context) ([]models.Popup, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("popup.List: %w", err)
	}

	return rows, nil
}

func (s *PopupService) ListActive(ctx context.Context) ([]models.Popup, error) {
	rows, err := s.repo.ListActiveWithImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("popup.ListActive: %w", err)
	}

	return rows, nil
}

func (s *PopupService) Create(ctx context.Context, title string, linkURL *string, active bool, createdByUserID int64) (*models.Popup, error) {
	title = normalizeTrim(title)
	normalizedLink, linkErr := normalizePopupLinkURL(linkURL)
	validator := newFieldValidator()
	validator.Required("title", title)
	validator.PositiveID("created_by_user_id", createdByUserID)
	if linkErr != nil {
		validator.fields = append(validator.fields, FieldError{Field: "link_url", Reason: "invalid"})
	}

	if active {
		validator.fields = append(validator.fields, FieldError{Field: "is_active", Reason: "image required"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	popup := &models.Popup{
		Title:           title,
		LinkURL:         normalizedLink,
		IsActive:        active,
		CreatedByUserID: &createdByUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, popup); err != nil {
		return nil, fmt.Errorf("popup.Create: %w", err)
	}

	return popup, nil
}

func (s *PopupService) GetByID(ctx context.Context, id int64) (*models.Popup, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	popup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrPopupNotFound
		}

		return nil, fmt.Errorf("popup.GetByID: %w", err)
	}

	return popup, nil
}

func (s *PopupService) Update(ctx context.Context, id int64, update PopupUpdate) (*models.Popup, error) {
	popup, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	validator := newFieldValidator()
	if update.TitleSet {
		if update.Title == nil {
			validator.fields = append(validator.fields, FieldError{Field: "title", Reason: "required"})
		} else {
			title := normalizeTrim(*update.Title)
			validator.Required("title", title)
			popup.Title = title
		}
	}

	if update.LinkSet {
		normalizedLink, err := normalizePopupLinkURL(update.LinkURL)
		if err != nil {
			validator.fields = append(validator.fields, FieldError{Field: "link_url", Reason: "invalid"})
		} else {
			popup.LinkURL = normalizedLink
		}
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if update.IsActive != nil {
		if *update.IsActive && popupImageKey(popup) == "" {
			return nil, NewValidationError(FieldError{Field: "is_active", Reason: "image required"})
		}

		popup.IsActive = *update.IsActive
	}

	popup.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, popup); err != nil {
		return nil, fmt.Errorf("popup.Update: %w", err)
	}

	return popup, nil
}

func (s *PopupService) Delete(ctx context.Context, id int64) error {
	popup, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	oldKey := ""
	if popup.ImageKey != nil {
		oldKey = strings.TrimSpace(*popup.ImageKey)
	}

	if err := s.repo.Delete(ctx, popup); err != nil {
		return fmt.Errorf("popup.Delete: %w", err)
	}

	if oldKey != "" && s.mediaStore != nil {
		_ = s.mediaStore.Delete(ctx, oldKey)
	}

	return nil
}

func (s *PopupService) RequestImageUpload(ctx context.Context, id int64, filename string) (*models.Popup, storage.PresignedUpload, error) {
	filename = normalizeTrim(filename)
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	validator.Required("filename", filename)
	if err := validator.Error(); err != nil {
		return nil, storage.PresignedUpload{}, err
	}

	if s.mediaStore == nil {
		return nil, storage.PresignedUpload{}, ErrStorageUnavailable
	}

	popup, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, storage.PresignedUpload{}, err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := popupImageContentType(ext)
	if contentType == "" {
		return nil, storage.PresignedUpload{}, NewValidationError(FieldError{Field: "filename", Reason: "must be a .png, .jpg, .jpeg, or .webp file"})
	}

	key := "popups/" + uuid.NewString() + ext
	upload, err := s.mediaStore.PresignUpload(ctx, key, contentType, maxPopupImageBytes)
	if err != nil {
		return nil, storage.PresignedUpload{}, fmt.Errorf("popup.RequestImageUpload presign: %w", err)
	}

	return popup, upload, nil
}

func (s *PopupService) FinalizeImageUpload(ctx context.Context, id int64, key, filename string) (*models.Popup, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	validator.Required("key", normalizeTrim(key))
	validator.Required("filename", normalizeTrim(filename))
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if s.mediaStore == nil {
		return nil, ErrStorageUnavailable
	}

	normalizedKey, err := normalizePopupImageKey(key)
	if err != nil {
		return nil, err
	}

	filename = normalizeTrim(filename)
	ext := strings.ToLower(filepath.Ext(filename))
	if popupImageContentType(ext) == "" {
		return nil, NewValidationError(FieldError{Field: "filename", Reason: "must be a .png, .jpg, .jpeg, or .webp file"})
	}

	popup, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldKey := ""
	if popup.ImageKey != nil {
		oldKey = strings.TrimSpace(*popup.ImageKey)
	}

	popup.ImageKey = &normalizedKey
	popup.ImageName = &filename
	popup.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, popup); err != nil {
		return nil, fmt.Errorf("popup.FinalizeImageUpload update: %w", err)
	}

	if oldKey != "" && oldKey != normalizedKey {
		_ = s.mediaStore.Delete(ctx, oldKey)
	}

	return popup, nil
}

func (s *PopupService) DeleteImage(ctx context.Context, id int64) (*models.Popup, error) {
	if s.mediaStore == nil {
		return nil, ErrStorageUnavailable
	}

	popup, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldKey := ""
	if popup.ImageKey != nil {
		oldKey = strings.TrimSpace(*popup.ImageKey)
	}

	popup.ImageKey = nil
	popup.ImageName = nil
	popup.IsActive = false
	popup.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, popup); err != nil {
		return nil, fmt.Errorf("popup.DeleteImage update: %w", err)
	}

	if oldKey != "" {
		_ = s.mediaStore.Delete(ctx, oldKey)
	}

	return popup, nil
}

func normalizePopupLinkURL(linkURL *string) (*string, error) {
	if linkURL == nil {
		return nil, nil
	}

	normalized := strings.TrimSpace(*linkURL)
	if normalized == "" {
		return nil, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidInput
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidInput
	}

	return &normalized, nil
}

func popupImageKey(popup *models.Popup) string {
	if popup == nil || popup.ImageKey == nil {
		return ""
	}

	return strings.TrimSpace(*popup.ImageKey)
}

func popupImageContentType(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func normalizePopupImageKey(key string) (string, error) {
	normalized := strings.TrimSpace(strings.TrimLeft(key, "/"))
	if normalized == "" || strings.Contains(normalized, "..") || !strings.HasPrefix(normalized, "popups/") {
		return "", NewValidationError(FieldError{Field: "key", Reason: "invalid"})
	}

	ext := strings.ToLower(filepath.Ext(normalized))
	if popupImageContentType(ext) == "" {
		return "", NewValidationError(FieldError{Field: "key", Reason: "invalid"})
	}

	return normalized, nil
}
