package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wargame/internal/auth"
	"wargame/internal/config"
	"wargame/internal/http/middleware"
	"wargame/internal/models"
	"wargame/internal/realtime"
	"wargame/internal/service"
	"wargame/internal/stack"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	cfg     config.Config
	auth    *service.AuthService
	wargame *service.WargameService
	users   *service.UserService
	score   *service.ScoreboardService
	stacks  *service.StackService
	redis   *redis.Client
}

func New(cfg config.Config, auth *service.AuthService, wargame *service.WargameService, users *service.UserService, score *service.ScoreboardService, stacks *service.StackService, redis *redis.Client) *Handler {
	return &Handler{cfg: cfg, auth: auth, wargame: wargame, users: users, score: score, stacks: stacks, redis: redis}
}

func (h *Handler) respondFromCache(ctx *gin.Context, cacheKey string) bool {
	cached, err := h.redis.Get(ctx.Request.Context(), cacheKey).Result()
	if err != nil {
		return false
	}

	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cached))
	return true
}

func (h *Handler) storeCache(ctx *gin.Context, cacheKey string, response any, ttl time.Duration) {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return
	}
	_ = h.redis.Set(ctx.Request.Context(), cacheKey, responseJSON, ttl).Err()
}

func (h *Handler) invalidateCacheByPrefix(ctx context.Context, prefix string) {
	iter := h.redis.Scan(ctx, 0, prefix+"*", 200).Iterator()
	for iter.Next(ctx) {
		_ = h.redis.Del(ctx, iter.Val()).Err()
	}
	if err := iter.Err(); err != nil {
		slog.Warn("cache scan failed", slog.String("prefix", prefix), slog.Any("error", err))
	}
}

func (h *Handler) notifyScoreboardChanged(ctx context.Context, reason string) {
	h.invalidateCacheByPrefix(ctx, "leaderboard:")
	h.invalidateCacheByPrefix(ctx, "timeline:")
	event := realtime.ScoreboardEvent{Scope: "all", Reason: reason, TS: time.Now().UTC()}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	_ = h.redis.Publish(ctx, "scoreboard.events", payload).Err()
}

func parseIDParam(ctx *gin.Context, name string) (int64, bool) {
	value := strings.TrimSpace(ctx.Param(name))
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseIDParamOrError(ctx *gin.Context, name string) (int64, bool) {
	id, ok := parseIDParam(ctx, name)
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: name, Reason: "invalid"}}})
		return 0, false
	}
	return id, true
}

func parsePaginationParams(ctx *gin.Context) (int, int, bool) {
	pageRaw := strings.TrimSpace(ctx.Query("page"))
	pageSizeRaw := strings.TrimSpace(ctx.Query("page_size"))

	page := 0
	if pageRaw != "" {
		parsed, err := strconv.Atoi(pageRaw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "page", Reason: "invalid"}}})
			return 0, 0, false
		}
		page = parsed
	}

	pageSize := 0
	if pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "page_size", Reason: "invalid"}}})
			return 0, 0, false
		}
		pageSize = parsed
	}

	return page, pageSize, true
}

func parseSearchQuery(ctx *gin.Context) (string, bool) {
	query := strings.TrimSpace(ctx.Query("q"))
	if query == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "q", Reason: "required"}}})
		return "", false
	}

	return query, true
}

type challengeQueryFilters struct {
	Category string
	Level    *int
	Solved   *bool
}

func parseChallengeFilters(ctx *gin.Context) (challengeQueryFilters, bool) {
	category := strings.TrimSpace(ctx.Query("category"))

	var level *int
	if levelRaw := strings.TrimSpace(ctx.Query("level")); levelRaw != "" {
		parsed, err := strconv.Atoi(levelRaw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "level", Reason: "invalid"}}})
			return challengeQueryFilters{}, false
		}
		level = &parsed
	}

	var solved *bool
	if solvedRaw := strings.TrimSpace(ctx.Query("solved")); solvedRaw != "" {
		parsed, err := strconv.ParseBool(solvedRaw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "solved", Reason: "invalid"}}})
			return challengeQueryFilters{}, false
		}
		solved = &parsed
	}

	return challengeQueryFilters{
		Category: category,
		Level:    level,
		Solved:   solved,
	}, true
}

