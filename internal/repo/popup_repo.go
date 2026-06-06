package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type PopupRepo struct {
	db *bun.DB
}

func NewPopupRepo(db *bun.DB) *PopupRepo {
	return &PopupRepo{db: db}
}

func (r *PopupRepo) Create(ctx context.Context, popup *models.Popup) error {
	if _, err := r.db.NewInsert().Model(popup).Exec(ctx); err != nil {
		return wrapError("popupRepo.Create", err)
	}

	return nil
}

func (r *PopupRepo) GetByID(ctx context.Context, id int64) (*models.Popup, error) {
	popup := new(models.Popup)
	if err := r.db.NewSelect().
		Model(popup).
		Where("id = ?", id).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("popupRepo.GetByID", err)
	}

	return popup, nil
}

func (r *PopupRepo) List(ctx context.Context) ([]models.Popup, error) {
	rows := make([]models.Popup, 0)
	if err := r.db.NewSelect().
		Model(&rows).
		OrderExpr("created_at DESC, id DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("popupRepo.List", err)
	}

	return rows, nil
}

func (r *PopupRepo) ListActiveWithImages(ctx context.Context) ([]models.Popup, error) {
	rows := make([]models.Popup, 0)
	if err := r.db.NewSelect().
		Model(&rows).
		Where("is_active = true").
		Where("image_key IS NOT NULL").
		OrderExpr("created_at DESC, id DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("popupRepo.ListActiveWithImages", err)
	}

	return rows, nil
}

func (r *PopupRepo) Update(ctx context.Context, popup *models.Popup) error {
	if _, err := r.db.NewUpdate().Model(popup).WherePK().Exec(ctx); err != nil {
		return wrapError("popupRepo.Update", err)
	}

	return nil
}

func (r *PopupRepo) Delete(ctx context.Context, popup *models.Popup) error {
	if _, err := r.db.NewDelete().Model(popup).WherePK().Exec(ctx); err != nil {
		return wrapError("popupRepo.Delete", err)
	}

	return nil
}
