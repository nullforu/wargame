package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"wargame/internal/config"
	"wargame/internal/db"
	"wargame/internal/models"
	"wargame/internal/repo"
	"wargame/internal/stack"
	"wargame/internal/storage"
	"wargame/internal/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisSubmitPrefix = "submit:"
	maxFlagLength     = 128
	maxWriteupContent = 100000
)

var challengeCategories = map[string]struct{}{
	"Web":         {},
	"Web3":        {},
	"Pwnable":     {},
	"Reversing":   {},
	"Crypto":      {},
	"Forensics":   {},
	"Network":     {},
	"Cloud":       {},
	"Misc":        {},
	"Programming": {},
	"Algorithms":  {},
	"Math":        {},
	"AI":          {},
	"Blockchain":  {},
}

func normalizeStackTargetPorts(ports stack.TargetPortSpecs, validator *fieldValidator) stack.TargetPortSpecs {
	if len(ports) == 0 {
		validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "required"})
		return nil
	}

	normalized := make(stack.TargetPortSpecs, 0, len(ports))
	seen := make(map[string]struct{})
	for _, port := range ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}

		protocol := strings.ToUpper(strings.TrimSpace(port.Protocol))
		if protocol != "TCP" && protocol != "UDP" {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}

		key := fmt.Sprintf("%d/%s", port.ContainerPort, protocol)
		if _, exists := seen[key]; exists {
			validator.fields = append(validator.fields, FieldError{Field: "stack_target_ports", Reason: "invalid"})
			continue
		}
		seen[key] = struct{}{}

		normalized = append(normalized, stack.TargetPortSpec{
			ContainerPort: port.ContainerPort,
			Protocol:      protocol,
		})
	}

	return normalized
}

type WargameService struct {
	cfg            config.Config
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	voteRepo       *repo.ChallengeVoteRepo
	writeupRepo    *repo.WriteupRepo
	redis          *redis.Client
	fileStore      storage.ChallengeFileStore
}

type ChallengeQueryFilter struct {
	Query    string
	Category string
	Level    *int
	Solved   *bool
	UserID   int64
	Sort     string
}

func NewWargameService(cfg config.Config, challengeRepo *repo.ChallengeRepo, submissionRepo *repo.SubmissionRepo, voteRepo *repo.ChallengeVoteRepo, writeupRepo *repo.WriteupRepo, redis *redis.Client, fileStore storage.ChallengeFileStore) *WargameService {
	return &WargameService{cfg: cfg, challengeRepo: challengeRepo, submissionRepo: submissionRepo, voteRepo: voteRepo, writeupRepo: writeupRepo, redis: redis, fileStore: fileStore}
}

func (s *WargameService) ListChallenges(ctx context.Context, page, pageSize int, filter ChallengeQueryFilter) ([]models.Challenge, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	queryFilter := repo.ChallengeListFilter{
		Query:    strings.TrimSpace(filter.Query),
		Category: strings.TrimSpace(filter.Category),
		Level:    filter.Level,
		Sort:     strings.TrimSpace(filter.Sort),
	}
	if filter.Solved != nil {
		if filter.UserID <= 0 {
			return nil, models.Pagination{}, NewValidationError(FieldError{Field: "solved", Reason: "auth_required"})
		}

		queryFilter.Solved = filter.Solved
		queryFilter.SolvedByUserID = &filter.UserID
	}
	if queryFilter.Level != nil && (*queryFilter.Level < models.UnknownLevel || *queryFilter.Level > models.MaxVoteLevel) {
		return nil, models.Pagination{}, NewValidationError(FieldError{Field: "level", Reason: "must be between 0 and 10"})
	}

	if queryFilter.Sort != "" {
		switch queryFilter.Sort {
		case "latest", "oldest", "most_solved", "least_solved":
		default:
			return nil, models.Pagination{}, NewValidationError(FieldError{Field: "sort", Reason: "invalid"})
		}
	}

	challenges, totalCount, err := s.challengeRepo.ListActiveFiltered(ctx, queryFilter, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.ListChallenges: %w", err)
	}

	ptrs := make([]*models.Challenge, 0, len(challenges))
	for i := range challenges {
		ptrs = append(ptrs, &challenges[i])
	}

	if err := s.applyChallengePoints(ctx, ptrs); err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.ListChallenges score: %w", err)
	}

	if err := s.applyChallengeLevels(ctx, ptrs); err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.ListChallenges level: %w", err)
	}

	return challenges, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) SearchChallenges(ctx context.Context, query string, page, pageSize int, filter ChallengeQueryFilter) ([]models.Challenge, models.Pagination, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, models.Pagination{}, NewValidationError(FieldError{Field: "q", Reason: "required"})
	}

	filter.Query = query
	return s.ListChallenges(ctx, page, pageSize, filter)
}

