package handlers

import (
	"encoding/json"
	"time"

	"wargame/internal/models"
	stackpkg "wargame/internal/stack"
)

type optionalString struct {
	Set   bool
	Value *string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type optionalInt64 struct {
	Set   bool
	Value *int64
}

func (o *optionalInt64) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type meUpdateRequest struct {
	Username      *string        `json:"username"`
	AffiliationID optionalInt64  `json:"affiliation_id"`
	Bio           optionalString `json:"bio"`
}

type adminAffiliationCreateRequest struct {
	Name string `json:"name" binding:"required"`
}

type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type createChallengeRequest struct {
	Title               string                    `json:"title" binding:"required"`
	Description         string                    `json:"description" binding:"required"`
	Category            string                    `json:"category" binding:"required"`
	Points              int                       `json:"points" binding:"required"`
	Flag                string                    `json:"flag" binding:"required"`
	PreviousChallengeID *int64                    `json:"previous_challenge_id"`
	IsActive            *bool                     `json:"is_active"`
	StackEnabled        *bool                     `json:"stack_enabled"`
	StackTargetPorts    []stackpkg.TargetPortSpec `json:"stack_target_ports"`
	StackPodSpec        *string                   `json:"stack_pod_spec"`
}

type updateChallengeRequest struct {
	Title               optionalString             `json:"title"`
	Description         optionalString             `json:"description"`
	Category            optionalString             `json:"category"`
	Points              *int                       `json:"points"`
	Flag                optionalString             `json:"flag"`
	PreviousChallengeID optionalInt64              `json:"previous_challenge_id"`
	IsActive            *bool                      `json:"is_active"`
	StackEnabled        *bool                      `json:"stack_enabled"`
	StackTargetPorts    *[]stackpkg.TargetPortSpec `json:"stack_target_ports"`
	StackPodSpec        optionalString             `json:"stack_pod_spec"`
}

type challengeFileUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
}

type submitRequest struct {
	Flag string `json:"flag" binding:"required"`
}

type levelVoteRequest struct {
	Level int `json:"level" binding:"required"`
}

type createWriteupRequest struct {
	Content string `json:"content" binding:"required"`
}

type updateWriteupRequest struct {
	Content optionalString `json:"content"`
}

type createChallengeCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type updateChallengeCommentRequest struct {
	Content optionalString `json:"content"`
}

