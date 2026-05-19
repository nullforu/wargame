package repo

import (
	"context"
	"strings"
	"time"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeSeriesRepo struct {
	db *bun.DB
}

func NewChallengeSeriesRepo(db *bun.DB) *ChallengeSeriesRepo {
	return &ChallengeSeriesRepo{db: db}
}

func (r *ChallengeSeriesRepo) Create(ctx context.Context, series *models.ChallengeSeries) error {
	if _, err := r.db.NewInsert().Model(series).Exec(ctx); err != nil {
		return wrapError("challengeSeriesRepo.Create", err)
	}

	return nil
}

func (r *ChallengeSeriesRepo) Update(ctx context.Context, series *models.ChallengeSeries) error {
	if _, err := r.db.NewUpdate().Model(series).WherePK().Exec(ctx); err != nil {
		return wrapError("challengeSeriesRepo.Update", err)
	}

	return nil
}

func (r *ChallengeSeriesRepo) DeleteByID(ctx context.Context, id int64) error {
	if _, err := r.db.NewDelete().Model((*models.ChallengeSeries)(nil)).Where("id = ?", id).Exec(ctx); err != nil {
		return wrapError("challengeSeriesRepo.DeleteByID", err)
	}

	return nil
}

func (r *ChallengeSeriesRepo) GetByID(ctx context.Context, id int64) (*models.ChallengeSeries, error) {
	row := new(models.ChallengeSeries)
	if err := r.db.NewSelect().Model(row).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		return nil, wrapNotFound("challengeSeriesRepo.GetByID", err)
	}

	return row, nil
}

func (r *ChallengeSeriesRepo) List(ctx context.Context, page, pageSize int, sort string) ([]models.ChallengeSeries, int, error) {
	rows := make([]models.ChallengeSeries, 0, pageSize)
	base := r.db.NewSelect().Model((*models.ChallengeSeries)(nil))

	totalCount, err := base.Count(ctx)
	if err != nil {
		return nil, 0, wrapError("challengeSeriesRepo.List count", err)
	}

	order := "id DESC"
	if strings.TrimSpace(sort) == "oldest" {
		order = "id ASC"
	}

	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().Model(&rows).
		Order(order).
		Limit(pageSize).
		Offset(offset).
		Scan(ctx); err != nil {
		return nil, 0, wrapError("challengeSeriesRepo.List list", err)
	}

	return rows, totalCount, nil
}

func (r *ChallengeSeriesRepo) ReplaceChallenges(ctx context.Context, seriesID int64, challengeIDs []int64) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*models.ChallengeSeriesChallenge)(nil)).Where("series_id = ?", seriesID).Exec(ctx); err != nil {
			return wrapError("challengeSeriesRepo.ReplaceChallenges delete", err)
		}

		if len(challengeIDs) == 0 {
			return nil
		}

		now := time.Now().UTC()
		rows := make([]models.ChallengeSeriesChallenge, 0, len(challengeIDs))
		for i, challengeID := range challengeIDs {
			rows = append(rows, models.ChallengeSeriesChallenge{SeriesID: seriesID, ChallengeID: challengeID, Position: i + 1, CreatedAt: now})
		}

		if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
			return wrapError("challengeSeriesRepo.ReplaceChallenges insert", err)
		}

		return nil
	})
}

func (r *ChallengeSeriesRepo) DetailChallenges(ctx context.Context, seriesID int64) ([]models.ChallengeSeriesDetailItem, error) {
	rows := make([]models.ChallengeSeriesDetailItem, 0)
	if err := r.db.NewSelect().
		TableExpr("challenge_series_challenges AS csc").
		Join("JOIN challenges AS challenge ON challenge.id = csc.challenge_id").
		Join("LEFT JOIN users AS author ON author.id = challenge.created_by_user_id").
		Join("LEFT JOIN affiliations AS author_aff ON author_aff.id = author.affiliation_id").
		ColumnExpr("csc.series_id").
		ColumnExpr("csc.position").
		ColumnExpr("challenge.*").
		ColumnExpr("author.username AS created_by_username").
		ColumnExpr("author.affiliation_id AS created_by_affiliation_id").
		ColumnExpr("author_aff.name AS created_by_affiliation").
		ColumnExpr("author.bio AS created_by_bio").
		ColumnExpr("author.profile_image AS created_by_profile_image").
		Where("csc.series_id = ?", seriesID).
		OrderExpr("csc.position ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("challengeSeriesRepo.DetailChallenges", err)
	}

	return rows, nil
}

func (r *ChallengeSeriesRepo) ExistsByTitle(ctx context.Context, title string, excludeID *int64) (bool, error) {
	title = strings.TrimSpace(title)
	q := r.db.NewSelect().Model((*models.ChallengeSeries)(nil)).Where("title = ?", title)
	if excludeID != nil {
		q = q.Where("id != ?", *excludeID)
	}

	n, err := q.Count(ctx)
	if err != nil {
		return false, wrapError("challengeSeriesRepo.ExistsByTitle", err)
	}

	return n > 0, nil
}
