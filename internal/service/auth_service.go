package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisRefreshPrefix = "refresh:"
)

type AuthService struct {
	cfg      config.Config
	userRepo *repo.UserRepo
	redis    *redis.Client
}

func NewAuthService(cfg config.Config, userRepo *repo.UserRepo, redis *redis.Client) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, redis: redis}
}

func (s *AuthService) Register(ctx context.Context, email, username, password string) (*models.User, error) {
	email = normalizeEmail(email)
	username = normalizeTrim(username)
	validator := newFieldValidator()
	validator.Required("email", email)
	validator.Required("username", username)
	validator.Required("password", password)
	validator.Email("email", email)
	validator.MaxBytes("password", password, bcryptInputMaxBytes)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	_, err := s.userRepo.GetByEmailOrUsername(ctx, email, username)
	switch {
	case err == nil:
		return nil, ErrUserExists
	case !errors.Is(err, repo.ErrNotFound):
		return nil, fmt.Errorf("auth.Register lookup: %w", err)
	}

	hash, err := auth.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{Email: email, Username: username, PasswordHash: hash, Role: models.UserRole, CreatedAt: now, UpdatedAt: now}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("auth.Register create: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *models.User, error) {
	email = normalizeEmail(email)
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", "", nil, ErrInvalidCreds
		}

		return "", "", nil, fmt.Errorf("auth.Login lookup: %w", err)
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return "", "", nil, ErrInvalidCreds
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return "", "", nil, fmt.Errorf("auth.Login issueTokens: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

func refreshKey(jti string) string {
	return redisRefreshPrefix + jti
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	if err := s.assertRefreshValid(ctx, claims.ID, claims.UserID); err != nil {
		return "", "", ErrInvalidCreds
	}

	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return "", "", fmt.Errorf("auth.Refresh revoke: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", "", ErrInvalidCreds
		}

		return "", "", fmt.Errorf("auth.Refresh lookup: %w", err)
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return err
	}

	if err := s.redis.Del(ctx, refreshKey(claims.ID)).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("auth.Logout revoke: %w", err)
	}

	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (string, string, error) {
	jti := uuid.NewString()
	accessToken, err := auth.GenerateAccessToken(s.cfg.JWT, user.ID, user.Role)
	if err != nil {
		return "", "", fmt.Errorf("auth.issueTokens access: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(s.cfg.JWT, user.ID, user.Role, jti)
	if err != nil {
		return "", "", fmt.Errorf("auth.issueTokens refresh: %w", err)
	}

	if err := s.redis.Set(ctx, refreshKey(jti), strconv.FormatInt(user.ID, 10), s.cfg.JWT.RefreshTTL).Err(); err != nil {
		return "", "", fmt.Errorf("auth.issueTokens store: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) assertRefreshValid(ctx context.Context, jti string, userID int64) error {
	val, err := s.redis.Get(ctx, refreshKey(jti)).Result()
	if err == redis.Nil {
		return ErrInvalidCreds
	}
	if err != nil {
		return fmt.Errorf("auth.assertRefreshValid lookup: %w", err)
	}
	if val == "" || val != strconv.FormatInt(userID, 10) {
		return ErrInvalidCreds
	}

	return nil
}

func (s *AuthService) parseRefreshToken(refreshToken string) (*auth.Claims, error) {
	claims, err := auth.ParseToken(s.cfg.JWT, refreshToken)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if claims.Type != auth.TokenTypeRefresh || claims.ID == "" {
		return nil, ErrInvalidCreds
	}

	return claims, nil
}
