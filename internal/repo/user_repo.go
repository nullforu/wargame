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
	if err := r.db.NewSelect().Model(user).Where("email = ?", email).Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmail", err)
	}

	return user, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	user := new(models.User)
	if err := r.db.NewSelect().Model(user).Where("email = ? OR username = ?", email, username).Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmailOrUsername", err)
	}

	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user := new(models.User)
	if err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx); err != nil {
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
	countQuery := r.db.NewSelect().Model((*models.User)(nil))
	listQuery := r.db.NewSelect().Model(&users)

	query = strings.TrimSpace(query)
	if query != "" {
		pattern := "%" + query + "%"
		countQuery = countQuery.Where("username ILIKE ?", pattern)
		listQuery = listQuery.Where("username ILIKE ?", pattern)
	}

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, wrapError("userRepo.listWithQuery count", err)
	}

	offset := (page - 1) * pageSize
	if err := listQuery.OrderExpr("id ASC").Limit(pageSize).Offset(offset).Scan(ctx); err != nil {
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
