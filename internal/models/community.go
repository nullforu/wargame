package models

import (
	"time"

	"github.com/uptrace/bun"
)

const (
	CommunityCategoryNotice  = 0
	CommunityCategoryFree    = 1
	CommunityCategoryQnA     = 2
	CommunityCategoryHumor   = 3
	PopularPostLikeThreshold = 5
)

// Database model for community posts.
type CommunityPost struct {
	bun.BaseModel `bun:"table:community_posts"`
	ID            int64     `bun:"id,pk,autoincrement"`
	UserID        int64     `bun:"user_id,notnull"`
	Category      int       `bun:"category,notnull"`
	Title         string    `bun:"title,notnull"`
	Content       string    `bun:"content,notnull"`
	ViewCount     int       `bun:"view_count,notnull,default:0"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type CommunityPostLike struct {
	bun.BaseModel `bun:"table:community_post_likes"`
	PostID        int64     `bun:"post_id,pk"`
	UserID        int64     `bun:"user_id,pk"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type CommunityPostDetail struct {
	ID            int64     `bun:"id"`
	UserID        int64     `bun:"user_id"`
	Category      int       `bun:"category"`
	Title         string    `bun:"title"`
	Content       string    `bun:"content"`
	ViewCount     int       `bun:"view_count"`
	LikeCount     int       `bun:"like_count"`
	CommentCount  int       `bun:"comment_count"`
	LikedByMe     bool      `bun:"liked_by_me"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	Username      string    `bun:"username"`
	AffiliationID *int64    `bun:"affiliation_id"`
	Affiliation   *string   `bun:"affiliation"`
	Bio           *string   `bun:"bio"`
	ProfileImage  *string   `bun:"profile_image"`
}

type CommunityPostLikeDetail struct {
	PostID        int64     `bun:"post_id"`
	UserID        int64     `bun:"user_id"`
	CreatedAt     time.Time `bun:"created_at"`
	Username      string    `bun:"username"`
	AffiliationID *int64    `bun:"affiliation_id"`
	Affiliation   *string   `bun:"affiliation"`
	Bio           *string   `bun:"bio"`
	ProfileImage  *string   `bun:"profile_image"`
}

type CommunityComment struct {
	bun.BaseModel `bun:"table:community_comments"`
	ID            int64     `bun:"id,pk,autoincrement"`
	PostID        int64     `bun:"post_id,notnull"`
	UserID        int64     `bun:"user_id,notnull"`
	Content       string    `bun:"content,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type CommunityCommentDetail struct {
	ID            int64     `bun:"id"`
	PostID        int64     `bun:"post_id"`
	UserID        int64     `bun:"user_id"`
	Content       string    `bun:"content"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	Username      string    `bun:"username"`
	AffiliationID *int64    `bun:"affiliation_id"`
	Affiliation   *string   `bun:"affiliation"`
	Bio           *string   `bun:"bio"`
	ProfileImage  *string   `bun:"profile_image"`
	PostTitle     string    `bun:"post_title"`
}
