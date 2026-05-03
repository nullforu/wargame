package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeCommentRepo struct {
	db *bun.DB
}

func NewChallengeCommentRepo(db *bun.DB) *ChallengeCommentRepo {
	return &ChallengeCommentRepo{db: db}
}

func (r *ChallengeCommentRepo) Create(ctx context.Context, comment *models.ChallengeCommentItem) error {
	if _, err := r.db.NewInsert().Model(comment).Exec(ctx); err != nil {
		return wrapError("challengeCommentRepo.Create", err)
	}

	return nil
}

func (r *ChallengeCommentRepo) Update(ctx context.Context, comment *models.ChallengeCommentItem) error {
	if _, err := r.db.NewUpdate().
		Model(comment).
		Column("content", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return wrapError("challengeCommentRepo.Update", err)
	}

	return nil
}

func (r *ChallengeCommentRepo) DeleteByID(ctx context.Context, id int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.ChallengeCommentItem)(nil)).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return wrapError("challengeCommentRepo.DeleteByID", err)
	}

	return nil
}

func (r *ChallengeCommentRepo) GetByID(ctx context.Context, id int64) (*models.ChallengeCommentItem, error) {
	row := new(models.ChallengeCommentItem)
	if err := r.db.NewSelect().Model(row).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		return nil, wrapNotFound("challengeCommentRepo.GetByID", err)
	}

	return row, nil
}

func (r *ChallengeCommentRepo) baseDetailQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("challenge_comments AS cc").
		ColumnExpr("cc.id").
		ColumnExpr("cc.user_id").
		ColumnExpr("cc.challenge_id").
		ColumnExpr("cc.content").
		ColumnExpr("cc.created_at").
		ColumnExpr("cc.updated_at").
		ColumnExpr("u.username").
		ColumnExpr("u.affiliation_id").
		ColumnExpr("aff.name AS affiliation").
		ColumnExpr("u.bio").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("c.title AS challenge_title").
		Join("JOIN users AS u ON u.id = cc.user_id").
		Join("JOIN challenges AS c ON c.id = cc.challenge_id").
		Join("LEFT JOIN affiliations AS aff ON aff.id = u.affiliation_id")
}

func (r *ChallengeCommentRepo) GetDetailByID(ctx context.Context, id int64) (*models.ChallengeCommentDetail, error) {
	row := new(models.ChallengeCommentDetail)
	if err := r.baseDetailQuery().Where("cc.id = ?", id).Limit(1).Scan(ctx, row); err != nil {
		return nil, wrapNotFound("challengeCommentRepo.GetDetailByID", err)
	}

	return row, nil
}

func (r *ChallengeCommentRepo) ChallengePage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.ChallengeCommentDetail, int, error) {
	rows := make([]models.ChallengeCommentDetail, 0, pageSize)
	base := r.baseDetailQuery().Where("cc.challenge_id = ?", challengeID)

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS comments", base).
		ColumnExpr("comments.id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("challengeCommentRepo.ChallengePage count", err)
	}

	if err := base.
		OrderExpr("cc.created_at DESC, cc.id DESC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("challengeCommentRepo.ChallengePage list", err)
	}

	return rows, totalCount, nil
}
