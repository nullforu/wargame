package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"wargame/internal/auth"
	"wargame/internal/models"

	"github.com/redis/go-redis/v9"
)

func TestAuthServiceRegisterSuccess(t *testing.T) {
	env := setupServiceTest(t)

	user, err := env.authSvc.Register(context.Background(), "USER@Example.com", "  user1  ", "pass1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.ID == 0 || user.Email != "user@example.com" || user.Username != "user1" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestAuthServiceRegisterValidation(t *testing.T) {
	env := setupServiceTest(t)

	_, err := env.authSvc.Register(context.Background(), "bad", "", "")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceRegisterUserExists(t *testing.T) {
	env := setupServiceTest(t)
	_ = createUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	_, err := env.authSvc.Register(context.Background(), "user@example.com", "newuser", "pass")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthServiceLoginRefreshLogout(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	if _, _, _, err := env.authSvc.Login(context.Background(), "user@example.com", "wrong"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	access, refresh, got, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if access == "" || refresh == "" || got.ID != user.ID {
		t.Fatalf("unexpected login response")
	}

	claims, err := auth.ParseToken(env.cfg.JWT, refresh)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}

	val, err := env.redis.Get(context.Background(), refreshKey(claims.ID)).Result()
	if err != nil || val == "" {
		t.Fatalf("expected refresh token stored, err %v val %s", err, val)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	newAccess, newRefresh, err := env.authSvc.Refresh(context.Background(), refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if newAccess == "" || newRefresh == "" {
		t.Fatalf("expected new tokens")
	}

	if _, err := env.redis.Get(context.Background(), refreshKey(claims.ID)).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected old refresh revoked, got %v", err)
	}

	if err := env.authSvc.Logout(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	newClaims, err := auth.ParseToken(env.cfg.JWT, newRefresh)
	if err != nil {
		t.Fatalf("parse new refresh: %v", err)
	}

	if err := env.authSvc.Logout(context.Background(), newRefresh); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := env.redis.Get(context.Background(), refreshKey(newClaims.ID)).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected refresh revoked, got %v", err)
	}
}

func TestAuthServiceLoginBlocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "blocked@example.com", "blocked", "pass", models.UserRole)
	user.Role = models.BlockedRole

	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, _, err := env.authSvc.Login(context.Background(), "blocked@example.com", "pass"); err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
}

func TestAuthServiceRefreshBlocked(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	_, refresh, _, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user.Role = models.BlockedRole
	if err := env.userRepo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}
}

func TestAuthServiceRefreshUserNotFound(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "user@example.com", "user1", "pass", models.UserRole)

	_, refresh, _, err := env.authSvc.Login(context.Background(), "user@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := env.db.NewDelete().Model(&models.User{}).Where("id = ?", user.ID).Exec(context.Background()); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestAuthServiceRegisterWithoutKey(t *testing.T) {
	env := setupServiceTest(t)
	user, err := env.authSvc.Register(context.Background(), "user@example.com", "user1", "pass")
	if err != nil {
		t.Fatalf("expected register success without key, got %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("expected persisted user")
	}
}

func TestAuthServiceLoginUserNotFound(t *testing.T) {
	env := setupServiceTest(t)

	if _, _, _, err := env.authSvc.Login(context.Background(), "missing@example.com", "pass"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestAuthServiceParseRefreshTokenValidation(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "parse@example.com", "parse", "pass", models.UserRole)

	access, err := auth.GenerateAccessToken(env.cfg.JWT, user.ID, user.Role)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	if _, err := env.authSvc.parseRefreshToken(access); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for access token, got %v", err)
	}

	if _, err := env.authSvc.parseRefreshToken("not-a-token"); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for malformed token, got %v", err)
	}
}

func TestAuthServiceAssertRefreshValidCases(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "assert@example.com", "assert", "pass", models.UserRole)
	key := "custom-jti"

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds when key missing, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), "", time.Minute).Err(); err != nil {
		t.Fatalf("seed redis empty value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for empty stored value, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), strconv.FormatInt(user.ID+1, 10), time.Minute).Err(); err != nil {
		t.Fatalf("seed redis mismatched value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for mismatched user id, got %v", err)
	}

	if err := env.redis.Set(context.Background(), refreshKey(key), strconv.FormatInt(user.ID, 10), time.Minute).Err(); err != nil {
		t.Fatalf("seed redis valid value: %v", err)
	}

	if err := env.authSvc.assertRefreshValid(context.Background(), key, user.ID); err != nil {
		t.Fatalf("expected valid refresh state, got %v", err)
	}
}

func TestAuthServiceRefreshOldTokenRevoked(t *testing.T) {
	env := setupServiceTest(t)
	createUser(t, env, "refresh-old@example.com", "refresh-old", "pass", models.UserRole)

	_, refresh, _, err := env.authSvc.Login(context.Background(), "refresh-old@example.com", "pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	if _, _, err := env.authSvc.Refresh(context.Background(), refresh); !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("expected ErrInvalidCreds for revoked refresh token, got %v", err)
	}
}

func TestAuthServiceRegisterTrimsNormalization(t *testing.T) {
	env := setupServiceTest(t)

	user, err := env.authSvc.Register(context.Background(), "  MiXeD@Example.com  ", "  trim-user  ", "pass1")
	if err != nil {
		t.Fatalf("register with normalization: %v", err)
	}

	if user.Email != "mixed@example.com" || user.Username != "trim-user" {
		t.Fatalf("unexpected normalized user: %+v", user)
	}
}