func (s *WargameService) GetChallengeByID(ctx context.Context, id int64) (*models.Challenge, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}

		return nil, fmt.Errorf("wargame.GetChallengeByID: %w", err)
	}

	if err := s.applyChallengePoints(ctx, []*models.Challenge{challenge}); err != nil {
		return nil, fmt.Errorf("wargame.GetChallengeByID score: %w", err)
	}

	if err := s.applyChallengeLevels(ctx, []*models.Challenge{challenge}); err != nil {
		return nil, fmt.Errorf("wargame.GetChallengeByID level: %w", err)
	}

	if err := s.applyChallengeLevelVoteCounts(ctx, challenge); err != nil {
		return nil, fmt.Errorf("wargame.GetChallengeByID vote counts: %w", err)
	}

	return challenge, nil
}

func (s *WargameService) CreateChallenge(ctx context.Context, title, description, category string, points int, flag string, active bool, stackEnabled bool, stackTargetPorts stack.TargetPortSpecs, stackPodSpec *string, previousChallengeID *int64) (*models.Challenge, error) {
	title = normalizeTrim(title)
	description = normalizeTrim(description)
	category = normalizeTrim(category)
	flag = normalizeTrim(flag)
	validator := newFieldValidator()
	validator.Required("title", title)
	validator.Required("description", description)
	validator.Required("category", category)
	validator.Required("flag", flag)
	validator.NonNegative("points", points)
	if previousChallengeID != nil {
		validator.PositiveID("previous_challenge_id", *previousChallengeID)
	}

	if _, ok := challengeCategories[category]; category != "" && !ok {
		validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
	}

	if stackEnabled {
		stackTargetPorts = normalizeStackTargetPorts(stackTargetPorts, validator)
		if stackPodSpec == nil || normalizeTrim(*stackPodSpec) == "" {
			validator.fields = append(validator.fields, FieldError{Field: "stack_pod_spec", Reason: "required"})
		}
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if previousChallengeID != nil {
		if _, err := s.challengeRepo.GetByID(ctx, *previousChallengeID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
			}

			return nil, fmt.Errorf("wargame.CreateChallenge previous challenge: %w", err)
		}
	}

	podSpec := (*string)(nil)
	if stackEnabled && stackPodSpec != nil {
		trimmed := normalizeTrim(*stackPodSpec)
		podSpec = &trimmed
	} else if !stackEnabled {
		stackTargetPorts = nil
	}

	challenge := &models.Challenge{
		Title:               title,
		Description:         description,
		Category:            category,
		Points:              points,
		PreviousChallengeID: previousChallengeID,
		StackEnabled:        stackEnabled,
		StackTargetPorts:    stackTargetPorts,
		StackPodSpec:        podSpec,
		IsActive:            active,
		CreatedAt:           time.Now().UTC(),
	}

	flagHash, err := utils.HashFlag(flag, s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("wargame.CreateChallenge hash flag: %w", err)
	}

	challenge.FlagHash = flagHash

	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("wargame.CreateChallenge: %w", err)
	}

	if err := s.applyChallengePoints(ctx, []*models.Challenge{challenge}); err != nil {
		return nil, fmt.Errorf("wargame.CreateChallenge score: %w", err)
	}

	return challenge, nil
}

