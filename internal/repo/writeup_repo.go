package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type WriteupRepo struct {
	db *bun.DB
}

func NewWriteupRepo(db *bun.DB) *WriteupRepo {
	return &WriteupRepo{db: db}
}

func (r *WriteupRepo) Create(ctx context.Context, writeup *models.Writeup) error {
	if _, err := r.db.NewInsert().Model(writeup).Exec(ctx); err != nil {
		return wrapError("writeupRepo.Create", err)
	}

	return nil
}

func (r *WriteupRepo) Update(ctx context.Context, writeup *models.Writeup) error {
	if _, err := r.db.NewUpdate().
		Model(writeup).
		Column("content", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return wrapError("writeupRepo.Update", err)
	}

	return nil
}

func (r *WriteupRepo) DeleteByID(ctx context.Context, id int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.Writeup)(nil)).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return wrapError("writeupRepo.DeleteByID", err)
	}

	return nil
}

func (r *WriteupRepo) GetByID(ctx context.Context, id int64) (*models.Writeup, error) {
	row := new(models.Writeup)
	if err := r.db.NewSelect().
		Model(row).
		Where("id = ?", id).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("writeupRepo.GetByID", err)
	}

	return row, nil
}

func (r *WriteupRepo) GetByUserAndChallenge(ctx context.Context, userID, challengeID int64) (*models.Writeup, error) {
	row := new(models.Writeup)
	if err := r.db.NewSelect().
		Model(row).
		Where("user_id = ?", userID).
		Where("challenge_id = ?", challengeID).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("writeupRepo.GetByUserAndChallenge", err)
	}

	return row, nil
}

func (r *WriteupRepo) baseDetailQuery(includeContent bool) *bun.SelectQuery {
	query := r.db.NewSelect().
		TableExpr("writeups AS w").
		ColumnExpr("w.id").
		ColumnExpr("w.user_id").
		ColumnExpr("w.challenge_id").
		ColumnExpr("w.created_at").
		ColumnExpr("w.updated_at").
		ColumnExpr("u.username").
		ColumnExpr("u.affiliation_id").
		ColumnExpr("aff.name AS affiliation").
		ColumnExpr("u.bio").
		ColumnExpr("u.profile_image AS profile_image").
		ColumnExpr("c.title AS challenge_title").
		ColumnExpr("c.category AS challenge_category").
		ColumnExpr("c.points AS challenge_points").
		Join("JOIN users AS u ON u.id = w.user_id").
		Join("JOIN challenges AS c ON c.id = w.challenge_id").
		Join("LEFT JOIN affiliations AS aff ON aff.id = u.affiliation_id")

	if includeContent {
		query = query.ColumnExpr("w.content")
	}

	return query
}

func (r *WriteupRepo) GetDetailByID(ctx context.Context, id int64) (*models.WriteupDetail, error) {
	return r.getDetailByID(ctx, id, true)
}

func (r *WriteupRepo) GetDetailByIDWithoutContent(ctx context.Context, id int64) (*models.WriteupDetail, error) {
	return r.getDetailByID(ctx, id, false)
}

func (r *WriteupRepo) getDetailByID(ctx context.Context, id int64, includeContent bool) (*models.WriteupDetail, error) {
	row := new(models.WriteupDetail)
	if err := r.baseDetailQuery(includeContent).
		Where("w.id = ?", id).
		Limit(1).
		Scan(ctx, row); err != nil {
		return nil, wrapNotFound("writeupRepo.GetDetailByID", err)
	}

	return row, nil
}

func (r *WriteupRepo) ChallengePage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.WriteupDetail, int, error) {
	return r.challengePage(ctx, challengeID, page, pageSize, true)
}

func (r *WriteupRepo) ChallengePageWithoutContent(ctx context.Context, challengeID int64, page, pageSize int) ([]models.WriteupDetail, int, error) {
	return r.challengePage(ctx, challengeID, page, pageSize, false)
}

func (r *WriteupRepo) challengePage(ctx context.Context, challengeID int64, page, pageSize int, includeContent bool) ([]models.WriteupDetail, int, error) {
	rows := make([]models.WriteupDetail, 0, pageSize)

	base := r.baseDetailQuery(includeContent).Where("w.challenge_id = ?", challengeID)

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS writeups", base).
		ColumnExpr("writeups.id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("writeupRepo.ChallengePage count", err)
	}

	if err := base.
		OrderExpr("w.created_at DESC, w.id DESC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("writeupRepo.ChallengePage list", err)
	}

	return rows, totalCount, nil
}

func (r *WriteupRepo) UserPage(ctx context.Context, userID int64, page, pageSize int) ([]models.WriteupDetail, int, error) {
	rows := make([]models.WriteupDetail, 0, pageSize)

	base := r.baseDetailQuery(true).Where("w.user_id = ?", userID)

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS writeups", base).
		ColumnExpr("writeups.id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("writeupRepo.UserPage count", err)
	}

	if err := base.
		OrderExpr("w.updated_at DESC, w.id DESC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("writeupRepo.UserPage list", err)
	}

	return rows, totalCount, nil
}
