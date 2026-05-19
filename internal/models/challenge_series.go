package models

import (
	"time"

	"github.com/uptrace/bun"
)

type ChallengeSeries struct {
	bun.BaseModel   `bun:"table:challenge_series"`
	ID              int64     `bun:"id,pk,autoincrement"`
	Title           string    `bun:"title,notnull"`
	Description     string    `bun:"description,notnull"`
	CreatedByUserID *int64    `bun:"created_by_user_id,nullzero"`
	CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt       time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type ChallengeSeriesChallenge struct {
	bun.BaseModel `bun:"table:challenge_series_challenges"`
	ID            int64     `bun:"id,pk,autoincrement"`
	SeriesID      int64     `bun:"series_id,notnull"`
	ChallengeID   int64     `bun:"challenge_id,notnull"`
	Position      int       `bun:"position,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type ChallengeSeriesDetailItem struct {
	SeriesID int64 `bun:"series_id"`
	Position int   `bun:"position"`
	Challenge
}
