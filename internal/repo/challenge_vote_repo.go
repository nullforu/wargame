package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeVoteRepo struct {
	db *bun.DB
}

func NewChallengeVoteRepo(db *bun.DB) *ChallengeVoteRepo {
	return &ChallengeVoteRepo{db: db}
}

func (r *ChallengeVoteRepo) Upsert(ctx context.Context, vote *models.ChallengeVote) error {
	if _, err := r.db.NewInsert().
		Model(vote).
		On("CONFLICT (user_id, challenge_id) DO UPDATE").
		Set("level = EXCLUDED.level").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx); err != nil {
		return wrapError("challengeVoteRepo.Upsert", err)
	}

	return nil
}

func (r *ChallengeVoteRepo) VoteCountsByChallengeID(ctx context.Context, challengeID int64) ([]models.LevelVoteCount, error) {
	rows := make([]models.LevelVoteCount, 0)
	if err := r.db.NewSelect().
		TableExpr("challenge_votes AS cv").
		ColumnExpr("cv.level AS level").
		ColumnExpr("COUNT(*) AS count").
		Where("cv.challenge_id = ?", challengeID).
		GroupExpr("cv.level").
		OrderExpr("cv.level ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("challengeVoteRepo.VoteCountsByChallengeID", err)
	}

	return rows, nil
}

func (r *ChallengeVoteRepo) RepresentativeLevelByChallengeID(ctx context.Context, challengeID int64) (int, error) {
	type row struct {
		Level int `bun:"level"`
	}
	rows := make([]row, 0, 1)

	ordered := r.db.NewSelect().
		TableExpr("challenge_votes AS cv").
		ColumnExpr("cv.level AS level").
		ColumnExpr("COUNT(*) AS vote_count").
		ColumnExpr("MAX(cv.updated_at) AS latest_vote_at").
		Where("cv.challenge_id = ?", challengeID).
		GroupExpr("cv.level").
		OrderExpr("vote_count DESC, latest_vote_at DESC, level DESC").
		Limit(1)

	if err := r.db.NewSelect().
		TableExpr("(?) AS ranked", ordered).
		ColumnExpr("ranked.level AS level").
		Scan(ctx, &rows); err != nil {
		return models.UnknownLevel, wrapError("challengeVoteRepo.RepresentativeLevelByChallengeID", err)
	}

	if len(rows) == 0 {
		return models.UnknownLevel, nil
	}

	return rows[0].Level, nil
}

func (r *ChallengeVoteRepo) RepresentativeLevelsByChallengeIDs(ctx context.Context, challengeIDs []int64) (map[int64]int, error) {
	if len(challengeIDs) == 0 {
		return map[int64]int{}, nil
	}

	type row struct {
		ChallengeID int64 `bun:"challenge_id"`
		Level       int   `bun:"level"`
	}
	rows := make([]row, 0)
	if err := r.db.NewSelect().
		With("ranked", r.db.NewSelect().
			TableExpr("challenge_votes AS cv").
			ColumnExpr("cv.challenge_id").
			ColumnExpr("cv.level").
			ColumnExpr("COUNT(*) AS vote_count").
			ColumnExpr("MAX(cv.updated_at) AS latest_vote_at").
			ColumnExpr("ROW_NUMBER() OVER (PARTITION BY cv.challenge_id ORDER BY COUNT(*) DESC, MAX(cv.updated_at) DESC, cv.level DESC) AS rn").
			Where("cv.challenge_id IN (?)", bun.In(challengeIDs)).
			GroupExpr("cv.challenge_id, cv.level"),
		).
		TableExpr("ranked").
		ColumnExpr("challenge_id, level").
		Where("rn = 1").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("challengeVoteRepo.RepresentativeLevelsByChallengeIDs", err)
	}

	result := make(map[int64]int, len(challengeIDs))
	for _, id := range challengeIDs {
		result[id] = models.UnknownLevel
	}

	for _, item := range rows {
		result[item.ChallengeID] = item.Level
	}

	return result, nil
}

func (r *ChallengeVoteRepo) VoteCountsByChallengeIDs(ctx context.Context, challengeIDs []int64) (map[int64][]models.LevelVoteCount, error) {
	if len(challengeIDs) == 0 {
		return map[int64][]models.LevelVoteCount{}, nil
	}

	type row struct {
		ChallengeID int64 `bun:"challenge_id"`
		Level       int   `bun:"level"`
		Count       int   `bun:"count"`
	}
	rows := make([]row, 0)
	if err := r.db.NewSelect().
		TableExpr("challenge_votes AS cv").
		ColumnExpr("cv.challenge_id").
		ColumnExpr("cv.level").
		ColumnExpr("COUNT(*) AS count").
		Where("cv.challenge_id IN (?)", bun.In(challengeIDs)).
		GroupExpr("cv.challenge_id, cv.level").
		OrderExpr("cv.challenge_id ASC, cv.level ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("challengeVoteRepo.VoteCountsByChallengeIDs", err)
	}

	result := make(map[int64][]models.LevelVoteCount, len(challengeIDs))
	for _, id := range challengeIDs {
		result[id] = []models.LevelVoteCount{}
	}

	for _, item := range rows {
		result[item.ChallengeID] = append(result[item.ChallengeID], models.LevelVoteCount{
			Level: item.Level,
			Count: item.Count,
		})
	}

	return result, nil
}

func (r *ChallengeVoteRepo) VoteLevelByUserAndChallengeID(ctx context.Context, userID, challengeID int64) (*int, error) {
	type row struct {
		Level int `bun:"level"`
	}
	rows := make([]row, 0, 1)
	if err := r.db.NewSelect().
		TableExpr("challenge_votes AS cv").
		ColumnExpr("cv.level AS level").
		Where("cv.user_id = ?", userID).
		Where("cv.challenge_id = ?", challengeID).
		Limit(1).
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("challengeVoteRepo.VoteLevelByUserAndChallengeID", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	level := rows[0].Level
	return &level, nil
}

func (r *ChallengeVoteRepo) VotesByChallengePage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.ChallengeVoteDetail, int, error) {
	rows := make([]models.ChallengeVoteDetail, 0, pageSize)
	base := r.db.NewSelect().
		TableExpr("challenge_votes AS cv").
		ColumnExpr("cv.user_id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("cv.level AS level").
		ColumnExpr("cv.updated_at AS updated_at").
		Join("JOIN users AS u ON u.id = cv.user_id").
		Where("cv.challenge_id = ?", challengeID)

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS votes", base).
		ColumnExpr("votes.user_id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("challengeVoteRepo.VotesByChallengePage count", err)
	}

	if err := base.
		OrderExpr("cv.updated_at DESC, cv.user_id ASC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("challengeVoteRepo.VotesByChallengePage list", err)
	}

	return rows, totalCount, nil
}