func (h *Handler) optionalUserID(ctx *gin.Context) int64 {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return 0
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0
	}
	claims, err := auth.ParseToken(h.cfg.JWT, parts[1])
	if err != nil || claims.Type != auth.TokenTypeAccess {
		return 0
	}
	return claims.UserID
}

func isChallengeLocked(challenge models.Challenge, solved map[int64]struct{}, userID int64) bool {
	if challenge.PreviousChallengeID == nil || *challenge.PreviousChallengeID <= 0 {
		return false
	}
	if userID <= 0 {
		return true
	}
	_, ok := solved[*challenge.PreviousChallengeID]
	return !ok
}

func (h *Handler) previousChallengeForResponse(ctx context.Context, byID map[int64]*models.Challenge, previousChallengeID *int64) *models.Challenge {
	if previousChallengeID == nil {
		return nil
	}

	if previous, ok := byID[*previousChallengeID]; ok {
		return previous
	}

	previous, err := h.wargame.GetChallengeByID(ctx, *previousChallengeID)
	if err != nil {
		return nil
	}

	return previous
}

func (h *Handler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	user, err := h.auth.Register(ctx.Request.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, registerResponse{ID: user.ID, Email: user.Email, Username: user.Username})
}

func (h *Handler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	accessToken, refreshToken, user, err := h.auth.Login(ctx.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeError(ctx, err)
		return
	}
	stackCount, stackLimit, _ := h.stacks.UserStackSummary(ctx.Request.Context(), user.ID)
	ctx.JSON(http.StatusOK, loginResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: newUserMeResponse(user, stackCount, stackLimit)})
}

func (h *Handler) Refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	accessToken, refreshToken, err := h.auth.Refresh(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, refreshResponse{AccessToken: accessToken, RefreshToken: refreshToken})
}

func (h *Handler) Logout(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	if err := h.auth.Logout(ctx.Request.Context(), req.RefreshToken); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Me(ctx *gin.Context) {
	user, err := h.users.GetByID(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}
	stackCount, stackLimit, _ := h.stacks.UserStackSummary(ctx.Request.Context(), user.ID)
	ctx.JSON(http.StatusOK, newUserMeResponse(user, stackCount, stackLimit))
}

func (h *Handler) UpdateMe(ctx *gin.Context) {
	var req meUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	user, err := h.users.UpdateProfile(ctx.Request.Context(), middleware.UserID(ctx), req.Username)
	if err != nil {
		writeError(ctx, err)
		return
	}
	stackCount, stackLimit, _ := h.stacks.UserStackSummary(ctx.Request.Context(), user.ID)
	ctx.JSON(http.StatusOK, newUserMeResponse(user, stackCount, stackLimit))
}

func (h *Handler) ListChallenges(ctx *gin.Context) {
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}
	filters, ok := parseChallengeFilters(ctx)
	if !ok {
		return
	}

	userID := h.optionalUserID(ctx)
	challenges, pagination, err := h.wargame.ListChallenges(ctx.Request.Context(), page, pageSize, service.ChallengeQueryFilter{
		Category: filters.Category,
		Level:    filters.Level,
		Solved:   filters.Solved,
		UserID:   userID,
	})
	if err != nil {
		writeError(ctx, err)
		return
	}

	solved := map[int64]struct{}{}
	if userID > 0 {
		solved, err = h.wargame.SolvedChallengeIDs(ctx.Request.Context(), userID)
		if err != nil {
			writeError(ctx, err)
			return
		}
	}

	resp := make([]any, 0, len(challenges))
	byID := make(map[int64]*models.Challenge, len(challenges))
	for i := range challenges {
		byID[challenges[i].ID] = &challenges[i]
	}
	for _, challenge := range challenges {
		ch := challenge
		_, isSolved := solved[ch.ID]
		if isChallengeLocked(ch, solved, userID) {
			previous := h.previousChallengeForResponse(ctx.Request.Context(), byID, ch.PreviousChallengeID)
			resp = append(resp, newLockedChallengeResponse(&ch, previous, isSolved))
			continue
		}
		resp = append(resp, newChallengeResponse(&ch, isSolved))
	}

	ctx.JSON(http.StatusOK, challengesListResponse{Challenges: resp, Pagination: pagination})
}

