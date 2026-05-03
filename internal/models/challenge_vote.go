package models

import (
	"time"

	"github.com/uptrace/bun"
)

type ChallengeVote struct {
	bun.BaseModel `bun:"table:challenge_votes"`
	ID            int64     `bun:"id,pk,autoincrement"`
	ChallengeID   int64     `bun:"challenge_id,notnull"`
	UserID        int64     `bun:"user_id,notnull"`
	Level         int       `bun:"level,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type ChallengeVoteDetail struct {
	UserID       int64     `bun:"user_id" json:"user_id"`
	Username     string    `bun:"username" json:"username"`
	ProfileImage *string   `bun:"profile_image" json:"profile_image"`
	Level        int       `bun:"level" json:"level"`
	UpdatedAt    time.Time `bun:"updated_at" json:"updated_at"`
}
