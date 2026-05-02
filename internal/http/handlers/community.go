package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"wargame/internal/http/middleware"
	"wargame/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListCommunityPosts(ctx *gin.Context) {
	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}

	query := strings.TrimSpace(ctx.Query("q"))
	sort := strings.TrimSpace(ctx.Query("sort"))
	excludeNotice := strings.EqualFold(strings.TrimSpace(ctx.Query("exclude_notice")), "true") || strings.TrimSpace(ctx.Query("exclude_notice")) == "1"
	popularOnly := strings.EqualFold(strings.TrimSpace(ctx.Query("popular_only")), "true") || strings.TrimSpace(ctx.Query("popular_only")) == "1"

	var category *int
	if categoryRaw := strings.TrimSpace(ctx.Query("category")); categoryRaw != "" {
		parsed, err := strconv.Atoi(categoryRaw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error(), Details: []service.FieldError{{Field: "category", Reason: "invalid"}}})
			return
		}

		category = &parsed
	}

	rows, pagination, err := h.wargame.CommunityPostsPage(ctx.Request.Context(), page, pageSize, query, category, excludeNotice, popularOnly, sort, middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]communityPostResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, newCommunityPostResponse(rows[i]))
	}

	ctx.JSON(http.StatusOK, communityPostsListResponse{Posts: resp, Pagination: pagination})
}

func (h *Handler) GetCommunityPost(ctx *gin.Context) {
	postID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	row, err := h.wargame.CommunityPostByID(ctx.Request.Context(), postID, middleware.UserID(ctx), true)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, communityPostDetailResponse{Post: newCommunityPostResponse(*row)})
}

func (h *Handler) CreateCommunityPost(ctx *gin.Context) {
	var req createCommunityPostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	row, err := h.wargame.CreateCommunityPost(ctx.Request.Context(), middleware.UserID(ctx), middleware.Role(ctx), req.Category, req.Title, req.Content)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newCommunityPostResponse(*row))
}

func (h *Handler) UpdateCommunityPost(ctx *gin.Context) {
	postID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	var req updateCommunityPostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeBindError(ctx, err)
		return
	}

	title, err := requireNonNullOptionalString("title", req.Title)
	if err != nil {
		writeError(ctx, err)
		return
	}

	content, err := requireNonNullOptionalString("content", req.Content)
	if err != nil {
		writeError(ctx, err)
		return
	}

	row, svcErr := h.wargame.UpdateCommunityPost(ctx.Request.Context(), middleware.UserID(ctx), middleware.Role(ctx), postID, req.Category, title, content)
	if svcErr != nil {
		writeError(ctx, svcErr)
		return
	}

	ctx.JSON(http.StatusOK, newCommunityPostResponse(*row))
}

func (h *Handler) DeleteCommunityPost(ctx *gin.Context) {
	postID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	if err := h.wargame.DeleteCommunityPost(ctx.Request.Context(), middleware.UserID(ctx), middleware.Role(ctx), postID); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ToggleCommunityPostLike(ctx *gin.Context) {
	postID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	liked, likeCount, err := h.wargame.ToggleCommunityPostLike(ctx.Request.Context(), middleware.UserID(ctx), postID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, communityLikeToggleResponse{Status: "ok", Liked: liked, LikeCount: likeCount})
}

func (h *Handler) CommunityPostLikes(ctx *gin.Context) {
	postID, ok := parseIDParamOrError(ctx, "id")
	if !ok {
		return
	}

	page, pageSize, ok := parsePaginationParams(ctx)
	if !ok {
		return
	}

	rows, pagination, err := h.wargame.CommunityPostLikesPage(ctx.Request.Context(), postID, page, pageSize)
	if err != nil {
		writeError(ctx, err)
		return
	}

	resp := make([]communityPostLikeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, newCommunityPostLikeResponse(rows[i]))
	}

	ctx.JSON(http.StatusOK, communityPostLikesListResponse{Likes: resp, Pagination: pagination})
}
