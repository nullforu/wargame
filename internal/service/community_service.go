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

const (
	maxCommunityTitle   = 200
	maxCommunityContent = 100000
)

func isValidCommunityCategory(category int) bool {
	switch category {
	case models.CommunityCategoryNotice, models.CommunityCategoryFree, models.CommunityCategoryQnA, models.CommunityCategoryHumor:
		return true
	default:
		return false
	}
}

func (s *WargameService) CreateCommunityPost(ctx context.Context, userID int64, role string, category int, title, content string) (*models.CommunityPostDetail, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.Required("title", title)
	validator.Required("content", content)
	if !isValidCommunityCategory(category) {
		validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
	}

	if len(title) > maxCommunityTitle {
		validator.fields = append(validator.fields, FieldError{Field: "title", Reason: "too_long"})
	}

	if len(content) > maxCommunityContent {
		validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "too_long"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if category == models.CommunityCategoryNotice && role != models.AdminRole {
		return nil, ErrCommunityForbidden
	}

	now := time.Now().UTC()
	row := &models.CommunityPost{UserID: userID, Category: category, Title: title, Content: content, ViewCount: 0, CreatedAt: now, UpdatedAt: now}
	if err := s.communityRepo.Create(ctx, row); err != nil {
		return nil, fmt.Errorf("wargame.CreateCommunityPost create: %w", err)
	}

	detail, err := s.communityRepo.GetDetailByID(ctx, row.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("wargame.CreateCommunityPost detail: %w", err)
	}

	return detail, nil
}

func (s *WargameService) UpdateCommunityPost(ctx context.Context, userID int64, role string, postID int64, category *int, title *string, content *string) (*models.CommunityPostDetail, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("id", postID)
	if category == nil && title == nil && content == nil {
		validator.fields = append(validator.fields, FieldError{Field: "request", Reason: "empty"})
	}

	if category != nil && !isValidCommunityCategory(*category) {
		validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
	}

	if title != nil {
		n := strings.TrimSpace(*title)
		if n == "" {
			validator.fields = append(validator.fields, FieldError{Field: "title", Reason: "required"})
		}

		if len(n) > maxCommunityTitle {
			validator.fields = append(validator.fields, FieldError{Field: "title", Reason: "too_long"})
		}
	}

	if content != nil {
		n := strings.TrimSpace(*content)
		if n == "" {
			validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "required"})
		}

		if len(n) > maxCommunityContent {
			validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "too_long"})
		}
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	post, err := s.communityRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrCommunityPostNotFound
		}

		return nil, fmt.Errorf("wargame.UpdateCommunityPost lookup: %w", err)
	}

	targetCategory := post.Category
	if category != nil {
		targetCategory = *category
	}

	if targetCategory == models.CommunityCategoryNotice && role != models.AdminRole {
		return nil, ErrCommunityForbidden
	}

	if post.Category == models.CommunityCategoryNotice && role != models.AdminRole {
		return nil, ErrCommunityForbidden
	}

	if role != models.AdminRole && post.UserID != userID {
		return nil, ErrCommunityForbidden
	}

	if category != nil {
		post.Category = *category
	}

	if title != nil {
		post.Title = strings.TrimSpace(*title)
	}

	if content != nil {
		post.Content = strings.TrimSpace(*content)
	}

	post.UpdatedAt = time.Now().UTC()

	if err := s.communityRepo.Update(ctx, post); err != nil {
		return nil, fmt.Errorf("wargame.UpdateCommunityPost update: %w", err)
	}

	detail, err := s.communityRepo.GetDetailByID(ctx, postID, userID)
	if err != nil {
		return nil, fmt.Errorf("wargame.UpdateCommunityPost detail: %w", err)
	}

	return detail, nil
}

