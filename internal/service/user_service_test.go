package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"wargame/internal/models"
)

func TestUserServiceGetByIDListUpdateProfile(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user", "pass", models.UserRole)

	got, err := env.userSvc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("unexpected user: %+v", got)
	}

	users, pagination, err := env.userSvc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("unexpected list: %+v", users)
	}

	if pagination.Page != 1 || pagination.PageSize != DefaultPageSize {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	newName := "newname"
	updated, err := env.userSvc.UpdateProfile(context.Background(), user.ID, &newName, nil, false, nil, false)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if updated.Username != newName {
		t.Fatalf("expected username %q, got %q", newName, updated.Username)
	}
}

func TestUserServiceBlockUnblock(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "block@example.com", "block", "pass", models.UserRole)
	admin := createUser(t, env, "admin@example.com", "admin", "pass", models.AdminRole)

	blocked, err := env.userSvc.BlockUser(context.Background(), user.ID, "policy")
	if err != nil {
		t.Fatalf("BlockUser: %v", err)
	}

	if blocked.Role != models.BlockedRole {
		t.Fatalf("expected blocked role, got %s", blocked.Role)
	}

	if _, err := env.userSvc.BlockUser(context.Background(), admin.ID, "policy"); err == nil {
		t.Fatalf("expected admin block error")
	}

	unblocked, err := env.userSvc.UnblockUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UnblockUser: %v", err)
	}

	if unblocked.Role != models.UserRole {
		t.Fatalf("expected user role after unblock, got %s", unblocked.Role)
	}
}

func TestUserServiceValidationAndNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, err := env.userSvc.GetByID(context.Background(), 0); err == nil {
		t.Fatalf("expected validation error")
	}

	if _, err := env.userSvc.GetByID(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := env.userSvc.BlockUser(context.Background(), 1, " "); err == nil {
		t.Fatalf("expected empty reason validation error")
	}

	if _, err := env.userSvc.UnblockUser(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserServiceSearchAndPagination(t *testing.T) {
	env := setupServiceTest(t)
	_ = createUser(t, env, "alpha@example.com", "alpha-user", "pass", models.UserRole)
	_ = createUser(t, env, "beta@example.com", "beta-user", "pass", models.UserRole)

	rows, pagination, err := env.userSvc.Search(context.Background(), "user", 1, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if pagination.TotalCount != 2 || pagination.TotalPages != 2 || !pagination.HasNext {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	if _, _, err := env.userSvc.Search(context.Background(), " ", 1, 10); err == nil {
		t.Fatalf("expected required query validation error")
	}

	if _, _, err := env.userSvc.ListPage(context.Background(), 1, MaxPageSize+1); err == nil {
		t.Fatalf("expected max page size validation error")
	}
}

func TestUserServiceUpdateProfileAffiliationAndListByAffiliation(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user", "pass", models.UserRole)
	affiliation := &models.Affiliation{Name: "Blue Team", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := env.affiliationRepo.Create(context.Background(), affiliation); err != nil {
		t.Fatalf("create affiliation: %v", err)
	}

	updated, err := env.userSvc.UpdateProfile(context.Background(), user.ID, nil, &affiliation.ID, true, nil, false)
	if err != nil {
		t.Fatalf("update profile affiliation: %v", err)
	}

	if updated.AffiliationID == nil || *updated.AffiliationID != affiliation.ID {
		t.Fatalf("unexpected affiliation id: %+v", updated.AffiliationID)
	}

	if updated.Affiliation == nil || *updated.Affiliation != affiliation.Name {
		t.Fatalf("unexpected affiliation name: %+v", updated.Affiliation)
	}

	rows, pagination, err := env.userSvc.ListByAffiliation(context.Background(), affiliation.ID, 1, 20)
	if err != nil {
		t.Fatalf("list by affiliation: %v", err)
	}

	if len(rows) != 1 || rows[0].ID != user.ID {
		t.Fatalf("unexpected rows: %+v", rows)
	}

	if pagination.TotalCount != 1 {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}

	cleared, err := env.userSvc.UpdateProfile(context.Background(), user.ID, nil, nil, true, nil, false)
	if err != nil {
		t.Fatalf("clear affiliation: %v", err)
	}

	if cleared.AffiliationID != nil {
		t.Fatalf("expected nil affiliation id, got %+v", cleared.AffiliationID)
	}

	badID := int64(99999)
	if _, err := env.userSvc.UpdateProfile(context.Background(), user.ID, nil, &badID, true, nil, false); err == nil {
		t.Fatalf("expected invalid affiliation id error")
	}
}

func TestUserServiceUpdateProfileBio(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "bio@example.com", "bio-user", "pass", models.UserRole)
	bio := "hello wargame"

	updated, err := env.userSvc.UpdateProfile(context.Background(), user.ID, nil, nil, false, &bio, true)
	if err != nil {
		t.Fatalf("set bio: %v", err)
	}

	if updated.Bio == nil || *updated.Bio != bio {
		t.Fatalf("unexpected bio after set: %+v", updated.Bio)
	}

	cleared, err := env.userSvc.UpdateProfile(context.Background(), user.ID, nil, nil, false, nil, true)
	if err != nil {
		t.Fatalf("clear bio: %v", err)
	}

	if cleared.Bio != nil {
		t.Fatalf("expected nil bio after clear, got %+v", cleared.Bio)
	}
}

func TestUserServiceUpdateProfileDuplicateUsername(t *testing.T) {
	env := setupServiceTest(t)
	user1 := createUser(t, env, "dup1@example.com", "dup-user-1", "pass", models.UserRole)
	_ = createUser(t, env, "dup2@example.com", "dup-user-2", "pass", models.UserRole)

	dup := "dup-user-2"
	if _, err := env.userSvc.UpdateProfile(context.Background(), user1.ID, &dup, nil, false, nil, false); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestUserServiceRequestProfileImageUpload(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "image@example.com", "image-user", "pass", models.UserRole)

	updated, upload, err := env.userSvc.RequestProfileImageUpload(context.Background(), user.ID, "avatar.png")
	if err != nil {
		t.Fatalf("RequestProfileImageUpload: %v", err)
	}

	if upload.Method != "POST" || upload.URL == "" {
		t.Fatalf("expected presigned POST upload, got %+v", upload)
	}

	if updated.ProfileImage == nil || !regexp.MustCompile(`^profiles/[0-9a-f-]+\.png$`).MatchString(*updated.ProfileImage) {
		t.Fatalf("unexpected profile image key: %+v", updated.ProfileImage)
	}

	prevKey := *updated.ProfileImage
	updated, _, err = env.userSvc.RequestProfileImageUpload(context.Background(), user.ID, "avatar.jpg")
	if err != nil {
		t.Fatalf("RequestProfileImageUpload overwrite: %v", err)
	}

	if updated.ProfileImage == nil || !regexp.MustCompile(`^profiles/[0-9a-f-]+\.jpg$`).MatchString(*updated.ProfileImage) {
		t.Fatalf("unexpected overwritten profile image key: %+v", updated.ProfileImage)
	}

	if *updated.ProfileImage == prevKey {
		t.Fatalf("expected new UUID key on each upload")
	}

	if _, _, err := env.userSvc.RequestProfileImageUpload(context.Background(), user.ID, "avatar.gif"); err == nil {
		t.Fatalf("expected extension validation error")
	}

	if _, _, err := env.userSvc.RequestProfileImageUpload(context.Background(), user.ID, " "); err == nil {
		t.Fatalf("expected required filename validation error")
	}

	if _, _, err := env.userSvc.RequestProfileImageUpload(context.Background(), user.ID, "avatar.jpeg"); err != nil {
		t.Fatalf("expected .jpeg allowed, got %v", err)
	}

	if _, _, err := NewUserService(env.userRepo, env.affiliationRepo, nil).RequestProfileImageUpload(context.Background(), user.ID, "avatar.png"); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}

	deleted, err := env.userSvc.DeleteProfileImage(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("DeleteProfileImage: %v", err)
	}

	if deleted.ProfileImage != nil {
		t.Fatalf("expected profile image to be nil, got %+v", deleted.ProfileImage)
	}
}