func (s *WargameService) UpdateChallenge(ctx context.Context, id int64, title, description, category *string, points *int, flag *string, active *bool, stackEnabled *bool, stackTargetPorts *[]stack.TargetPortSpec, stackPodSpec *string, previousChallengeID *int64, previousChallengeSet bool) (*models.Challenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)

	var normalizedTitle *string
	if title != nil {
		value := *title
		normalizedTitle = &value
	}

	var normalizedDescription *string
	if description != nil {
		value := *description
		normalizedDescription = &value
	}

	var normalizedCategory *string
	if category != nil {
		value := strings.TrimSpace(*category)
		normalizedCategory = &value
		if value == "" {
			validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "required"})
		} else if _, ok := challengeCategories[value]; !ok {
			validator.fields = append(validator.fields, FieldError{Field: "category", Reason: "invalid"})
		}
	}

	var normalizedFlag *string
	if flag != nil {
		value := strings.TrimSpace(*flag)
		if value == "" {
			validator.fields = append(validator.fields, FieldError{Field: "flag", Reason: "required"})
		} else {
			normalizedFlag = &value
		}
	}

	var normalizedPodSpec *string
	if stackPodSpec != nil {
		value := strings.TrimSpace(*stackPodSpec)
		normalizedPodSpec = &value
	}

	if points != nil {
		validator.NonNegative("points", *points)
	}

	if previousChallengeSet && previousChallengeID != nil {
		validator.PositiveID("previous_challenge_id", *previousChallengeID)
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("wargame.UpdateChallenge lookup: %w", err)
	}

	if previousChallengeSet && previousChallengeID != nil {
		if *previousChallengeID == challenge.ID {
			return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
		}

		if _, err := s.challengeRepo.GetByID(ctx, *previousChallengeID); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, NewValidationError(FieldError{Field: "previous_challenge_id", Reason: "invalid"})
			}

			return nil, fmt.Errorf("wargame.UpdateChallenge previous challenge: %w", err)
		}
	}

	if normalizedTitle != nil {
		challenge.Title = *normalizedTitle
	}

	if normalizedDescription != nil {
		challenge.Description = *normalizedDescription
	}

	if normalizedCategory != nil {
		challenge.Category = *normalizedCategory
	}

	if points != nil {
		challenge.Points = *points
	}

	if previousChallengeSet {
		challenge.PreviousChallengeID = previousChallengeID
	}

	if active != nil {
		challenge.IsActive = *active
	}

	if stackEnabled != nil {
		challenge.StackEnabled = *stackEnabled
		if !*stackEnabled {
			challenge.StackTargetPorts = nil
			challenge.StackPodSpec = nil
		}
	}

	if stackTargetPorts != nil {
		if !challenge.StackEnabled {
			return nil, NewValidationError(FieldError{Field: "stack_target_ports", Reason: "stack disabled"})
		}

		validator := newFieldValidator()
		normalized := normalizeStackTargetPorts(stack.TargetPortSpecs(*stackTargetPorts), validator)
		if err := validator.Error(); err != nil {
			return nil, err
		}

		challenge.StackTargetPorts = normalized
	}

	if normalizedPodSpec != nil {
		if !challenge.StackEnabled {
			return nil, NewValidationError(FieldError{Field: "stack_pod_spec", Reason: "stack disabled"})
		}

		if *normalizedPodSpec == "" {
			challenge.StackPodSpec = nil
		} else {
			challenge.StackPodSpec = normalizedPodSpec
		}
	}

	if normalizedFlag != nil {
		flagHash, err := utils.HashFlag(*normalizedFlag, s.cfg.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("wargame.UpdateChallenge hash flag: %w", err)
		}

		challenge.FlagHash = flagHash
	}

	if challenge.StackEnabled {
		if len(challenge.StackTargetPorts) == 0 {
			return nil, NewValidationError(FieldError{Field: "stack_target_ports", Reason: "required"})
		}

		if challenge.StackPodSpec == nil || normalizeTrim(*challenge.StackPodSpec) == "" {
			return nil, NewValidationError(FieldError{Field: "stack_pod_spec", Reason: "required"})
		}
	}

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, fmt.Errorf("wargame.UpdateChallenge update: %w", err)
	}

	if err := s.applyChallengePoints(ctx, []*models.Challenge{challenge}); err != nil {
		return nil, fmt.Errorf("wargame.UpdateChallenge score: %w", err)
	}

	return challenge, nil
}

func (s *WargameService) VoteChallengeLevel(ctx context.Context, userID, challengeID int64, level int) error {
	validator := newFieldValidator()
	validator.PositiveID("challenge_id", challengeID)
	if userID <= 0 {
		validator.fields = append(validator.fields, FieldError{Field: "user_id", Reason: "invalid"})
	}

	if level < models.MinVoteLevel || level > models.MaxVoteLevel {
		validator.fields = append(validator.fields, FieldError{Field: "level", Reason: "invalid"})
	}

	if err := validator.Error(); err != nil {
		return err
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrChallengeNotFound
		}
		return fmt.Errorf("wargame.VoteChallengeLevel challenge lookup: %w", err)
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return fmt.Errorf("wargame.VoteChallengeLevel solved check: %w", err)
	}

	if !solved {
		return ErrChallengeNotSolvedByUser
	}

	now := time.Now().UTC()
	if err := s.voteRepo.Upsert(ctx, &models.ChallengeVote{
		ChallengeID: challengeID,
		UserID:      userID,
		Level:       level,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("wargame.VoteChallengeLevel upsert: %w", err)
	}

	return nil
}

