package models

import "time"

type LeaderboardEntry struct {
	UserID       int64              `bun:"user_id" json:"user_id"`
	Username     string             `bun:"username" json:"username"`
	ProfileImage *string            `bun:"profile_image" json:"profile_image"`
	Score        int                `bun:"score" json:"score"`
	Solves       []LeaderboardSolve `json:"solves"`
}

type LeaderboardChallenge struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Points   int    `json:"points"`
}

type LeaderboardSolve struct {
	ChallengeID  int64     `json:"challenge_id"`
	SolvedAt     time.Time `json:"solved_at"`
	IsFirstBlood bool      `json:"is_first_blood"`
}

type LeaderboardResponse struct {
	Challenges []LeaderboardChallenge `json:"challenges"`
	Entries    []LeaderboardEntry     `json:"entries"`
}

type UserRankingEntry struct {
	UserID        int64   `bun:"user_id" json:"user_id"`
	Username      string  `bun:"username" json:"username"`
	ProfileImage  *string `bun:"profile_image" json:"profile_image"`
	Score         int     `bun:"score" json:"score"`
	SolvedCount   int     `bun:"solved_count" json:"solved_count"`
	AffiliationID *int64  `bun:"affiliation_id" json:"affiliation_id"`
	Affiliation   *string `bun:"affiliation_name" json:"affiliation_name"`
	Bio           *string `bun:"bio" json:"bio"`
}

type AffiliationRankingEntry struct {
	AffiliationID int64  `bun:"affiliation_id" json:"affiliation_id"`
	Name          string `bun:"name" json:"name"`
	Score         int    `bun:"score" json:"score"`
	SolvedCount   int    `bun:"solved_count" json:"solved_count"`
	UserCount     int    `bun:"user_count" json:"user_count"`
}

type UserTimelineRow struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	UserID      int64     `bun:"user_id"`
	Username    string    `bun:"username"`
	ChallengeID int64     `bun:"challenge_id"`
	Points      int       `bun:"points"`
}

type TimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}