type createCommunityPostRequest struct {
	Category int    `json:"category"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

type updateCommunityPostRequest struct {
	Category *int           `json:"category"`
	Title    optionalString `json:"title"`
	Content  optionalString `json:"content"`
}

type createCommunityCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

type updateCommunityCommentRequest struct {
	Content optionalString `json:"content"`
}

type adminBlockUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type registerResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type loginUserResponse = userMeResponse

type loginResponse struct {
	User loginUserResponse `json:"user"`
}

type refreshResponse struct {
	Status string `json:"status"`
}

type userMeResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	AffiliationID *int64     `json:"affiliation_id"`
	Affiliation   *string    `json:"affiliation"`
	Bio           *string    `json:"bio"`
	StackCount    int        `json:"stack_count"`
	StackLimit    int        `json:"stack_limit"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type userDetailResponse struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	AffiliationID *int64     `json:"affiliation_id"`
	Affiliation   *string    `json:"affiliation"`
	Bio           *string    `json:"bio"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type adminUserResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	AffiliationID *int64     `json:"affiliation_id"`
	Affiliation   *string    `json:"affiliation"`
	Bio           *string    `json:"bio"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type challengeResponse struct {
	ID                  int64                     `json:"id"`
	Title               string                    `json:"title"`
	Description         string                    `json:"description"`
	Category            string                    `json:"category"`
	CreatedAt           time.Time                 `json:"created_at"`
	Level               int                       `json:"level"`
	Points              int                       `json:"points"`
	SolveCount          int                       `json:"solve_count"`
	LevelVoteCounts     []models.LevelVoteCount   `json:"level_vote_counts,omitempty"`
	CreatedBy           *challengeCreatorResponse `json:"created_by,omitempty"`
	FirstBlood          *challengeSolverResponse  `json:"first_blood,omitempty"`
	PreviousChallengeID *int64                    `json:"previous_challenge_id,omitempty"`
	IsActive            bool                      `json:"is_active"`
	IsLocked            bool                      `json:"is_locked"`
	IsSolved            bool                      `json:"is_solved"`
	HasFile             bool                      `json:"has_file"`
	FileName            *string                   `json:"file_name,omitempty"`
	StackEnabled        bool                      `json:"stack_enabled"`
	StackTargetPorts    []stackpkg.TargetPortSpec `json:"stack_target_ports"`
}

type lockedChallengeResponse struct {
	ID                        int64                     `json:"id"`
	Title                     string                    `json:"title"`
	Category                  string                    `json:"category"`
	CreatedAt                 time.Time                 `json:"created_at"`
	Level                     int                       `json:"level"`
	Points                    int                       `json:"points"`
	SolveCount                int                       `json:"solve_count"`
	CreatedBy                 *challengeCreatorResponse `json:"created_by,omitempty"`
	FirstBlood                *challengeSolverResponse  `json:"first_blood,omitempty"`
	PreviousChallengeID       *int64                    `json:"previous_challenge_id,omitempty"`
	PreviousChallengeTitle    *string                   `json:"previous_challenge_title,omitempty"`
	PreviousChallengeCategory *string                   `json:"previous_challenge_category,omitempty"`
	IsActive                  bool                      `json:"is_active"`
	IsLocked                  bool                      `json:"is_locked"`
	IsSolved                  bool                      `json:"is_solved"`
}

type challengeCreatorResponse struct {
	UserID        *int64  `json:"user_id,omitempty"`
	Username      string  `json:"username,omitempty"`
	AffiliationID *int64  `json:"affiliation_id,omitempty"`
	Affiliation   *string `json:"affiliation,omitempty"`
	Bio           *string `json:"bio,omitempty"`
}

type challengeVotesResponse struct {
	Votes      []models.ChallengeVoteDetail `json:"votes,omitempty"`
	Pagination models.Pagination            `json:"pagination"`
}

type challengeMyVoteResponse struct {
	Level *int `json:"level"`
}

type challengesListResponse struct {
	Challenges []any             `json:"challenges,omitempty"`
	Pagination models.Pagination `json:"pagination"`
}

type usersListResponse struct {
	Users      []userDetailResponse `json:"users,omitempty"`
	Pagination models.Pagination    `json:"pagination"`
}

type userSolvedListResponse struct {
	Solved     []models.SolvedChallenge `json:"solved,omitempty"`
	Pagination models.Pagination        `json:"pagination"`
}

type challengeSolverResponse struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Affiliation  *string   `json:"affiliation,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	SolvedAt     time.Time `json:"solved_at"`
	IsFirstBlood bool      `json:"is_first_blood"`
}

type challengeSolversResponse struct {
	Solvers    []challengeSolverResponse `json:"solvers,omitempty"`
	Pagination models.Pagination         `json:"pagination"`
}

type writeupAuthorResponse struct {
	UserID        int64   `json:"user_id"`
	Username      string  `json:"username"`
	AffiliationID *int64  `json:"affiliation_id,omitempty"`
	Affiliation   *string `json:"affiliation,omitempty"`
	Bio           *string `json:"bio,omitempty"`
}

type writeupChallengeResponse struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Points   int    `json:"points"`
	Level    int    `json:"level"`
}

type writeupResponse struct {
	ID        int64                    `json:"id"`
	Content   *string                  `json:"content,omitempty"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	Author    writeupAuthorResponse    `json:"author"`
	Challenge writeupChallengeResponse `json:"challenge"`
}

type writeupsListResponse struct {
	Writeups       []writeupResponse `json:"writeups,omitempty"`
	CanViewContent bool              `json:"can_view_content"`
	Pagination     models.Pagination `json:"pagination"`
}

type writeupDetailResponse struct {
	Writeup        writeupResponse `json:"writeup"`
	CanViewContent bool            `json:"can_view_content"`
}

type commentAuthorResponse struct {
	UserID        int64   `json:"user_id"`
	Username      string  `json:"username"`
	AffiliationID *int64  `json:"affiliation_id,omitempty"`
	Affiliation   *string `json:"affiliation,omitempty"`
	Bio           *string `json:"bio,omitempty"`
}