func (s *WargameService) ChallengeVotesPage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.ChallengeVoteDetail, models.Pagination, error) {
	validator := newFieldValidator()
	validator.PositiveID("challenge_id", challengeID)
	if err := validator.Error(); err != nil {
		return nil, models.Pagination{}, err
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, models.Pagination{}, ErrChallengeNotFound
		}

		return nil, models.Pagination{}, fmt.Errorf("wargame.ChallengeVotesPage challenge lookup: %w", err)
	}

	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.voteRepo.VotesByChallengePage(ctx, challengeID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.ChallengeVotesPage: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) ChallengeVoteLevelByUser(ctx context.Context, userID, challengeID int64) (*int, error) {
	validator := newFieldValidator()
	validator.PositiveID("challenge_id", challengeID)
	if userID <= 0 {
		validator.fields = append(validator.fields, FieldError{Field: "user_id", Reason: "invalid"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}

		return nil, fmt.Errorf("wargame.ChallengeVoteLevelByUser challenge lookup: %w", err)
	}

	level, err := s.voteRepo.VoteLevelByUserAndChallengeID(ctx, userID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("wargame.ChallengeVoteLevelByUser: %w", err)
	}

	return level, nil
}

func (s *WargameService) DeleteChallenge(ctx context.Context, id int64) error {
	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrChallengeNotFound
		}
		return fmt.Errorf("wargame.DeleteChallenge lookup: %w", err)
	}

	if err := s.challengeRepo.Delete(ctx, challenge); err != nil {
		return fmt.Errorf("wargame.DeleteChallenge delete: %w", err)
	}

	return nil
}

func (s *WargameService) UpdateChallengeCreator(ctx context.Context, challengeID, userID int64) error {
	if challengeID <= 0 || userID <= 0 {
		return ErrInvalidInput
	}

	challenge, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrChallengeNotFound
		}

		return fmt.Errorf("wargame.UpdateChallengeCreator lookup: %w", err)
	}

	challenge.CreatedByUserID = &userID
	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return fmt.Errorf("wargame.UpdateChallengeCreator update: %w", err)
	}

	return nil
}

func (s *WargameService) SubmitFlag(ctx context.Context, userID, challengeID int64, flag string) (bool, error) {
	flag = normalizeTrim(flag)
	validator := newFieldValidator()
	validator.Required("flag", flag)
	validator.PositiveID("challenge_id", challengeID)

	if err := validator.Error(); err != nil {
		return false, err
	}

	challenge, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return false, ErrChallengeNotFound
		}
		return false, fmt.Errorf("wargame.SubmitFlag lookup: %w", err)
	}

	if !challenge.IsActive {
		return false, ErrChallengeNotFound
	}

	if err := s.ensureUnlocked(ctx, userID, challenge); err != nil {
		return false, err
	}

	if err := s.rateLimit(ctx, userID); err != nil {
		return false, err
	}

	already, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return false, fmt.Errorf("wargame.SubmitFlag check: %w", err)
	}

	if already {
		return true, ErrAlreadySolved
	}

	correct, err := utils.CheckFlag(challenge.FlagHash, flag)
	if err != nil {
		return false, fmt.Errorf("wargame.SubmitFlag check flag: %w", err)
	}

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    trimTo(flag, maxFlagLength),
		Correct:     correct,
		SubmittedAt: time.Now().UTC(),
	}

	if correct {
		inserted, err := s.submissionRepo.CreateCorrectIfNotSolvedByUser(ctx, sub)
		if err != nil {
			return false, fmt.Errorf("wargame.SubmitFlag create: %w", err)
		}

		if !inserted {
			return true, ErrAlreadySolved
		}
	} else if err := s.submissionRepo.Create(ctx, sub); err != nil {
		return false, fmt.Errorf("wargame.SubmitFlag create: %w", err)
	}

	return correct, nil
}

