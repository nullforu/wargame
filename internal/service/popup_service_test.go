package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/storage"
)

func TestPopupServiceCreateListUpdateDelete(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	if _, err := env.popupSvc.Create(context.Background(), " ", nil, true, 1); err == nil {
		t.Fatalf("expected validation error for blank title")
	}

	if _, err := env.popupSvc.Create(context.Background(), "Notice", nil, true, 0); err == nil {
		t.Fatalf("expected validation error for invalid creator")
	}

	if _, err := env.popupSvc.Create(context.Background(), "Notice", nil, true, admin.ID); err == nil {
		t.Fatalf("expected validation error for active popup without image")
	}

	badLink := "javascript:alert(1)"
	if _, err := env.popupSvc.Create(context.Background(), "Notice", &badLink, false, admin.ID); err == nil {
		t.Fatalf("expected validation error for invalid link")
	}

	link := "https://example.com/notice"
	first, err := env.popupSvc.Create(context.Background(), "First", &link, false, admin.ID)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}

	if first.LinkURL == nil || *first.LinkURL != link {
		t.Fatalf("expected link to be stored, got %+v", first.LinkURL)
	}

	second, err := env.popupSvc.Create(context.Background(), "Second", nil, false, admin.ID)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	rows, err := env.popupSvc.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(rows) != 2 || rows[0].Title != "Second" || rows[1].Title != "First" {
		t.Fatalf("unexpected list order: %+v", rows)
	}

	title := "Updated"
	active := true
	updated, err := env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{Title: &title, TitleSet: true, IsActive: &active})
	if err == nil {
		t.Fatalf("expected active update without image to fail")
	}

	updated, err = env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{Title: &title, TitleSet: true})
	if err != nil {
		t.Fatalf("update title: %v", err)
	}

	if updated.Title != "Updated" || updated.IsActive {
		t.Fatalf("unexpected update: %+v", updated)
	}

	if _, err := env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{TitleSet: true}); err == nil {
		t.Fatalf("expected validation error for null title")
	}

	updatedLink := "http://example.com/updated"
	updated, err = env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{LinkURL: &updatedLink, LinkSet: true})
	if err != nil {
		t.Fatalf("update link: %v", err)
	}

	if updated.LinkURL == nil || *updated.LinkURL != updatedLink {
		t.Fatalf("unexpected updated link: %+v", updated.LinkURL)
	}

	if _, err := env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{LinkURL: &badLink, LinkSet: true}); err == nil {
		t.Fatalf("expected validation error for invalid update link")
	}

	updated, err = env.popupSvc.Update(context.Background(), second.ID, PopupUpdate{LinkSet: true})
	if err != nil {
		t.Fatalf("clear link: %v", err)
	}

	if updated.LinkURL != nil {
		t.Fatalf("expected link to be cleared, got %+v", updated.LinkURL)
	}

	if err := env.popupSvc.Delete(context.Background(), first.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := env.popupSvc.GetByID(context.Background(), first.ID); !errors.Is(err, ErrPopupNotFound) {
		t.Fatalf("expected popup not found, got %v", err)
	}
}

func TestPopupServiceImageUploadAndActiveList(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	popup, err := env.popupSvc.Create(context.Background(), "Notice", nil, false, admin.ID)
	if err != nil {
		t.Fatalf("create popup: %v", err)
	}

	if _, _, err := env.popupSvc.RequestImageUpload(context.Background(), popup.ID, "notice.gif"); err == nil {
		t.Fatalf("expected invalid extension error")
	}

	_, upload, err := env.popupSvc.RequestImageUpload(context.Background(), popup.ID, "notice.webp")
	if err != nil {
		t.Fatalf("request upload: %v", err)
	}

	key := upload.Fields["key"]
	if !strings.HasPrefix(key, "popups/") || !strings.HasSuffix(key, ".webp") {
		t.Fatalf("unexpected upload key %q", key)
	}

	if upload.Fields["Content-Type"] != "image/webp" {
		t.Fatalf("unexpected content type fields: %+v", upload.Fields)
	}

	finalized, err := env.popupSvc.FinalizeImageUpload(context.Background(), popup.ID, key, "notice.webp")
	if err != nil {
		t.Fatalf("finalize upload: %v", err)
	}

	if finalized.ImageKey == nil || *finalized.ImageKey != key || finalized.ImageName == nil || *finalized.ImageName != "notice.webp" {
		t.Fatalf("unexpected finalized popup: %+v", finalized)
	}

	setActive := true
	finalized, err = env.popupSvc.Update(context.Background(), popup.ID, PopupUpdate{IsActive: &setActive})
	if err != nil {
		t.Fatalf("activate popup with image: %v", err)
	}

	active, err := env.popupSvc.ListActive(context.Background())
	if err != nil {
		t.Fatalf("list active: %v", err)
	}

	if len(active) != 1 || active[0].ID != popup.ID {
		t.Fatalf("expected active popup after image finalize, got %+v", active)
	}

	if _, err := env.popupSvc.FinalizeImageUpload(context.Background(), popup.ID, "../bad.png", "bad.png"); err == nil {
		t.Fatalf("expected invalid key error")
	}

	withoutImage, err := env.popupSvc.DeleteImage(context.Background(), popup.ID)
	if err != nil {
		t.Fatalf("delete image: %v", err)
	}

	if withoutImage.ImageKey != nil || withoutImage.ImageName != nil || withoutImage.IsActive {
		t.Fatalf("expected image fields to be cleared: %+v", withoutImage)
	}
}

func TestPopupServiceStorageUnavailable(t *testing.T) {
	env := setupServiceTest(t)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	popup, err := env.popupSvc.Create(context.Background(), "Notice", nil, false, admin.ID)
	if err != nil {
		t.Fatalf("create popup: %v", err)
	}

	svc := NewPopupService(repo.NewPopupRepo(env.db), nil)
	if _, _, err := svc.RequestImageUpload(context.Background(), popup.ID, "notice.png"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected storage unavailable for upload, got %v", err)
	}

	if _, err := svc.FinalizeImageUpload(context.Background(), popup.ID, "popups/x.png", "x.png"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected storage unavailable for finalize, got %v", err)
	}

	if _, err := svc.DeleteImage(context.Background(), popup.ID); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected storage unavailable for delete image, got %v", err)
	}

	errorSvc := NewPopupService(repo.NewPopupRepo(env.db), errorPopupMediaStore{})
	if _, _, err := errorSvc.RequestImageUpload(context.Background(), popup.ID, "notice.png"); err == nil || !strings.Contains(err.Error(), "presign") {
		t.Fatalf("expected wrapped presign error, got %v", err)
	}
}

type errorPopupMediaStore struct{}

func (errorPopupMediaStore) PresignUpload(context.Context, string, string, int64) (storage.PresignedUpload, error) {
	return storage.PresignedUpload{}, errors.New("presign failed")
}

func (errorPopupMediaStore) Delete(context.Context, string) error {
	return nil
}
