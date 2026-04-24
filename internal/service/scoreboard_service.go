package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"wargame/internal/models"
	"wargame/internal/repo"
)

type ScoreboardService struct {
	repo *repo.ScoreboardRepo
}

func NewScoreboardService(scoreRepo *repo.ScoreboardRepo) *ScoreboardService {
	return &ScoreboardService{repo: scoreRepo}
}

func (s *ScoreboardService) Leaderboard(ctx context.Context, page, pageSize int) (models.LeaderboardResponse, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return models.LeaderboardResponse{}, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.Leaderboard(ctx, params.Page, params.PageSize)
	if err != nil {
		return models.LeaderboardResponse{}, models.Pagination{}, fmt.Errorf("scoreboard.Leaderboard: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *ScoreboardService) UserTimeline(ctx context.Context, since *time.Time) ([]models.TimelineSubmission, error) {
	raw, err := s.repo.TimelineSubmissions(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("scoreboard.UserTimeline: %w", err)
	}

	return aggregateUserTimeline(raw), nil
}

func (s *ScoreboardService) UserRanking(ctx context.Context, page, pageSize int) ([]models.UserRankingEntry, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.UserRanking(ctx, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("scoreboard.UserRanking: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *ScoreboardService) AffiliationRanking(ctx context.Context, page, pageSize int) ([]models.AffiliationRankingEntry, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.AffiliationRanking(ctx, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("scoreboard.AffiliationRanking: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *ScoreboardService) AffiliationUserRanking(ctx context.Context, affiliationID int64, page, pageSize int) ([]models.UserRankingEntry, models.Pagination, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", affiliationID)
	if err := validator.Error(); err != nil {
		return nil, models.Pagination{}, err
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.repo.AffiliationUserRanking(ctx, affiliationID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("scoreboard.AffiliationUserRanking: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func aggregateUserTimeline(raw []models.UserTimelineRow) []models.TimelineSubmission {
	if len(raw) == 0 {
		return []models.TimelineSubmission{}
	}

	type userKey struct {
		userID int64
		bucket time.Time
	}

	users := make(map[userKey]*models.TimelineSubmission)
	for _, sub := range raw {
		bucket := sub.SubmittedAt.Truncate(10 * time.Minute)
		key := userKey{userID: sub.UserID, bucket: bucket}

		if user, exists := users[key]; exists {
			user.Points += sub.Points
			user.ChallengeCount++
		} else {
			users[key] = &models.TimelineSubmission{Timestamp: bucket, UserID: sub.UserID, Username: sub.Username, Points: sub.Points, ChallengeCount: 1}
		}
	}

	result := make([]models.TimelineSubmission, 0, len(users))
	for _, user := range users {
		result = append(result, *user)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp.Equal(result[j].Timestamp) {
			return result[i].UserID < result[j].UserID
		}
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}