func (s *WargameService) RequestChallengeFileUpload(ctx context.Context, id int64, filename string) (*models.Challenge, storage.PresignedPost, error) {
	filename = normalizeTrim(filename)
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	validator.Required("filename", filename)

	if !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		validator.fields = append(validator.fields, FieldError{Field: "filename", Reason: "must be a .zip file"})
	}

	if err := validator.Error(); err != nil {
		return nil, storage.PresignedPost{}, err
	}

	if s.fileStore == nil {
		return nil, storage.PresignedPost{}, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, storage.PresignedPost{}, ErrChallengeNotFound
		}
		return nil, storage.PresignedPost{}, fmt.Errorf("wargame.RequestChallengeFileUpload lookup: %w", err)
	}

	key := uuid.NewString() + ".zip"
	upload, err := s.fileStore.PresignUpload(ctx, key, "application/zip")
	if err != nil {
		return nil, storage.PresignedPost{}, fmt.Errorf("wargame.RequestChallengeFileUpload presign: %w", err)
	}

	previousKey := challenge.FileKey
	now := time.Now().UTC()
	challenge.FileKey = &key
	challenge.FileName = &filename
	challenge.FileUploadedAt = &now

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, storage.PresignedPost{}, fmt.Errorf("wargame.RequestChallengeFileUpload update: %w", err)
	}

	if previousKey != nil && *previousKey != "" && s.fileStore != nil {
		// Best-effort cleanup. Keep the new file active even if cleanup fails. See https://github.com/nullforu/wargame/pull/38 for more details.
		_ = s.fileStore.Delete(ctx, *previousKey)
	}

	return challenge, upload, nil
}

func (s *WargameService) RequestChallengeFileDownload(ctx context.Context, userID, id int64) (storage.PresignedURL, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return storage.PresignedURL{}, err
	}

	if s.fileStore == nil {
		return storage.PresignedURL{}, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return storage.PresignedURL{}, ErrChallengeNotFound
		}
		return storage.PresignedURL{}, fmt.Errorf("wargame.RequestChallengeFileDownload lookup: %w", err)
	}

	if err := s.ensureUnlocked(ctx, userID, challenge); err != nil {
		return storage.PresignedURL{}, err
	}

	if challenge.FileKey == nil || *challenge.FileKey == "" {
		return storage.PresignedURL{}, ErrChallengeFileNotFound
	}

	filename := ""
	if challenge.FileName != nil {
		filename = *challenge.FileName
	}
	download, err := s.fileStore.PresignDownload(ctx, *challenge.FileKey, filename)
	if err != nil {
		return storage.PresignedURL{}, fmt.Errorf("wargame.RequestChallengeFileDownload presign: %w", err)
	}

	return download, nil
}

func (s *WargameService) SolvedChallengeIDs(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	if userID <= 0 || s.submissionRepo == nil {
		return map[int64]struct{}{}, nil
	}

	ids, err := s.submissionRepo.SolvedChallengeIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("wargame.SolvedChallengeIDs: %w", err)
	}

	return ids, nil
}

func (s *WargameService) ensureUnlocked(ctx context.Context, userID int64, challenge *models.Challenge) error {
	if challenge.PreviousChallengeID == nil || *challenge.PreviousChallengeID <= 0 {
		return nil
	}

	if userID <= 0 || s.submissionRepo == nil {
		return ErrChallengeLocked
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, *challenge.PreviousChallengeID)
	if err != nil {
		return fmt.Errorf("wargame.ensureUnlocked: %w", err)
	}

	if !solved {
		return ErrChallengeLocked
	}

	return nil
}

func (s *WargameService) DeleteChallengeFile(ctx context.Context, id int64) (*models.Challenge, error) {
	validator := newFieldValidator()
	validator.PositiveID("id", id)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	if s.fileStore == nil {
		return nil, ErrStorageUnavailable
	}

	challenge, err := s.challengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("wargame.DeleteChallengeFile lookup: %w", err)
	}

	if challenge.FileKey == nil || *challenge.FileKey == "" {
		return nil, ErrChallengeFileNotFound
	}

	if err := s.fileStore.Delete(ctx, *challenge.FileKey); err != nil {
		return nil, fmt.Errorf("wargame.DeleteChallengeFile delete: %w", err)
	}

	challenge.FileKey = nil
	challenge.FileName = nil
	challenge.FileUploadedAt = nil

	if err := s.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, fmt.Errorf("wargame.DeleteChallengeFile update: %w", err)
	}

	return challenge, nil
}
func (s *WargameService) SolvedChallengesPage(ctx context.Context, userID int64, page, pageSize int) ([]models.SolvedChallenge, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	rows, totalCount, err := s.submissionRepo.SolvedChallengesPage(ctx, userID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.SolvedChallengesPage: %w", err)
	}

	pointsMap, err := s.challengeRepo.ChallengePoints(ctx)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.SolvedChallengesPage score: %w", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) ChallengeSolversPage(ctx context.Context, challengeID int64, page, pageSize int) ([]models.ChallengeSolver, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	if challengeID <= 0 {
		return nil, models.Pagination{}, ErrInvalidInput
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, models.Pagination{}, ErrChallengeNotFound
		}

		return nil, models.Pagination{}, fmt.Errorf("wargame.ChallengeSolversPage lookup: %w", err)
	}

	rows, totalCount, err := s.submissionRepo.ChallengeSolversPage(ctx, challengeID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.ChallengeSolversPage: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) ChallengeFirstBlood(ctx context.Context, challengeID int64) (*models.ChallengeSolver, error) {
	if challengeID <= 0 {
		return nil, ErrInvalidInput
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("wargame.ChallengeFirstBlood lookup: %w", err)
	}

	row, err := s.submissionRepo.ChallengeFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("wargame.ChallengeFirstBlood: %w", err)
	}

	return row, nil
}

