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

func (s *ScoreboardService) Leaderboard(ctx context.Context) (models.LeaderboardResponse, error) {
	rows, err := s.repo.Leaderboard(ctx)
	if err != nil {
		return models.LeaderboardResponse{}, fmt.Errorf("scoreboard.Leaderboard: %w", err)
	}

	return rows, nil
}

func (s *ScoreboardService) UserTimeline(ctx context.Context, since *time.Time) ([]models.TimelineSubmission, error) {
	raw, err := s.repo.TimelineSubmissions(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("scoreboard.UserTimeline: %w", err)
	}

	return aggregateUserTimeline(raw), nil
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