func (s *WargameService) DeleteCommunityPost(ctx context.Context, userID int64, role string, postID int64) error {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("id", postID)
	if err := validator.Error(); err != nil {
		return err
	}

	post, err := s.communityRepo.GetByID(ctx, postID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrCommunityPostNotFound
		}
		return fmt.Errorf("wargame.DeleteCommunityPost lookup: %w", err)
	}

	if post.Category == models.CommunityCategoryNotice && role != models.AdminRole {
		return ErrCommunityForbidden
	}

	if role != models.AdminRole && post.UserID != userID {
		return ErrCommunityForbidden
	}

	if err := s.communityRepo.DeleteByID(ctx, postID); err != nil {
		return fmt.Errorf("wargame.DeleteCommunityPost delete: %w", err)
	}

	return nil
}

func (s *WargameService) CommunityPostByID(ctx context.Context, postID int64, viewerID int64, increaseView bool) (*models.CommunityPostDetail, error) {
	if postID <= 0 {
		return nil, ErrInvalidInput
	}

	if increaseView {
		if err := s.communityRepo.IncrementViewCount(ctx, postID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, ErrCommunityPostNotFound
			}

			return nil, fmt.Errorf("wargame.CommunityPostByID increment view: %w", err)
		}
	}

	row, err := s.communityRepo.GetDetailByID(ctx, postID, viewerID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrCommunityPostNotFound
		}

		return nil, fmt.Errorf("wargame.CommunityPostByID detail: %w", err)
	}

	return row, nil
}

func (s *WargameService) CommunityPostsPage(ctx context.Context, page, pageSize int, query string, category *int, excludeNotice bool, popularOnly bool, sort string, viewerID int64) ([]models.CommunityPostDetail, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	sort = strings.TrimSpace(sort)
	if sort != "" {
		switch sort {
		case "latest", "oldest", "popular":
		default:
			return nil, models.Pagination{}, NewValidationError(FieldError{Field: "sort", Reason: "invalid"})
		}
	}

	if category != nil && !isValidCommunityCategory(*category) {
		return nil, models.Pagination{}, NewValidationError(FieldError{Field: "category", Reason: "invalid"})
	}

	rows, totalCount, err := s.communityRepo.Page(ctx, repo.CommunityListFilter{
		Query:         strings.TrimSpace(query),
		Category:      category,
		ExcludeNotice: excludeNotice,
		PopularOnly:   popularOnly,
		Sort:          sort,
	}, params.Page, params.PageSize, viewerID)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.CommunityPostsPage: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) ToggleCommunityPostLike(ctx context.Context, userID, postID int64) (bool, int, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("id", postID)
	if err := validator.Error(); err != nil {
		return false, 0, err
	}

	if _, err := s.communityRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return false, 0, ErrCommunityPostNotFound
		}

		return false, 0, fmt.Errorf("wargame.ToggleCommunityPostLike lookup: %w", err)
	}

	exists, err := s.communityRepo.HasLikeByPostAndUser(ctx, postID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("wargame.ToggleCommunityPostLike exists: %w", err)
	}

	liked := false
	if exists {
		if err := s.communityRepo.DeleteLike(ctx, postID, userID); err != nil {
			return false, 0, fmt.Errorf("wargame.ToggleCommunityPostLike delete: %w", err)
		}
	} else {
		if err := s.communityRepo.CreateLike(ctx, postID, userID); err != nil {
			return false, 0, fmt.Errorf("wargame.ToggleCommunityPostLike create: %w", err)
		}
		liked = true
	}

	likeCount, err := s.communityRepo.CountLikesByPostID(ctx, postID)
	if err != nil {
		return liked, 0, fmt.Errorf("wargame.ToggleCommunityPostLike count: %w", err)
	}

	return liked, likeCount, nil
}

func (s *WargameService) CommunityPostLikesPage(ctx context.Context, postID int64, page, pageSize int) ([]models.CommunityPostLikeDetail, models.Pagination, error) {
	if postID <= 0 {
		return nil, models.Pagination{}, ErrInvalidInput
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	if _, err := s.communityRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, models.Pagination{}, ErrCommunityPostNotFound
		}

		return nil, models.Pagination{}, fmt.Errorf("wargame.CommunityPostLikesPage lookup: %w", err)
	}

	rows, totalCount, err := s.communityRepo.LikesByPostPage(ctx, postID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.CommunityPostLikesPage: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}
