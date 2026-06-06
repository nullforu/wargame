package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for site popups.
type Popup struct {
	bun.BaseModel   `bun:"table:popups"`
	ID              int64     `bun:"id,pk,autoincrement"`
	Title           string    `bun:"title,notnull"`
	ImageKey        *string   `bun:"image_key,nullzero"`
	ImageName       *string   `bun:"image_name,nullzero"`
	LinkURL         *string   `bun:"link_url,nullzero"`
	IsActive        bool      `bun:"is_active,notnull,default:false"`
	CreatedByUserID *int64    `bun:"created_by_user_id,nullzero"`
	CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt       time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