func (s *WargameService) ListAllSubmissions(ctx context.Context) ([]models.Submission, error) {
	rows, err := s.submissionRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("wargame.ListAllSubmissions: %w", err)
	}

	return rows, nil
}

func (s *WargameService) CreateWriteup(ctx context.Context, userID, challengeID int64, content string) (*models.WriteupDetail, error) {
	content = strings.TrimSpace(content)

	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("challenge_id", challengeID)
	validator.Required("content", content)
	if len(content) > maxWriteupContent {
		validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "too_long"})
	}

	if err := validator.Error(); err != nil {
		return nil, err
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrChallengeNotFound
		}

		return nil, fmt.Errorf("wargame.CreateWriteup challenge lookup: %w", err)
	}

	solved, err := s.submissionRepo.HasCorrect(ctx, userID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("wargame.CreateWriteup solved check: %w", err)
	}

	if !solved {
		return nil, ErrChallengeNotSolvedByUser
	}

	if _, err := s.writeupRepo.GetByUserAndChallenge(ctx, userID, challengeID); err == nil {
		return nil, ErrWriteupExists
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("wargame.CreateWriteup existing lookup: %w", err)
	}

	now := time.Now().UTC()
	row := &models.Writeup{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.writeupRepo.Create(ctx, row); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrWriteupExists
		}
		return nil, fmt.Errorf("wargame.CreateWriteup create: %w", err)
	}

	detail, err := s.writeupRepo.GetDetailByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("wargame.CreateWriteup detail: %w", err)
	}

	if err := s.applyWriteupChallengeLevels(ctx, []*models.WriteupDetail{detail}); err != nil {
		return nil, fmt.Errorf("wargame.CreateWriteup level: %w", err)
	}

	return detail, nil
}

func (s *WargameService) UpdateWriteup(ctx context.Context, userID, writeupID int64, content *string) (*models.WriteupDetail, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("id", writeupID)
	if content == nil {
		validator.fields = append(validator.fields, FieldError{Field: "request", Reason: "empty"})
	}

	if content != nil {
		normalizedContent := strings.TrimSpace(*content)
		if normalizedContent == "" {
			validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "required"})
		}

		if len(normalizedContent) > maxWriteupContent {
			validator.fields = append(validator.fields, FieldError{Field: "content", Reason: "too_long"})
		}
	}
	if err := validator.Error(); err != nil {
		return nil, err
	}

	row, err := s.writeupRepo.GetByID(ctx, writeupID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrWriteupNotFound
		}

		return nil, fmt.Errorf("wargame.UpdateWriteup lookup: %w", err)
	}

	if row.UserID != userID {
		return nil, ErrWriteupForbidden
	}

	if content != nil {
		row.Content = strings.TrimSpace(*content)
	}
	row.UpdatedAt = time.Now().UTC()

	if err := s.writeupRepo.Update(ctx, row); err != nil {
		return nil, fmt.Errorf("wargame.UpdateWriteup update: %w", err)
	}

	detail, err := s.writeupRepo.GetDetailByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("wargame.UpdateWriteup detail: %w", err)
	}

	if err := s.applyWriteupChallengeLevels(ctx, []*models.WriteupDetail{detail}); err != nil {
		return nil, fmt.Errorf("wargame.UpdateWriteup level: %w", err)
	}

	return detail, nil
}