func (h *Handler) SearchChallenges(ctx *gin.Context) {
	query, ok := parseSearchQuery(ctx)
	if !ok {
		return
	}
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}
	filters, ok := parseChallengeFilters(ctx)
	if !ok {
		return
	}

	userID := h.optionalUserID(ctx)
	challenges, pagination, err := h.wargame.SearchChallenges(ctx.Request.Context(), query, page, pageSize, service.ChallengeQueryFilter{
		Query:    query,
		Category: filters.Category,
		Level:    filters.Level,
		Solved:   filters.Solved,
		UserID:   userID,
	})
	if err != nil {
		writeError(ctx, err)
		return
	}

	solved := map[int64]struct{}{}
	if userID > 0 {
		solved, err = h.wargame.SolvedChallengeIDs(ctx.Request.Context(), userID)
		if err != nil {
			writeError(ctx, err)
			return
		}
	}

	resp := make([]any, 0, len(challenges))
	byID := make(map[int64]*models.Challenge, len(challenges))
	for i := range challenges {
		byID[challenges[i].ID] = &challenges[i]
	}
	for _, challenge := range challenges {
		ch := challenge
		_, isSolved := solved[ch.ID]
		if isChallengeLocked(ch, solved, userID) {
			previous := h.previousChallengeForResponse(ctx.Request.Context(), byID, ch.PreviousChallengeID)
			resp = append(resp, newLockedChallengeResponse(&ch, previous, isSolved))
			continue
		}
		resp = append(resp, newChallengeResponse(&ch, isSolved))
	}

	ctx.JSON(http.StatusOK, challengesListResponse{Challenges: resp, Pagination: pagination})
}

func (h *Handler) GetChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	challenge, err := h.wargame.GetChallengeByID(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	userID := h.optionalUserID(ctx)
	solved := map[int64]struct{}{}
	if userID > 0 {
		solved, err = h.wargame.SolvedChallengeIDs(ctx.Request.Context(), userID)
		if err != nil {
			writeError(ctx, err)
			return
		}
	}
	_, isSolved := solved[challenge.ID]

	if isChallengeLocked(*challenge, solved, userID) {
		previous := h.previousChallengeForResponse(ctx.Request.Context(), map[int64]*models.Challenge{}, challenge.PreviousChallengeID)
		ctx.JSON(http.StatusOK, newLockedChallengeResponse(challenge, previous, isSolved))
		return
	}

	ctx.JSON(http.StatusOK, newChallengeResponse(challenge, isSolved))
}

func (h *Handler) ChallengeSolvers(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}

	rows, pagination, err := h.wargame.ChallengeSolversPage(ctx.Request.Context(), challengeID, page, pageSize)
	if err != nil {
		writeError(ctx, err)
		return
	}

	solvers := make([]challengeSolverResponse, 0, len(rows))
	for _, row := range rows {
		solvers = append(solvers, challengeSolverResponse{
			UserID:       row.UserID,
			Username:     row.Username,
			SolvedAt:     row.SolvedAt.UTC(),
			IsFirstBlood: row.IsFirstBlood,
		})
	}

	ctx.JSON(http.StatusOK, challengeSolversResponse{Solvers: solvers, Pagination: pagination})
}

func (h *Handler) SubmitFlag(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	var req submitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	correct, err := h.wargame.SubmitFlag(ctx.Request.Context(), middleware.UserID(ctx), challengeID, req.Flag)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if correct {
		h.notifyScoreboardChanged(ctx.Request.Context(), "submission_correct")
		if h.stacks != nil {
			_ = h.stacks.DeleteStackByUserAndChallenge(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"correct": correct})
}

func (h *Handler) CreateStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	stackModel, err := h.stacks.GetOrCreateStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, newStackResponse(stackModel))
}

func (h *Handler) GetStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	stackModel, err := h.stacks.GetStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newStackResponse(stackModel))
}

