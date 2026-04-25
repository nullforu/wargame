package repo

import (
	"context"
	"strings"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type AffiliationRepo struct {
	db *bun.DB
}

func NewAffiliationRepo(db *bun.DB) *AffiliationRepo {
	return &AffiliationRepo{db: db}
}

func (r *AffiliationRepo) Create(ctx context.Context, affiliation *models.Affiliation) error {
	affiliation.Name = strings.TrimSpace(affiliation.Name)
	if _, err := r.db.NewInsert().Model(affiliation).Exec(ctx); err != nil {
		return wrapError("affiliationRepo.Create", err)
	}

	return nil
}

func (r *AffiliationRepo) ExistsByID(ctx context.Context, id int64) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("affiliations AS a").
		Where("a.id = ?", id).
		Count(ctx)
	if err != nil {
		return false, wrapError("affiliationRepo.ExistsByID", err)
	}

	return count > 0, nil
}

func (r *AffiliationRepo) ExistsByNameCI(ctx context.Context, name string) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("affiliations AS a").
		Where("LOWER(a.name) = LOWER(?)", strings.TrimSpace(name)).
		Count(ctx)
	if err != nil {
		return false, wrapError("affiliationRepo.ExistsByNameCI", err)
	}

	return count > 0, nil
}

func (r *AffiliationRepo) GetByID(ctx context.Context, id int64) (*models.Affiliation, error) {
	affiliation := new(models.Affiliation)
	if err := r.db.NewSelect().
		Model(affiliation).
		Where("id = ?", id).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("affiliationRepo.GetByID", err)
	}

	return affiliation, nil
}

func (r *AffiliationRepo) List(ctx context.Context, page, pageSize int) ([]models.Affiliation, int, error) {
	rows := make([]models.Affiliation, 0, pageSize)

	totalCount, err := r.db.NewSelect().Model((*models.Affiliation)(nil)).Count(ctx)
	if err != nil {
		return nil, 0, wrapError("affiliationRepo.List count", err)
	}

	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		Model(&rows).
		OrderExpr("LOWER(name) ASC, id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx); err != nil {
		return nil, 0, wrapError("affiliationRepo.List list", err)
	}

	return rows, totalCount, nil
}

func (r *AffiliationRepo) Search(ctx context.Context, query string, page, pageSize int) ([]models.Affiliation, int, error) {
	rows := make([]models.Affiliation, 0, pageSize)
	query = strings.TrimSpace(query)

	totalCount, err := r.db.NewSelect().
		Model((*models.Affiliation)(nil)).
		Where("name ILIKE ?", "%"+query+"%").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("affiliationRepo.Search count", err)
	}

	offset := (page - 1) * pageSize
	if err := r.db.NewSelect().
		Model(&rows).
		Where("name ILIKE ?", "%"+query+"%").
		OrderExpr("LOWER(name) ASC, id ASC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx); err != nil {
		return nil, 0, wrapError("affiliationRepo.Search list", err)
	}

	return rows, totalCount, nil
}