func (s *WargameService) DeleteWriteup(ctx context.Context, userID, writeupID int64) error {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("id", writeupID)
	if err := validator.Error(); err != nil {
		return err
	}

	row, err := s.writeupRepo.GetByID(ctx, writeupID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrWriteupNotFound
		}

		return fmt.Errorf("wargame.DeleteWriteup lookup: %w", err)
	}

	if row.UserID != userID {
		return ErrWriteupForbidden
	}

	if err := s.writeupRepo.DeleteByID(ctx, writeupID); err != nil {
		return fmt.Errorf("wargame.DeleteWriteup delete: %w", err)
	}

	return nil
}

func (s *WargameService) ChallengeWriteupsPage(ctx context.Context, challengeID, viewerUserID int64, page, pageSize int) ([]models.WriteupDetail, models.Pagination, bool, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, false, err
	}

	if challengeID <= 0 {
		return nil, models.Pagination{}, false, ErrInvalidInput
	}

	if _, err := s.challengeRepo.GetByID(ctx, challengeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, models.Pagination{}, false, ErrChallengeNotFound
		}

		return nil, models.Pagination{}, false, fmt.Errorf("wargame.ChallengeWriteupsPage challenge lookup: %w", err)
	}

	canViewContent := false
	if viewerUserID > 0 {
		canViewContent, err = s.submissionRepo.HasCorrect(ctx, viewerUserID, challengeID)
		if err != nil {
			return nil, models.Pagination{}, false, fmt.Errorf("wargame.ChallengeWriteupsPage viewer solved check: %w", err)
		}
	}

	var rows []models.WriteupDetail
	var totalCount int
	if canViewContent {
		rows, totalCount, err = s.writeupRepo.ChallengePage(ctx, challengeID, params.Page, params.PageSize)
	} else {
		rows, totalCount, err = s.writeupRepo.ChallengePageWithoutContent(ctx, challengeID, params.Page, params.PageSize)
	}
	if err != nil {
		return nil, models.Pagination{}, false, fmt.Errorf("wargame.ChallengeWriteupsPage list: %w", err)
	}

	rowPtrs := make([]*models.WriteupDetail, 0, len(rows))
	for i := range rows {
		rowPtrs = append(rowPtrs, &rows[i])
	}

	if err := s.applyWriteupChallengeLevels(ctx, rowPtrs); err != nil {
		return nil, models.Pagination{}, false, fmt.Errorf("wargame.ChallengeWriteupsPage level: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), canViewContent, nil
}

func (s *WargameService) GetWriteupByID(ctx context.Context, writeupID, viewerUserID int64) (*models.WriteupDetail, bool, error) {
	if writeupID <= 0 {
		return nil, false, ErrInvalidInput
	}

	row, err := s.writeupRepo.GetDetailByIDWithoutContent(ctx, writeupID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, false, ErrWriteupNotFound
		}

		return nil, false, fmt.Errorf("wargame.GetWriteupByID detail: %w", err)
	}

	canViewContent := false
	if viewerUserID > 0 {
		canViewContent, err = s.submissionRepo.HasCorrect(ctx, viewerUserID, row.ChallengeID)
		if err != nil {
			return nil, false, fmt.Errorf("wargame.GetWriteupByID viewer solved check: %w", err)
		}
	}

	if canViewContent {
		row, err = s.writeupRepo.GetDetailByID(ctx, writeupID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return nil, false, ErrWriteupNotFound
			}
			return nil, false, fmt.Errorf("wargame.GetWriteupByID detail with content: %w", err)
		}
	}

	if err := s.applyWriteupChallengeLevels(ctx, []*models.WriteupDetail{row}); err != nil {
		return nil, false, fmt.Errorf("wargame.GetWriteupByID level: %w", err)
	}

	return row, canViewContent, nil
}

func (s *WargameService) MyWriteupByChallenge(ctx context.Context, userID, challengeID int64) (*models.WriteupDetail, error) {
	validator := newFieldValidator()
	validator.PositiveID("user_id", userID)
	validator.PositiveID("challenge_id", challengeID)
	if err := validator.Error(); err != nil {
		return nil, err
	}

	row, err := s.writeupRepo.GetByUserAndChallenge(ctx, userID, challengeID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrWriteupNotFound
		}
		return nil, fmt.Errorf("wargame.MyWriteupByChallenge lookup: %w", err)
	}

	detail, err := s.writeupRepo.GetDetailByID(ctx, row.ID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrWriteupNotFound
		}
		return nil, fmt.Errorf("wargame.MyWriteupByChallenge detail: %w", err)
	}

	if err := s.applyWriteupChallengeLevels(ctx, []*models.WriteupDetail{detail}); err != nil {
		return nil, fmt.Errorf("wargame.MyWriteupByChallenge level: %w", err)
	}

	return detail, nil
}