func (h *Handler) DeleteStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	if err := h.stacks.DeleteStack(ctx.Request.Context(), middleware.UserID(ctx), challengeID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListStacks(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	stacks, err := h.stacks.ListUserStacks(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}
	resp := make([]stackResponse, 0, len(stacks))
	for i := range stacks {
		stackModel := stacks[i]
		resp = append(resp, newStackResponse(&stackModel))
	}
	ctx.JSON(http.StatusOK, stacksListResponse{Stacks: resp})
}

func (h *Handler) AdminListStacks(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	stacks, err := h.stacks.ListAdminStacks(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	resp := make([]adminStackResponse, 0, len(stacks))
	for i := range stacks {
		resp = append(resp, newAdminStackResponse(stacks[i]))
	}
	ctx.JSON(http.StatusOK, adminStacksListResponse{Stacks: resp})
}

func (h *Handler) AdminDeleteStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	stackID := strings.TrimSpace(ctx.Param("stack_id"))
	if stackID == "" {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "stack_id", Reason: "required"}))
		return
	}
	if err := h.stacks.DeleteStackByStackID(ctx.Request.Context(), stackID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"deleted": true, "stack_id": stackID})
}

func (h *Handler) AdminGetStack(ctx *gin.Context) {
	if h.stacks == nil {
		writeError(ctx, service.ErrStackDisabled)
		return
	}
	stackID := strings.TrimSpace(ctx.Param("stack_id"))
	if stackID == "" {
		writeError(ctx, service.NewValidationError(service.FieldError{Field: "stack_id", Reason: "required"}))
		return
	}
	stackModel, err := h.stacks.GetStackByStackID(ctx.Request.Context(), stackID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newStackResponse(stackModel))
}

func (h *Handler) CreateChallenge(ctx *gin.Context) {
	var req createChallengeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	stackEnabled := false
	if req.StackEnabled != nil {
		stackEnabled = *req.StackEnabled
	}
	challenge, err := h.wargame.CreateChallenge(ctx.Request.Context(), req.Title, req.Description, req.Category, req.Level, req.Points, req.Flag, active, stackEnabled, stack.TargetPortSpecs(req.StackTargetPorts), req.StackPodSpec, req.PreviousChallengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_created")
	ctx.JSON(http.StatusCreated, newChallengeResponse(challenge, false))
}

func (h *Handler) UpdateChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	var req updateChallengeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	title, err := requireNonNullOptionalString("title", req.Title)
	if err != nil {
		writeError(ctx, err)
		return
	}
	description, err := requireNonNullOptionalString("description", req.Description)
	if err != nil {
		writeError(ctx, err)
		return
	}
	category, err := requireNonNullOptionalString("category", req.Category)
	if err != nil {
		writeError(ctx, err)
		return
	}
	flag, err := requireNonNullOptionalString("flag", req.Flag)
	if err != nil {
		writeError(ctx, err)
		return
	}
	stackPodSpec := optionalStringToPointer(req.StackPodSpec)
	previousChallengeID := (*int64)(nil)
	previousChallengeSet := req.PreviousChallengeID.Set
	if previousChallengeSet {
		previousChallengeID = req.PreviousChallengeID.Value
	}

	challenge, err := h.wargame.UpdateChallenge(ctx.Request.Context(), challengeID, title, description, category, req.Level, req.Points, flag, req.IsActive, req.StackEnabled, req.StackTargetPorts, stackPodSpec, previousChallengeID, previousChallengeSet)
	if err != nil {
		writeError(ctx, err)
		return
	}

	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_updated")
	ctx.JSON(http.StatusOK, newChallengeResponse(challenge, false))
}

func requireNonNullOptionalString(field string, value optionalString) (*string, error) {
	if !value.Set {
		return nil, nil
	}
	if value.Value == nil {
		return nil, service.NewValidationError(service.FieldError{Field: field, Reason: "invalid"})
	}
	return value.Value, nil
}

func optionalStringToPointer(value optionalString) *string {
	if !value.Set {
		return nil
	}
	if value.Value == nil {
		empty := ""
		return &empty
	}
	return value.Value
}

func (h *Handler) AdminGetChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	challenge, err := h.wargame.GetChallengeByID(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, adminChallengeResponse{challengeResponse: newChallengeResponse(challenge, false), StackPodSpec: challenge.StackPodSpec})
}

