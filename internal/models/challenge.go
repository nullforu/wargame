package models

import (
	"time"

	"wargame/internal/stack"

	"github.com/uptrace/bun"
)

// Database model for challenges
type Challenge struct {
	bun.BaseModel       `bun:"table:challenges"`
	ID                  int64                 `bun:"id,pk,autoincrement"`
	Title               string                `bun:"title,notnull"`
	Description         string                `bun:"description,notnull"`
	Points              int                   `bun:"points,notnull,default:0"`
	Category            string                `bun:"category,notnull"`
	FlagHash            string                `bun:"flag_hash,notnull"`
	CreatedByUserID     *int64                `bun:"created_by_user_id,nullzero"`
	CreatedByUsername   string                `bun:"created_by_username,scanonly"`
	PreviousChallengeID *int64                `bun:"previous_challenge_id,nullzero"`
	FileKey             *string               `bun:"file_key,nullzero"`
	FileName            *string               `bun:"file_name,nullzero"`
	FileUploadedAt      *time.Time            `bun:"file_uploaded_at,nullzero"`
	StackEnabled        bool                  `bun:"stack_enabled,notnull,default:false"`
	StackTargetPorts    stack.TargetPortSpecs `bun:"stack_target_ports,type:jsonb,nullzero"`
	StackPodSpec        *string               `bun:"stack_pod_spec,nullzero"`
	IsActive            bool                  `bun:"is_active,notnull"`
	CreatedAt           time.Time             `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	SolveCount          int                   `bun:"-"`
	Level               int                   `bun:"-"`
	LevelVotes          []LevelVoteCount      `bun:"-"`
}

const (
	UnknownLevel int = 0
	MinVoteLevel int = 1
	MaxVoteLevel int = 10
)

type LevelVoteCount struct {
	Level int `json:"level"`
	Count int `json:"count"`
}
