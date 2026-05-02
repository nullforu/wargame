package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type CommunityListFilter struct {
	Query         string
	Category      *int
	ExcludeNotice bool
	PopularOnly   bool
	Sort          string
}

type CommunityRepo struct {
	db *bun.DB
}

func NewCommunityRepo(db *bun.DB) *CommunityRepo {
	return &CommunityRepo{db: db}
}

func (r *CommunityRepo) Create(ctx context.Context, post *models.CommunityPost) error {
	if _, err := r.db.NewInsert().Model(post).Exec(ctx); err != nil {
		return wrapError("communityRepo.Create", err)
	}

	return nil
}

func (r *CommunityRepo) Update(ctx context.Context, post *models.CommunityPost) error {
	if _, err := r.db.NewUpdate().
		Model(post).
		Column("category", "title", "content", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return wrapError("communityRepo.Update", err)
	}

	return nil
}

func (r *CommunityRepo) DeleteByID(ctx context.Context, id int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.CommunityPost)(nil)).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return wrapError("communityRepo.DeleteByID", err)
	}

	return nil
}

func (r *CommunityRepo) GetByID(ctx context.Context, id int64) (*models.CommunityPost, error) {
	row := new(models.CommunityPost)
	if err := r.db.NewSelect().Model(row).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		return nil, wrapNotFound("communityRepo.GetByID", err)
	}

	return row, nil
}

func (r *CommunityRepo) IncrementViewCount(ctx context.Context, id int64) error {
	if _, err := r.db.NewUpdate().
		Model((*models.CommunityPost)(nil)).
		Set("view_count = view_count + 1").
		Set("updated_at = updated_at").
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return wrapError("communityRepo.IncrementViewCount", err)
	}

	return nil
}

func (r *CommunityRepo) baseDetailQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("community_posts AS cp").
		ColumnExpr("cp.id").
		ColumnExpr("cp.user_id").
		ColumnExpr("cp.category").
		ColumnExpr("cp.title").
		ColumnExpr("cp.content").
		ColumnExpr("cp.view_count").
		ColumnExpr("(SELECT COUNT(*) FROM community_post_likes AS cpl WHERE cpl.post_id = cp.id) AS like_count").
		ColumnExpr("(SELECT COUNT(*) FROM community_comments AS cpc WHERE cpc.post_id = cp.id) AS comment_count").
		ColumnExpr("cp.created_at").
		ColumnExpr("cp.updated_at").
		ColumnExpr("u.username").
		ColumnExpr("u.affiliation_id").
		ColumnExpr("aff.name AS affiliation").
		ColumnExpr("u.bio").
		Join("JOIN users AS u ON u.id = cp.user_id").
		Join("LEFT JOIN affiliations AS aff ON aff.id = u.affiliation_id")
}

func withLikedByMe(base *bun.SelectQuery, viewerID int64) *bun.SelectQuery {
	if viewerID <= 0 {
		return base.ColumnExpr("FALSE AS liked_by_me")
	}

	return base.ColumnExpr("EXISTS (SELECT 1 FROM community_post_likes AS cplm WHERE cplm.post_id = cp.id AND cplm.user_id = ?) AS liked_by_me", viewerID)
}

func (r *CommunityRepo) GetDetailByID(ctx context.Context, id int64, viewerID int64) (*models.CommunityPostDetail, error) {
	row := new(models.CommunityPostDetail)
	if err := withLikedByMe(r.baseDetailQuery(), viewerID).Where("cp.id = ?", id).Limit(1).Scan(ctx, row); err != nil {
		return nil, wrapNotFound("communityRepo.GetDetailByID", err)
	}

	return row, nil
}

func (r *CommunityRepo) Page(ctx context.Context, filter CommunityListFilter, page, pageSize int, viewerID int64) ([]models.CommunityPostDetail, int, error) {
	rows := make([]models.CommunityPostDetail, 0, pageSize)
	base := withLikedByMe(r.baseDetailQuery(), viewerID)

	if filter.Category != nil {
		base = base.Where("cp.category = ?", *filter.Category)
	}

	if filter.ExcludeNotice {
		base = base.Where("cp.category <> ?", models.CommunityCategoryNotice)
	}

	if filter.PopularOnly {
		base = base.Where("(SELECT COUNT(*) FROM community_post_likes AS cpl_pop WHERE cpl_pop.post_id = cp.id) >= ?", models.PopularPostLikeThreshold)
	}

	if query := strings.TrimSpace(filter.Query); query != "" {
		base = base.Where("cp.title ILIKE ? OR cp.content ILIKE ?", "%"+query+"%", "%"+query+"%")
	}

	totalCount, err := r.db.NewSelect().TableExpr("(?) AS community_posts", base).ColumnExpr("community_posts.id").Count(ctx)
	if err != nil {
		return nil, 0, wrapError("communityRepo.Page count", err)
	}

	sort := strings.TrimSpace(filter.Sort)
	switch sort {
	case "oldest":
		base = base.OrderExpr("cp.created_at ASC, cp.id ASC")
	case "popular":
		base = base.OrderExpr("like_count DESC, cp.view_count DESC, cp.created_at DESC, cp.id DESC")
	default:
		base = base.OrderExpr("cp.created_at DESC, cp.id DESC")
	}

	if err := base.Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &rows); err != nil {
		return nil, 0, wrapError("communityRepo.Page list", err)
	}

	return rows, totalCount, nil
}

func (r *CommunityRepo) HasLikeByPostAndUser(ctx context.Context, postID, userID int64) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*models.CommunityPostLike)(nil)).
		Where("post_id = ?", postID).
		Where("user_id = ?", userID).
		Exists(ctx)
	if err != nil {
		return false, wrapError("communityRepo.HasLikeByPostAndUser", err)
	}

	return exists, nil
}