func (h *Handler) DeleteChallenge(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	if err := h.wargame.DeleteChallenge(ctx.Request.Context(), challengeID); err != nil {
		writeError(ctx, err)
		return
	}
	h.notifyScoreboardChanged(ctx.Request.Context(), "challenge_deleted")
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) RequestChallengeFileUpload(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	var req challengeFileUploadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	challenge, upload, err := h.wargame.RequestChallengeFileUpload(ctx.Request.Context(), challengeID, req.Filename)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, challengeFileUploadResponse{Challenge: newChallengeResponse(challenge, false), Upload: presignedPostResponse{URL: upload.URL, Fields: upload.Fields, ExpiresAt: upload.ExpiresAt}})
}

func (h *Handler) RequestChallengeFileDownload(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	download, err := h.wargame.RequestChallengeFileDownload(ctx.Request.Context(), middleware.UserID(ctx), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, presignedURLResponse{URL: download.URL, ExpiresAt: download.ExpiresAt})
}

func (h *Handler) DeleteChallengeFile(ctx *gin.Context) {
	challengeID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	challenge, err := h.wargame.DeleteChallengeFile(ctx.Request.Context(), challengeID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newChallengeResponse(challenge, false))
}

func (h *Handler) Leaderboard(ctx *gin.Context) {
	cacheKey := "leaderboard:users"
	if h.respondFromCache(ctx, cacheKey) {
		return
	}
	rows, err := h.score.Leaderboard(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	h.storeCache(ctx, cacheKey, rows, h.cfg.Cache.LeaderboardTTL)
	ctx.JSON(http.StatusOK, rows)
}

func (h *Handler) Timeline(ctx *gin.Context) {
	cacheKey := "timeline:users"
	if h.respondFromCache(ctx, cacheKey) {
		return
	}
	submissions, err := h.score.UserTimeline(ctx.Request.Context(), nil)
	if err != nil {
		writeError(ctx, err)
		return
	}
	response := timelineResponse{Submissions: submissions}
	h.storeCache(ctx, cacheKey, response, h.cfg.Cache.TimelineTTL)
	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) ListUsers(ctx *gin.Context) {
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}

	users, pagination, err := h.users.ListPage(ctx.Request.Context(), page, pageSize)
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]userDetailResponse, 0, len(users))
	for _, user := range users {
		u := user
		resp = append(resp, newUserDetailResponse(&u))
	}

	ctx.JSON(http.StatusOK, usersListResponse{Users: resp, Pagination: pagination})
}

func (h *Handler) SearchUsers(ctx *gin.Context) {
	query, ok := parseSearchQuery(ctx)
	if !ok {
		return
	}
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}

	users, pagination, err := h.users.Search(ctx.Request.Context(), query, page, pageSize)
	if err != nil {
		writeError(ctx, err)
		return
	}
	resp := make([]userDetailResponse, 0, len(users))
	for _, user := range users {
		u := user
		resp = append(resp, newUserDetailResponse(&u))
	}
	ctx.JSON(http.StatusOK, usersListResponse{Users: resp, Pagination: pagination})
}

func (h *Handler) GetUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	user, err := h.users.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newUserDetailResponse(user))
}

func (h *Handler) GetUserSolved(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}
	if _, err := h.users.GetByID(ctx.Request.Context(), userID); err != nil {
		writeError(ctx, err)
		return
	}
	rows, pagination, err := h.wargame.SolvedChallengesPage(ctx.Request.Context(), userID, page, pageSize)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, userSolvedListResponse{Solved: rows, Pagination: pagination})
}

func (h *Handler) AdminBlockUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	var req adminBlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}
	updated, err := h.users.BlockUser(ctx.Request.Context(), userID, req.Reason)
	if err != nil {
		writeError(ctx, err)
		return
	}
	h.notifyScoreboardChanged(ctx.Request.Context(), "user_blocked")
	ctx.JSON(http.StatusOK, newAdminUserResponse(updated))
}

func (h *Handler) AdminUnblockUser(ctx *gin.Context) {
	userID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}
	updated, err := h.users.UnblockUser(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	h.notifyScoreboardChanged(ctx.Request.Context(), "user_unblocked")
	ctx.JSON(http.StatusOK, newAdminUserResponse(updated))
}
