package repo

import (
	"context"
	"strings"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type UserRepo struct {
	db *bun.DB
}

func NewUserRepo(db *bun.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	if _, err := r.db.NewInsert().Model(user).Exec(ctx); err != nil {
		return wrapError("userRepo.Create", err)
	}

	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("a.name AS affiliation_name").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Where("u.email = ?", email).
		Scan(ctx, user); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmail", err)
	}

	return user, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	user := new(models.User)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("a.name AS affiliation_name").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Where("u.email = ? OR u.username = ?", email, username).
		Scan(ctx, user); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmailOrUsername", err)
	}

	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user := new(models.User)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("a.name AS affiliation_name").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Where("u.id = ?", id).
		Scan(ctx, user); err != nil {
		return nil, wrapNotFound("userRepo.GetByID", err)
	}

	return user, nil
}

func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]models.User, int, error) {
	return r.listWithQuery(ctx, "", page, pageSize)
}

func (r *UserRepo) Search(ctx context.Context, query string, page, pageSize int) ([]models.User, int, error) {
	return r.listWithQuery(ctx, query, page, pageSize)
}

func (r *UserRepo) listWithQuery(ctx context.Context, query string, page, pageSize int) ([]models.User, int, error) {
	users := make([]models.User, 0, pageSize)
	countQuery := r.db.NewSelect().TableExpr("users AS u")
	listQuery := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("a.name AS affiliation_name").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id")

	query = strings.TrimSpace(query)
	if query != "" {
		pattern := "%" + query + "%"
		countQuery = countQuery.Where("u.username ILIKE ?", pattern)
		listQuery = listQuery.Where("u.username ILIKE ?", pattern)
	}

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, wrapError("userRepo.listWithQuery count", err)
	}

	offset := (page - 1) * pageSize
	if err := listQuery.OrderExpr("u.id ASC").Limit(pageSize).Offset(offset).Scan(ctx, &users); err != nil {
		return nil, 0, wrapError("userRepo.listWithQuery list", err)
	}

	return users, totalCount, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	if _, err := r.db.NewUpdate().Model(user).WherePK().Exec(ctx); err != nil {
		return wrapError("userRepo.Update", err)
	}

	return nil
}

func (r *UserRepo) ListByAffiliation(ctx context.Context, affiliationID int64, page, pageSize int) ([]models.User, int, error) {
	rows := make([]models.User, 0, pageSize)
	base := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("a.name AS affiliation_name").
		Join("LEFT JOIN affiliations AS a ON a.id = u.affiliation_id").
		Where("u.affiliation_id = ?", affiliationID)
	countQuery := r.db.NewSelect().
		TableExpr("users AS u").
		Where("u.affiliation_id = ?", affiliationID)

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, wrapError("userRepo.ListByAffiliation count", err)
	}

	offset := (page - 1) * pageSize
	if err := base.OrderExpr("u.id ASC").Limit(pageSize).Offset(offset).Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("userRepo.ListByAffiliation list", err)
	}

	return rows, totalCount, nil
}
