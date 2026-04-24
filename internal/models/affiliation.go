package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for affiliations.
type Affiliation struct {
	bun.BaseModel `bun:"table:affiliations"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Name          string    `bun:"name,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