type commentChallengeResponse struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type challengeCommentResponse struct {
	ID        int64                    `json:"id"`
	Content   string                   `json:"content"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	Author    commentAuthorResponse    `json:"author"`
	Challenge commentChallengeResponse `json:"challenge"`
}

type challengeCommentsListResponse struct {
	Comments   []challengeCommentResponse `json:"comments,omitempty"`
	Pagination models.Pagination          `json:"pagination"`
}

type communityPostAuthorResponse struct {
	UserID        int64   `json:"user_id"`
	Username      string  `json:"username"`
	AffiliationID *int64  `json:"affiliation_id,omitempty"`
	Affiliation   *string `json:"affiliation,omitempty"`
	Bio           *string `json:"bio,omitempty"`
}

type communityPostResponse struct {
	ID           int64                       `json:"id"`
	Category     int                         `json:"category"`
	Title        string                      `json:"title"`
	Content      string                      `json:"content"`
	ViewCount    int                         `json:"view_count"`
	LikeCount    int                         `json:"like_count"`
	CommentCount int                         `json:"comment_count"`
	LikedByMe    bool                        `json:"liked_by_me"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Author       communityPostAuthorResponse `json:"author"`
}

type communityPostsListResponse struct {
	Posts      []communityPostResponse `json:"posts,omitempty"`
	Pagination models.Pagination       `json:"pagination"`
}

type communityPostDetailResponse struct {
	Post communityPostResponse `json:"post"`
}

type communityPostLikeResponse struct {
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	AffiliationID *int64    `json:"affiliation_id,omitempty"`
	Affiliation   *string   `json:"affiliation,omitempty"`
	Bio           *string   `json:"bio,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type communityPostLikesListResponse struct {
	Likes      []communityPostLikeResponse `json:"likes,omitempty"`
	Pagination models.Pagination           `json:"pagination"`
}

type communityCommentResponse struct {
	ID        int64                 `json:"id"`
	Content   string                `json:"content"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Author    commentAuthorResponse `json:"author"`
	Post      struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	} `json:"post"`
}

type communityCommentsListResponse struct {
	Comments   []communityCommentResponse `json:"comments,omitempty"`
	Pagination models.Pagination          `json:"pagination"`
}

type communityLikeToggleResponse struct {
	Status    string `json:"status"`
	Liked     bool   `json:"liked"`
	LikeCount int    `json:"like_count"`
}

type adminChallengeResponse struct {
	challengeResponse
	StackPodSpec *string `json:"stack_pod_spec,omitempty"`
}

type presignedPostResponse struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type presignedURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type challengeFileUploadResponse struct {
	Challenge challengeResponse     `json:"challenge"`
	Upload    presignedPostResponse `json:"upload"`
}

type timelineResponse struct {
	Submissions []models.TimelineSubmission `json:"submissions"`
}

type affiliationResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type affiliationsListResponse struct {
	Affiliations []affiliationResponse `json:"affiliations,omitempty"`
	Pagination   models.Pagination     `json:"pagination"`
}

type leaderboardListResponse struct {
	Challenges []models.LeaderboardChallenge `json:"challenges"`
	Entries    []models.LeaderboardEntry     `json:"entries"`
	Pagination models.Pagination             `json:"pagination"`
}

type userRankingListResponse struct {
	Entries    []models.UserRankingEntry `json:"entries"`
	Pagination models.Pagination         `json:"pagination"`
}

type affiliationRankingListResponse struct {
	Entries    []models.AffiliationRankingEntry `json:"entries"`
	Pagination models.Pagination                `json:"pagination"`
}

type stackResponse struct {
	StackID           string                 `json:"stack_id"`
	ChallengeID       int64                  `json:"challenge_id"`
	Status            string                 `json:"status"`
	NodePublicIP      *string                `json:"node_public_ip,omitempty"`
	Ports             []stackpkg.PortMapping `json:"ports,omitempty"`
	TTLExpiresAt      *time.Time             `json:"ttl_expires_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	CreatedByUserID   int64                  `json:"created_by_user_id"`
	CreatedByUsername string                 `json:"created_by_username"`
	ChallengeTitle    string                 `json:"challenge_title"`
}

type stacksListResponse struct {
	Stacks []stackResponse `json:"stacks,omitempty"`
}

type adminStackResponse struct {
	StackID           string     `json:"stack_id"`
	TTLExpiresAt      *time.Time `json:"ttl_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UserID            int64      `json:"user_id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	ChallengeID       int64      `json:"challenge_id"`
	ChallengeTitle    string     `json:"challenge_title"`
	ChallengeCategory string     `json:"challenge_category"`
}

type adminStacksListResponse struct {
	Stacks []adminStackResponse `json:"stacks,omitempty"`
}

