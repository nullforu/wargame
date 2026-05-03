package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for challenge writeups.
type Writeup struct {
	bun.BaseModel `bun:"table:writeups"`
	ID            int64     `bun:"id,pk,autoincrement"`
	UserID        int64     `bun:"user_id,notnull"`
	ChallengeID   int64     `bun:"challenge_id,notnull"`
	Content       string    `bun:"content,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type WriteupDetail struct {
	ID                int64     `bun:"id"`
	UserID            int64     `bun:"user_id"`
	ChallengeID       int64     `bun:"challenge_id"`
	Content           string    `bun:"content"`
	CreatedAt         time.Time `bun:"created_at"`
	UpdatedAt         time.Time `bun:"updated_at"`
	Username          string    `bun:"username"`
	AffiliationID     *int64    `bun:"affiliation_id"`
	Affiliation       *string   `bun:"affiliation"`
	Bio               *string   `bun:"bio"`
	ProfileImage      *string   `bun:"profile_image"`
	ChallengeTitle    string    `bun:"challenge_title"`
	ChallengeCategory string    `bun:"challenge_category"`
	ChallengePoints   int       `bun:"challenge_points"`
	ChallengeLevel    int       `bun:"challenge_level"`
}
