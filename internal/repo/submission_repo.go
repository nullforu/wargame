package repo

import (
	"context"

	"wargame/internal/models"

	"github.com/uptrace/bun"
)

type SubmissionRepo struct {
	db *bun.DB
}

func NewSubmissionRepo(db *bun.DB) *SubmissionRepo {
	return &SubmissionRepo{db: db}
}

func (r *SubmissionRepo) ListAll(ctx context.Context) ([]models.Submission, error) {
	rows := make([]models.Submission, 0)
	if err := r.db.NewSelect().
		Model(&rows).
		OrderExpr("submitted_at DESC, id DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("submissionRepo.ListAll", err)
	}

	return rows, nil
}

func (r *SubmissionRepo) Create(ctx context.Context, sub *models.Submission) error {
	if _, err := r.db.NewInsert().Model(sub).Exec(ctx); err != nil {
		return wrapError("submissionRepo.Create", err)
	}

	return nil
}

func (r *SubmissionRepo) lockUserScope(ctx context.Context, db bun.IDB, userID int64) error {
	var lockedID int64
	if err := db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id").
		Where("u.id = ?", userID).
		For("UPDATE").
		Scan(ctx, &lockedID); err != nil {
		return err
	}

	return nil
}

func (r *SubmissionRepo) lockChallengeScope(ctx context.Context, db bun.IDB, challengeID int64) error {
	var lockedID int64
	if err := db.NewSelect().
		TableExpr("challenges AS c").
		ColumnExpr("c.id").
		Where("c.id = ?", challengeID).
		For("UPDATE").
		Scan(ctx, &lockedID); err != nil {
		return err
	}

	return nil
}

func (r *SubmissionRepo) correctSubmissionCount(ctx context.Context, db bun.IDB, challengeID, userID int64) (int, error) {
	return db.NewSelect().
		TableExpr("submissions AS s").
		Where("s.correct = true").
		Where("s.challenge_id = ?", challengeID).
		Where("s.user_id = ?", userID).
		Count(ctx)
}

func (r *SubmissionRepo) correctSubmissionCountForChallenge(ctx context.Context, db bun.IDB, challengeID int64) (int, error) {
	return db.NewSelect().
		TableExpr("submissions AS s").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.role != ?", models.BlockedRole).
		Where("s.challenge_id = ?", challengeID).
		Count(ctx)
}

func (r *SubmissionRepo) solvedChallengesQuery(db bun.IDB) *bun.SelectQuery {
	return db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")
}

func (r *SubmissionRepo) CreateCorrectIfNotSolvedByUser(ctx context.Context, sub *models.Submission) (bool, error) {
	if !sub.Correct {
		if err := r.Create(ctx, sub); err != nil {
			return false, err
		}
		return true, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser begin", err)
	}

	if err := r.lockUserScope(ctx, tx, sub.UserID); err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser lock user", err)
	}

	if err := r.lockChallengeScope(ctx, tx, sub.ChallengeID); err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser lock challenge", err)
	}

	count, err := r.correctSubmissionCount(ctx, tx, sub.ChallengeID, sub.UserID)
	if err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser check", err)
	}

	if count > 0 {
		_ = tx.Rollback()
		return false, nil
	}

	challengeCount, err := r.correctSubmissionCountForChallenge(ctx, tx, sub.ChallengeID)
	if err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser first blood check", err)
	}

	sub.IsFirstBlood = challengeCount == 0

	if _, err := tx.NewInsert().Model(sub).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser insert", err)
	}

	if err := tx.Commit(); err != nil {
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByUser commit", err)
	}

	return true, nil
}

func (r *SubmissionRepo) HasCorrect(ctx context.Context, userID, challengeID int64) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("submissions AS s").
		Where("s.correct = true").
		Where("s.challenge_id = ?", challengeID).
		Where("s.user_id = ?", userID).
		Count(ctx)
	if err != nil {
		return false, wrapError("submissionRepo.HasCorrect", err)
	}

	return count > 0, nil
}

func (r *SubmissionRepo) SolvedChallengesPage(ctx context.Context, userID int64, page, pageSize int) ([]models.SolvedChallenge, int, error) {
	rows := make([]models.SolvedChallenge, 0)
	base := r.solvedChallengesQuery(r.db).Where("s.user_id = ?", userID).GroupExpr("s.challenge_id, c.title, c.points")

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS solved", base).
		ColumnExpr("solved.challenge_id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("submissionRepo.SolvedChallengesPage count", err)
	}

	err = base.
		OrderExpr("solved_at ASC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows)
	if err != nil {
		return nil, 0, wrapError("submissionRepo.SolvedChallengesPage list", err)
	}

	return rows, totalCount, nil
}

func (r *SubmissionRepo) SolvedChallengeIDs(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	rows := make([]int64, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("DISTINCT s.challenge_id").
		Where("s.correct = true").
		Where("s.user_id = ?", userID).
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("submissionRepo.SolvedChallengeIDs", err)
	}

	ids := make(map[int64]struct{}, len(rows))
	for _, id := range rows {
		ids[id] = struct{}{}
	}

	return ids, nil
}

func (r *SubmissionRepo) ChallengeSolversPage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.ChallengeSolver, int, error) {
	rows := make([]models.ChallengeSolver, 0, pageSize)

	base := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.user_id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		ColumnExpr("BOOL_OR(s.is_first_blood) AS is_first_blood").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("s.challenge_id = ?", challengeID).
		GroupExpr("s.user_id, u.username")

	totalCount, err := r.db.NewSelect().
		TableExpr("(?) AS solvers", base).
		ColumnExpr("solvers.user_id").
		Count(ctx)
	if err != nil {
		return nil, 0, wrapError("submissionRepo.ChallengeSolversPage count", err)
	}

	err = base.
		OrderExpr("solved_at ASC, user_id ASC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows)
	if err != nil {
		return nil, 0, wrapError("submissionRepo.ChallengeSolversPage list", err)
	}

	return rows, totalCount, nil
}
