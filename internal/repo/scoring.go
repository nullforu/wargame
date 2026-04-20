package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type challengeScoreRow struct {
	ID     int64 `bun:"id"`
	Points int   `bun:"points"`
}

type challengeSolveCountRow struct {
	ChallengeID int64 `bun:"challenge_id"`
	SolveCount  int   `bun:"solve_count"`
}

func fixedPointsMap(ctx context.Context, db *bun.DB) (map[int64]int, error) {
	challenges, err := listChallengesForScoring(ctx, db)
	if err != nil {
		return nil, err
	}

	points := make(map[int64]int, len(challenges))
	for _, ch := range challenges {
		points[ch.ID] = ch.Points
	}

	return points, nil
}

func fixedPointsMapByIDs(ctx context.Context, db *bun.DB, challengeIDs []int64) (map[int64]int, error) {
	if len(challengeIDs) == 0 {
		return map[int64]int{}, nil
	}

	challenges, err := listChallengesForScoringByIDs(ctx, db, challengeIDs)
	if err != nil {
		return nil, err
	}

	points := make(map[int64]int, len(challenges))
	for _, ch := range challenges {
		points[ch.ID] = ch.Points
	}

	return points, nil
}

func listChallengesForScoring(ctx context.Context, db *bun.DB) ([]challengeScoreRow, error) {
	rows := make([]challengeScoreRow, 0)
	if err := db.NewSelect().
		TableExpr("challenges").
		ColumnExpr("id").
		ColumnExpr("points").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.listChallenges", err)
	}

	return rows, nil
}

func listChallengesForScoringByIDs(ctx context.Context, db *bun.DB, challengeIDs []int64) ([]challengeScoreRow, error) {
	rows := make([]challengeScoreRow, 0, len(challengeIDs))
	if err := db.NewSelect().
		TableExpr("challenges").
		ColumnExpr("id").
		ColumnExpr("points").
		Where("id IN (?)", bun.In(challengeIDs)).
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.listChallengesByIDs", err)
	}

	return rows, nil
}

func solveCountsByChallenge(ctx context.Context, db *bun.DB, challengeIDs []int64) (map[int64]int, error) {
	rows := make([]challengeSolveCountRow, 0)
	query := db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id").
		ColumnExpr("COUNT(*) AS solve_count").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole})).
		GroupExpr("challenge_id")
	if len(challengeIDs) > 0 {
		query = query.Where("s.challenge_id IN (?)", bun.In(challengeIDs))
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.solveCountsByChallenge", err)
	}

	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.ChallengeID] = row.SolveCount
	}

	return counts, nil
}

func challengeSolveCounts(ctx context.Context, db *bun.DB) (map[int64]int, error) {
	counts, err := solveCountsByChallenge(ctx, db, nil)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func challengeSolveCountsByIDs(ctx context.Context, db *bun.DB, challengeIDs []int64) (map[int64]int, error) {
	if len(challengeIDs) == 0 {
		return map[int64]int{}, nil
	}

	counts, err := solveCountsByChallenge(ctx, db, challengeIDs)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func decayFactor(ctx context.Context, db *bun.DB) (int, error) {
	var userCount int
	query := db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("COUNT(DISTINCT u.id)").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole}))

	if err := query.Scan(ctx, &userCount); err != nil {
		return 0, wrapError("score.userCount", err)
	}

	return userCount, nil
}