func newStackResponse(stack *models.Stack) stackResponse {
	return stackResponse{StackID: stack.StackID, ChallengeID: stack.ChallengeID, Status: stack.Status, NodePublicIP: stack.NodePublicIP, Ports: []stackpkg.PortMapping(stack.Ports), TTLExpiresAt: stack.TTLExpiresAt, CreatedAt: stack.CreatedAt.UTC(), UpdatedAt: stack.UpdatedAt.UTC(), CreatedByUserID: stack.UserID, CreatedByUsername: stack.Username, ChallengeTitle: stack.ChallengeTitle}
}

func newAdminStackResponse(stack models.AdminStackSummary) adminStackResponse {
	return adminStackResponse{StackID: stack.StackID, TTLExpiresAt: timePtrUTC(stack.TTLExpiresAt), CreatedAt: stack.CreatedAt.UTC(), UpdatedAt: stack.UpdatedAt.UTC(), UserID: stack.UserID, Username: stack.Username, Email: stack.Email, ChallengeID: stack.ChallengeID, ChallengeTitle: stack.ChallengeTitle, ChallengeCategory: stack.ChallengeCategory}
}

func timePtrUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func newUserMeResponse(user *models.User, stackCount, stackLimit int) userMeResponse {
	return userMeResponse{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		AffiliationID: user.AffiliationID,
		Affiliation:   user.Affiliation,
		Bio:           user.Bio,
		StackCount:    stackCount,
		StackLimit:    stackLimit,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newUserDetailResponse(user *models.User) userDetailResponse {
	return userDetailResponse{
		ID:            user.ID,
		Username:      user.Username,
		Role:          user.Role,
		AffiliationID: user.AffiliationID,
		Affiliation:   user.Affiliation,
		Bio:           user.Bio,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newAdminUserResponse(user *models.User) adminUserResponse {
	return adminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Role:          user.Role,
		AffiliationID: user.AffiliationID,
		Affiliation:   user.Affiliation,
		Bio:           user.Bio,
		BlockedReason: user.BlockedReason,
		BlockedAt:     user.BlockedAt,
	}
}

func newChallengeResponse(challenge *models.Challenge, isSolved bool, firstBlood *models.ChallengeSolver) challengeResponse {
	hasFile := challenge.FileKey != nil && *challenge.FileKey != ""
	firstBloodResp := (*challengeSolverResponse)(nil)
	if firstBlood != nil {
		mapped := newChallengeSolverResponse(*firstBlood)
		firstBloodResp = &mapped
	}

	return challengeResponse{
		ID:                  challenge.ID,
		Title:               challenge.Title,
		Description:         challenge.Description,
		Category:            challenge.Category,
		CreatedAt:           challenge.CreatedAt.UTC(),
		Level:               challenge.Level,
		Points:              challenge.Points,
		SolveCount:          challenge.SolveCount,
		LevelVoteCounts:     challenge.LevelVotes,
		CreatedBy:           newChallengeCreatorResponse(challenge),
		FirstBlood:          firstBloodResp,
		PreviousChallengeID: challenge.PreviousChallengeID,
		IsActive:            challenge.IsActive,
		IsLocked:            false,
		IsSolved:            isSolved,
		HasFile:             hasFile,
		FileName:            challenge.FileName,
		StackEnabled:        challenge.StackEnabled,
		StackTargetPorts:    []stackpkg.TargetPortSpec(challenge.StackTargetPorts),
	}
}

func newLockedChallengeResponse(challenge *models.Challenge, previous *models.Challenge, isSolved bool, firstBlood *models.ChallengeSolver) lockedChallengeResponse {
	var prevTitle *string
	var prevCategory *string
	firstBloodResp := (*challengeSolverResponse)(nil)
	if previous != nil {
		prevTitle = &previous.Title
		prevCategory = &previous.Category
	}
	if firstBlood != nil {
		mapped := newChallengeSolverResponse(*firstBlood)
		firstBloodResp = &mapped
	}
	return lockedChallengeResponse{
		ID:                        challenge.ID,
		Title:                     challenge.Title,
		Category:                  challenge.Category,
		CreatedAt:                 challenge.CreatedAt.UTC(),
		Level:                     challenge.Level,
		Points:                    challenge.Points,
		SolveCount:                challenge.SolveCount,
		CreatedBy:                 newChallengeCreatorResponse(challenge),
		FirstBlood:                firstBloodResp,
		PreviousChallengeID:       challenge.PreviousChallengeID,
		PreviousChallengeTitle:    prevTitle,
		PreviousChallengeCategory: prevCategory,
		IsActive:                  challenge.IsActive,
		IsLocked:                  true,
		IsSolved:                  isSolved,
	}
}

func newChallengeCreatorResponse(challenge *models.Challenge) *challengeCreatorResponse {
	if challenge.CreatedByUserID == nil && challenge.CreatedByUsername == "" && challenge.CreatedByAffiliationID == nil && challenge.CreatedByAffiliation == nil && challenge.CreatedByBio == nil {
		return nil
	}

	return &challengeCreatorResponse{
		UserID:        challenge.CreatedByUserID,
		Username:      challenge.CreatedByUsername,
		AffiliationID: challenge.CreatedByAffiliationID,
		Affiliation:   challenge.CreatedByAffiliation,
		Bio:           challenge.CreatedByBio,
	}
}

func newChallengeSolverResponse(row models.ChallengeSolver) challengeSolverResponse {
	return challengeSolverResponse{
		UserID:       row.UserID,
		Username:     row.Username,
		Affiliation:  row.Affiliation,
		Bio:          row.Bio,
		SolvedAt:     row.SolvedAt.UTC(),
		IsFirstBlood: row.IsFirstBlood,
	}
}

func newWriteupResponse(row models.WriteupDetail, includeContent bool) writeupResponse {
	content := (*string)(nil)
	if includeContent {
		value := row.Content
		content = &value
	}

	return writeupResponse{
		ID:        row.ID,
		Content:   content,
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
		Author: writeupAuthorResponse{
			UserID:        row.UserID,
			Username:      row.Username,
			AffiliationID: row.AffiliationID,
			Affiliation:   row.Affiliation,
			Bio:           row.Bio,
		},
		Challenge: writeupChallengeResponse{
			ID:       row.ChallengeID,
			Title:    row.ChallengeTitle,
			Category: row.ChallengeCategory,
			Points:   row.ChallengePoints,
			Level:    row.ChallengeLevel,
		},
	}
}

func newChallengeCommentResponse(row models.ChallengeCommentDetail) challengeCommentResponse {
	return challengeCommentResponse{
		ID:        row.ID,
		Content:   row.Content,
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
		Author: commentAuthorResponse{
			UserID:        row.UserID,
			Username:      row.Username,
			AffiliationID: row.AffiliationID,
			Affiliation:   row.Affiliation,
			Bio:           row.Bio,
		},
		Challenge: commentChallengeResponse{
			ID:    row.ChallengeID,
			Title: row.ChallengeTitle,
		},
	}
}

func newCommunityPostResponse(row models.CommunityPostDetail) communityPostResponse {
	return communityPostResponse{
		ID:           row.ID,
		Category:     row.Category,
		Title:        row.Title,
		Content:      row.Content,
		ViewCount:    row.ViewCount,
		LikeCount:    row.LikeCount,
		CommentCount: row.CommentCount,
		LikedByMe:    row.LikedByMe,
		CreatedAt:    row.CreatedAt.UTC(),
		UpdatedAt:    row.UpdatedAt.UTC(),
		Author: communityPostAuthorResponse{
			UserID:        row.UserID,
			Username:      row.Username,
			AffiliationID: row.AffiliationID,
			Affiliation:   row.Affiliation,
			Bio:           row.Bio,
		},
	}
}

func newCommunityCommentResponse(row models.CommunityCommentDetail) communityCommentResponse {
	resp := communityCommentResponse{
		ID:        row.ID,
		Content:   row.Content,
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
		Author: commentAuthorResponse{
			UserID:        row.UserID,
			Username:      row.Username,
			AffiliationID: row.AffiliationID,
			Affiliation:   row.Affiliation,
			Bio:           row.Bio,
		},
	}
	resp.Post.ID = row.PostID
	resp.Post.Title = row.PostTitle
	return resp
}

func newCommunityPostLikeResponse(row models.CommunityPostLikeDetail) communityPostLikeResponse {
	return communityPostLikeResponse{
		UserID:        row.UserID,
		Username:      row.Username,
		AffiliationID: row.AffiliationID,
		Affiliation:   row.Affiliation,
		Bio:           row.Bio,
		CreatedAt:     row.CreatedAt.UTC(),
	}
}