func (r *CommunityRepo) CreateLike(ctx context.Context, postID, userID int64) error {
	row := &models.CommunityPostLike{
		PostID:    postID,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
	}
	if _, err := r.db.NewInsert().Model(row).Exec(ctx); err != nil {
		return wrapError("communityRepo.CreateLike", err)
	}

	return nil
}

func (r *CommunityRepo) DeleteLike(ctx context.Context, postID, userID int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.CommunityPostLike)(nil)).
		Where("post_id = ?", postID).
		Where("user_id = ?", userID).
		Exec(ctx); err != nil {
		return wrapError("communityRepo.DeleteLike", err)
	}

	return nil
}

func (r *CommunityRepo) CountLikesByPostID(ctx context.Context, postID int64) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.CommunityPostLike)(nil)).
		Where("post_id = ?", postID).
		Count(ctx)
	if err != nil {
		return 0, wrapError("communityRepo.CountLikesByPostID", err)
	}

	return count, nil
}

func (r *CommunityRepo) LikesByPostPage(ctx context.Context, postID int64, page, pageSize int) ([]models.CommunityPostLikeDetail, int, error) {
	rows := make([]models.CommunityPostLikeDetail, 0, pageSize)
	base := r.db.NewSelect().
		TableExpr("community_post_likes AS cpl").
		ColumnExpr("cpl.post_id").
		ColumnExpr("cpl.user_id").
		ColumnExpr("cpl.created_at").
		ColumnExpr("u.username").
		ColumnExpr("u.affiliation_id").
		ColumnExpr("aff.name AS affiliation").
		ColumnExpr("u.bio").
		Join("JOIN users AS u ON u.id = cpl.user_id").
		Join("LEFT JOIN affiliations AS aff ON aff.id = u.affiliation_id").
		Where("cpl.post_id = ?", postID)

	totalCount, err := r.db.NewSelect().TableExpr("(?) AS likes", base).ColumnExpr("likes.user_id").Count(ctx)
	if err != nil {
		return nil, 0, wrapError("communityRepo.LikesByPostPage count", err)
	}

	if err := base.OrderExpr("cpl.created_at DESC, cpl.user_id DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &rows); err != nil {
		if err == sql.ErrNoRows {
			return rows, totalCount, nil
		}

		return nil, 0, wrapError("communityRepo.LikesByPostPage list", err)
	}

	return rows, totalCount, nil
}

func (r *CommunityRepo) CreateComment(ctx context.Context, comment *models.CommunityComment) error {
	if _, err := r.db.NewInsert().Model(comment).Exec(ctx); err != nil {
		return wrapError("communityRepo.CreateComment", err)
	}

	return nil
}

func (r *CommunityRepo) UpdateComment(ctx context.Context, comment *models.CommunityComment) error {
	if _, err := r.db.NewUpdate().
		Model(comment).
		Column("content", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return wrapError("communityRepo.UpdateComment", err)
	}

	return nil
}

func (r *CommunityRepo) DeleteCommentByID(ctx context.Context, commentID int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.CommunityComment)(nil)).
		Where("id = ?", commentID).
		Exec(ctx); err != nil {
		return wrapError("communityRepo.DeleteCommentByID", err)
	}

	return nil
}

func (r *CommunityRepo) GetCommentByID(ctx context.Context, commentID int64) (*models.CommunityComment, error) {
	row := new(models.CommunityComment)
	if err := r.db.NewSelect().Model(row).Where("id = ?", commentID).Limit(1).Scan(ctx); err != nil {
		return nil, wrapNotFound("communityRepo.GetCommentByID", err)
	}

	return row, nil
}

func (r *CommunityRepo) commentDetailBaseQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("community_comments AS cc").
		ColumnExpr("cc.id").
		ColumnExpr("cc.post_id").
		ColumnExpr("cc.user_id").
		ColumnExpr("cc.content").
		ColumnExpr("cc.created_at").
		ColumnExpr("cc.updated_at").
		ColumnExpr("u.username").
		ColumnExpr("u.affiliation_id").
		ColumnExpr("aff.name AS affiliation").
		ColumnExpr("u.bio").
		ColumnExpr("cp.title AS post_title").
		Join("JOIN users AS u ON u.id = cc.user_id").
		Join("JOIN community_posts AS cp ON cp.id = cc.post_id").
		Join("LEFT JOIN affiliations AS aff ON aff.id = u.affiliation_id")
}

func (r *CommunityRepo) GetCommentDetailByID(ctx context.Context, commentID int64) (*models.CommunityCommentDetail, error) {
	row := new(models.CommunityCommentDetail)
	if err := r.commentDetailBaseQuery().Where("cc.id = ?", commentID).Limit(1).Scan(ctx, row); err != nil {
		return nil, wrapNotFound("communityRepo.GetCommentDetailByID", err)
	}

	return row, nil
}

func (r *CommunityRepo) CommentsByPostPage(ctx context.Context, postID int64, page, pageSize int) ([]models.CommunityCommentDetail, int, error) {
	rows := make([]models.CommunityCommentDetail, 0, pageSize)
	base := r.commentDetailBaseQuery().Where("cc.post_id = ?", postID)

	totalCount, err := r.db.NewSelect().TableExpr("(?) AS comments", base).ColumnExpr("comments.id").Count(ctx)
	if err != nil {
		return nil, 0, wrapError("communityRepo.CommentsByPostPage count", err)
	}

	if err := base.OrderExpr("cc.created_at DESC, cc.id DESC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &rows); err != nil {
		if err == sql.ErrNoRows {
			return rows, totalCount, nil
		}

		return nil, 0, wrapError("communityRepo.CommentsByPostPage list", err)
	}

	return rows, totalCount, nil
}
