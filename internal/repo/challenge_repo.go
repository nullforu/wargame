package repo

import (
	"context"
	"strings"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeRepo struct {
	db *bun.DB
}

type ChallengeListFilter struct {
	Query          string
	Category       string
	Level          *int
	SolvedByUserID *int64
	Solved         *bool
}

func NewChallengeRepo(db *bun.DB) *ChallengeRepo {
	return &ChallengeRepo{db: db}
}

func (r *ChallengeRepo) ListActive(ctx context.Context, page, pageSize int) ([]models.Challenge, int, error) {
	return r.ListActiveFiltered(ctx, ChallengeListFilter{}, page, pageSize)
}

func (r *ChallengeRepo) SearchActive(ctx context.Context, query string, page, pageSize int) ([]models.Challenge, int, error) {
	return r.ListActiveFiltered(ctx, ChallengeListFilter{Query: query}, page, pageSize)
}

func (r *ChallengeRepo) ListActiveFiltered(ctx context.Context, filter ChallengeListFilter, page, pageSize int) ([]models.Challenge, int, error) {
	challenges := make([]models.Challenge, 0, pageSize)
	countQuery := r.db.NewSelect().Model((*models.Challenge)(nil))
	listQuery := r.db.NewSelect().Model(&challenges)

	query := strings.TrimSpace(filter.Query)
	if query != "" {
		pattern := "%" + query + "%"
		countQuery = countQuery.Where("title ILIKE ?", pattern)
		listQuery = listQuery.Where("title ILIKE ?", pattern)
	}

	if category := strings.TrimSpace(filter.Category); category != "" {
		countQuery = countQuery.Where("category = ?", category)
		listQuery = listQuery.Where("category = ?", category)
	}

	if filter.Level != nil {
		countQuery = countQuery.Where("level = ?", *filter.Level)
		listQuery = listQuery.Where("level = ?", *filter.Level)
	}

	if filter.Solved != nil && filter.SolvedByUserID != nil {
		if *filter.Solved {
			countQuery = countQuery.Where("EXISTS (SELECT 1 FROM submissions s WHERE s.correct = true AND s.challenge_id = challenge.id AND s.user_id = ?)", *filter.SolvedByUserID)
			listQuery = listQuery.Where("EXISTS (SELECT 1 FROM submissions s WHERE s.correct = true AND s.challenge_id = challenge.id AND s.user_id = ?)", *filter.SolvedByUserID)
		} else {
			countQuery = countQuery.Where("NOT EXISTS (SELECT 1 FROM submissions s WHERE s.correct = true AND s.challenge_id = challenge.id AND s.user_id = ?)", *filter.SolvedByUserID)
			listQuery = listQuery.Where("NOT EXISTS (SELECT 1 FROM submissions s WHERE s.correct = true AND s.challenge_id = challenge.id AND s.user_id = ?)", *filter.SolvedByUserID)
		}
	}

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, wrapError("challengeRepo.ListActiveFiltered count", err)
	}

	offset := (page - 1) * pageSize
	if err := listQuery.Order("id ASC").Limit(pageSize).Offset(offset).Scan(ctx); err != nil {
		return nil, 0, wrapError("challengeRepo.ListActiveFiltered list", err)
	}

	return challenges, totalCount, nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id int64) (*models.Challenge, error) {
	challenge := new(models.Challenge)
	if err := r.db.NewSelect().Model(challenge).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("challengeRepo.GetByID", err)
	}

	return challenge, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewInsert().Model(challenge).Exec(ctx); err != nil {
		return wrapError("challengeRepo.Create", err)
	}

	return nil
}

func (r *ChallengeRepo) Update(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewUpdate().Model(challenge).WherePK().Exec(ctx); err != nil {
		return wrapError("challengeRepo.Update", err)
	}

	return nil
}

func (r *ChallengeRepo) Delete(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewDelete().Model(challenge).WherePK().Exec(ctx); err != nil {
		return wrapError("challengeRepo.Delete", err)
	}

	return nil
}

func (r *ChallengeRepo) ChallengePoints(ctx context.Context) (map[int64]int, error) {
	points, err := fixedPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("challengeRepo.ChallengePoints", err)
	}

	return points, nil
}

func (r *ChallengeRepo) ChallengePointsByIDs(ctx context.Context, challengeIDs []int64) (map[int64]int, error) {
	points, err := fixedPointsMapByIDs(ctx, r.db, challengeIDs)
	if err != nil {
		return nil, wrapError("challengeRepo.ChallengePointsByIDs", err)
	}

	return points, nil
}

func (r *ChallengeRepo) SolveCounts(ctx context.Context) (map[int64]int, error) {
	counts, err := challengeSolveCounts(ctx, r.db)
	if err != nil {
		return nil, wrapError("challengeRepo.SolveCounts", err)
	}

	return counts, nil
}

func (r *ChallengeRepo) SolveCountsByIDs(ctx context.Context, challengeIDs []int64) (map[int64]int, error) {
	counts, err := challengeSolveCountsByIDs(ctx, r.db, challengeIDs)
	if err != nil {
		return nil, wrapError("challengeRepo.SolveCountsByIDs", err)
	}

	return counts, nil
}
