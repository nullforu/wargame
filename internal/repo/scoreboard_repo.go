package repo

import (
	"context"
	"sort"
	"time"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type ScoreboardRepo struct {
	db *bun.DB
}

func NewScoreboardRepo(db *bun.DB) *ScoreboardRepo {
	return &ScoreboardRepo{db: db}
}

type leaderboardChallengeRow struct {
	ID       int64  `bun:"id"`
	Title    string `bun:"title"`
	Category string `bun:"category"`
	Points   int    `bun:"points"`
}

func (r *ScoreboardRepo) leaderboardChallenges(ctx context.Context) ([]models.LeaderboardChallenge, error) {
	rows := make([]leaderboardChallengeRow, 0)
	if err := r.db.NewSelect().
		TableExpr("challenges AS c").
		ColumnExpr("c.id AS id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.category AS category").
		ColumnExpr("c.points AS points").
		OrderExpr("c.id ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("scoreboardRepo.leaderboardChallenges", err)
	}

	challenges := make([]models.LeaderboardChallenge, 0, len(rows))
	for _, row := range rows {
		challenges = append(challenges, models.LeaderboardChallenge{ID: row.ID, Title: row.Title, Category: row.Category, Points: row.Points})
	}

	return challenges, nil
}

func (r *ScoreboardRepo) Leaderboard(ctx context.Context, page, pageSize int) (models.LeaderboardResponse, int, error) {
	challenges, err := r.leaderboardChallenges(ctx)
	if err != nil {
		return models.LeaderboardResponse{}, 0, wrapError("scoreboardRepo.Leaderboard", err)
	}

	totalCount, err := r.db.NewSelect().
		TableExpr("users AS u").
		Where("u.role != ?", models.BlockedRole).
		Count(ctx)
	if err != nil {
		return models.LeaderboardResponse{}, 0, wrapError("scoreboardRepo.Leaderboard count", err)
	}

	rows := make([]models.LeaderboardEntry, 0)
	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		Where("u.role != ?", models.BlockedRole).
		GroupExpr("u.id, u.username, u.profile_image").
		OrderExpr("score DESC, u.id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx, &rows); err != nil {
		return models.LeaderboardResponse{}, 0, wrapError("scoreboardRepo.Leaderboard", err)
	}

	if len(rows) == 0 {
		return models.LeaderboardResponse{Challenges: challenges, Entries: []models.LeaderboardEntry{}}, totalCount, nil
	}

	userIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	type solveRow struct {
		UserID       int64     `bun:"user_id"`
		ChallengeID  int64     `bun:"challenge_id"`
		SolvedAt     time.Time `bun:"solved_at"`
		IsFirstBlood bool      `bun:"is_first_blood"`
	}
	solvedRows := make([]solveRow, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.user_id AS user_id").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		ColumnExpr("BOOL_OR(s.is_first_blood) AS is_first_blood").
		Where("s.correct = true").
		Where("s.user_id IN (?)", bun.In(userIDs)).
		GroupExpr("s.user_id, s.challenge_id").
		Scan(ctx, &solvedRows); err != nil {
		return models.LeaderboardResponse{}, 0, wrapError("scoreboardRepo.Leaderboard solves", err)
	}

	solvedByUser := make(map[int64][]models.LeaderboardSolve)
	for _, row := range solvedRows {
		solvedByUser[row.UserID] = append(solvedByUser[row.UserID], models.LeaderboardSolve{ChallengeID: row.ChallengeID, SolvedAt: row.SolvedAt, IsFirstBlood: row.IsFirstBlood})
	}

	for i := range rows {
		rows[i].Solves = solvedByUser[rows[i].UserID]
		if rows[i].Solves == nil {
			rows[i].Solves = []models.LeaderboardSolve{}
		}
		sort.Slice(rows[i].Solves, func(a, b int) bool {
			return rows[i].Solves[a].ChallengeID < rows[i].Solves[b].ChallengeID
		})
	}

	return models.LeaderboardResponse{Challenges: challenges, Entries: rows}, totalCount, nil
}

func (r *ScoreboardRepo) UserRanking(ctx context.Context, page, pageSize int) ([]models.UserRankingEntry, int, error) {
	totalCount, err := r.db.NewSelect().
		TableExpr("users AS u").
		Where("u.role != ?", models.BlockedRole).
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("scoreboardRepo.UserRanking count", err)
	}

	rows := make([]models.UserRankingEntry, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		With("solved", r.db.NewSelect().
			TableExpr("submissions AS s").
			ColumnExpr("s.user_id AS user_id").
			ColumnExpr("s.challenge_id AS challenge_id").
			Where("s.correct = true").
			GroupExpr("s.user_id, s.challenge_id")).
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("u.affiliation_id AS affiliation_id").
		ColumnExpr("a.name AS affiliation_name").
		ColumnExpr("u.bio AS bio").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		ColumnExpr("COALESCE(COUNT(s.challenge_id), 0) AS solved_count").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Join("LEFT JOIN solved AS s ON s.user_id = u.id").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		Where("u.role != ?", models.BlockedRole).
		GroupExpr("u.id, u.username, u.profile_image, u.affiliation_id, a.name, u.bio").
		OrderExpr("score DESC, solved_count DESC, u.id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("scoreboardRepo.UserRanking", err)
	}

	return rows, totalCount, nil
}

func (r *ScoreboardRepo) AffiliationRanking(ctx context.Context, page, pageSize int) ([]models.AffiliationRankingEntry, int, error) {
	var totalCount int
	if err := r.db.NewSelect().
		TableExpr("affiliations AS a").
		ColumnExpr("COUNT(DISTINCT a.id)").
		Join("JOIN users AS u ON u.affiliation_id = a.id AND u.role != ?", models.BlockedRole).
		Scan(ctx, &totalCount); err != nil {
		return nil, 0, wrapError("scoreboardRepo.AffiliationRanking count", err)
	}

	rows := make([]models.AffiliationRankingEntry, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		With("solved", r.db.NewSelect().
			TableExpr("submissions AS s").
			ColumnExpr("s.user_id AS user_id").
			ColumnExpr("s.challenge_id AS challenge_id").
			Where("s.correct = true").
			GroupExpr("s.user_id, s.challenge_id")).
		TableExpr("affiliations AS a").
		ColumnExpr("a.id AS affiliation_id").
		ColumnExpr("a.name AS name").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		ColumnExpr("COALESCE(COUNT(s.challenge_id), 0) AS solved_count").
		ColumnExpr("COALESCE(COUNT(DISTINCT u.id), 0) AS user_count").
		Join("JOIN users AS u ON u.affiliation_id = a.id AND u.role != ?", models.BlockedRole).
		Join("LEFT JOIN solved AS s ON s.user_id = u.id").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("a.id, a.name").
		OrderExpr("score DESC, solved_count DESC, LOWER(a.name) ASC, a.id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("scoreboardRepo.AffiliationRanking", err)
	}

	return rows, totalCount, nil
}

func (r *ScoreboardRepo) AffiliationUserRanking(ctx context.Context, affiliationID int64, page, pageSize int) ([]models.UserRankingEntry, int, error) {
	totalCount, err := r.db.NewSelect().
		TableExpr("users AS u").
		Where("u.role != ?", models.BlockedRole).
		Where("u.affiliation_id = ?", affiliationID).
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("scoreboardRepo.AffiliationUserRanking count", err)
	}

	rows := make([]models.UserRankingEntry, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		With("solved", r.db.NewSelect().
			TableExpr("submissions AS s").
			ColumnExpr("s.user_id AS user_id").
			ColumnExpr("s.challenge_id AS challenge_id").
			Where("s.correct = true").
			GroupExpr("s.user_id, s.challenge_id")).
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("u.affiliation_id AS affiliation_id").
		ColumnExpr("a.name AS affiliation_name").
		ColumnExpr("u.bio AS bio").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		ColumnExpr("COALESCE(COUNT(s.challenge_id), 0) AS solved_count").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Join("LEFT JOIN solved AS s ON s.user_id = u.id").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		Where("u.role != ?", models.BlockedRole).
		Where("u.affiliation_id = ?", affiliationID).
		GroupExpr("u.id, u.username, u.profile_image, u.affiliation_id, a.name, u.bio").
		OrderExpr("score DESC, solved_count DESC, u.id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("scoreboardRepo.AffiliationUserRanking", err)
	}

	return rows, totalCount, nil
}

func (r *ScoreboardRepo) TimelineSubmissions(ctx context.Context, since *time.Time) ([]models.UserTimelineRow, error) {
	pointsMap, err := fixedPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("scoreboardRepo.TimelineSubmissions", err)
	}

	rows := make([]models.UserTimelineRow, 0)
	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.role != ?", models.BlockedRole)

	query = applyTimelineWindow(query, since)
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("scoreboardRepo.TimelineSubmissions", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}

func applyTimelineWindow(query *bun.SelectQuery, since *time.Time) *bun.SelectQuery {
	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	return query.OrderExpr("s.submitted_at ASC, s.id ASC")
}