func (s *WargameService) MyWriteupsPage(ctx context.Context, userID int64, page, pageSize int) ([]models.WriteupDetail, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	if userID <= 0 {
		return nil, models.Pagination{}, ErrInvalidInput
	}

	rows, totalCount, err := s.writeupRepo.UserPage(ctx, userID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.MyWriteupsPage list: %w", err)
	}

	rowPtrs := make([]*models.WriteupDetail, 0, len(rows))
	for i := range rows {
		rowPtrs = append(rowPtrs, &rows[i])
	}

	if err := s.applyWriteupChallengeLevels(ctx, rowPtrs); err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.MyWriteupsPage level: %w", err)
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) UserWriteupsPage(ctx context.Context, targetUserID, viewerUserID int64, page, pageSize int) ([]models.WriteupDetail, models.Pagination, error) {
	params, err := NormalizePagination(page, pageSize)
	if err != nil {
		return nil, models.Pagination{}, err
	}

	if targetUserID <= 0 {
		return nil, models.Pagination{}, ErrInvalidInput
	}

	rows, totalCount, err := s.writeupRepo.UserPage(ctx, targetUserID, params.Page, params.PageSize)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.UserWriteupsPage list: %w", err)
	}

	rowPtrs := make([]*models.WriteupDetail, 0, len(rows))
	for i := range rows {
		rowPtrs = append(rowPtrs, &rows[i])
	}

	if err := s.applyWriteupChallengeLevels(ctx, rowPtrs); err != nil {
		return nil, models.Pagination{}, fmt.Errorf("wargame.UserWriteupsPage level: %w", err)
	}

	if viewerUserID > 0 {
		solvedIDs, err := s.SolvedChallengeIDs(ctx, viewerUserID)
		if err != nil {
			return nil, models.Pagination{}, fmt.Errorf("wargame.UserWriteupsPage solved ids: %w", err)
		}

		for i := range rows {
			if _, ok := solvedIDs[rows[i].ChallengeID]; !ok {
				rows[i].Content = ""
			}
		}
	} else {
		for i := range rows {
			rows[i].Content = ""
		}
	}

	return rows, BuildPagination(params.Page, params.PageSize, totalCount), nil
}

func (s *WargameService) applyWriteupChallengeLevels(ctx context.Context, rows []*models.WriteupDetail) error {
	if len(rows) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(rows))
	seen := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.ChallengeID]; ok {
			continue
		}

		seen[row.ChallengeID] = struct{}{}
		ids = append(ids, row.ChallengeID)
	}

	levels, err := s.voteRepo.RepresentativeLevelsByChallengeIDs(ctx, ids)
	if err != nil {
		return err
	}

	pointsMap, err := s.challengeRepo.ChallengePointsByIDs(ctx, ids)
	if err != nil {
		return err
	}

	for i := range rows {
		rows[i].ChallengeLevel = levels[rows[i].ChallengeID]
		if points, ok := pointsMap[rows[i].ChallengeID]; ok {
			rows[i].ChallengePoints = points
		}
	}

	return nil
}

func (s *WargameService) applyChallengePoints(ctx context.Context, challenges []*models.Challenge) error {
	if len(challenges) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(challenges))
	for _, challenge := range challenges {
		ids = append(ids, challenge.ID)
	}

	pointsMap, err := s.challengeRepo.ChallengePointsByIDs(ctx, ids)
	if err != nil {
		return err
	}

	solveCounts, err := s.challengeRepo.SolveCountsByIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, challenge := range challenges {
		if points, ok := pointsMap[challenge.ID]; ok {
			challenge.Points = points
		}

		challenge.SolveCount = solveCounts[challenge.ID]
	}

	return nil
}

func (s *WargameService) applyChallengeLevels(ctx context.Context, challenges []*models.Challenge) error {
	if len(challenges) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(challenges))
	for _, challenge := range challenges {
		ids = append(ids, challenge.ID)
	}

	levels, err := s.voteRepo.RepresentativeLevelsByChallengeIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, challenge := range challenges {
		challenge.Level = levels[challenge.ID]
	}

	return nil
}

func (s *WargameService) applyChallengeLevelVoteCounts(ctx context.Context, challenge *models.Challenge) error {
	if challenge == nil {
		return nil
	}

	counts, err := s.voteRepo.VoteCountsByChallengeID(ctx, challenge.ID)
	if err != nil {
		return err
	}
	challenge.LevelVotes = counts

	return nil
}
