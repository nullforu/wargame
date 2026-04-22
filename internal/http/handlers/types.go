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
	Username *string `json:"username"`
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	User         loginUserResponse `json:"user"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type userMeResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	StackCount    int        `json:"stack_count"`
	StackLimit    int        `json:"stack_limit"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type userDetailResponse struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type adminUserResponse struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	BlockedReason *string    `json:"blocked_reason"`
	BlockedAt     *time.Time `json:"blocked_at"`
}

type challengeResponse struct {
	ID                  int64                     `json:"id"`
	Title               string                    `json:"title"`
	Description         string                    `json:"description"`
	Category            string                    `json:"category"`
	Level               int                       `json:"level"`
	Points              int                       `json:"points"`
	SolveCount          int                       `json:"solve_count"`
	LevelVoteCounts     []models.LevelVoteCount   `json:"level_vote_counts,omitempty"`
	CreatedByUserID     *int64                    `json:"created_by_user_id,omitempty"`
	CreatedByUsername   string                    `json:"created_by_username,omitempty"`
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
	ID                        int64   `json:"id"`
	Title                     string  `json:"title"`
	Category                  string  `json:"category"`
	Level                     int     `json:"level"`
	Points                    int     `json:"points"`
	SolveCount                int     `json:"solve_count"`
	CreatedByUserID           *int64  `json:"created_by_user_id,omitempty"`
	CreatedByUsername         string  `json:"created_by_username,omitempty"`
	PreviousChallengeID       *int64  `json:"previous_challenge_id,omitempty"`
	PreviousChallengeTitle    *string `json:"previous_challenge_title,omitempty"`
	PreviousChallengeCategory *string `json:"previous_challenge_category,omitempty"`
	IsActive                  bool    `json:"is_active"`
	IsLocked                  bool    `json:"is_locked"`
	IsSolved                  bool    `json:"is_solved"`
}

type challengeVotesResponse struct {
	Votes      []models.ChallengeVoteDetail `json:"votes,omitempty"`
	Pagination models.Pagination            `json:"pagination"`
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
	SolvedAt     time.Time `json:"solved_at"`
	IsFirstBlood bool      `json:"is_first_blood"`
}

type challengeSolversResponse struct {
	Solvers    []challengeSolverResponse `json:"solvers,omitempty"`
	Pagination models.Pagination         `json:"pagination"`
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

type leaderboardListResponse struct {
	Challenges []models.LeaderboardChallenge `json:"challenges"`
	Entries    []models.LeaderboardEntry     `json:"entries"`
	Pagination models.Pagination             `json:"pagination"`
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
	return userMeResponse{ID: user.ID, Email: user.Email, Username: user.Username, Role: user.Role, StackCount: stackCount, StackLimit: stackLimit, BlockedReason: user.BlockedReason, BlockedAt: user.BlockedAt}
}

func newUserDetailResponse(user *models.User) userDetailResponse {
	return userDetailResponse{ID: user.ID, Username: user.Username, Role: user.Role, BlockedReason: user.BlockedReason, BlockedAt: user.BlockedAt}
}

func newAdminUserResponse(user *models.User) adminUserResponse {
	return adminUserResponse{ID: user.ID, Email: user.Email, Username: user.Username, Role: user.Role, BlockedReason: user.BlockedReason, BlockedAt: user.BlockedAt}
}

func newChallengeResponse(challenge *models.Challenge, isSolved bool) challengeResponse {
	hasFile := challenge.FileKey != nil && *challenge.FileKey != ""
	return challengeResponse{
		ID:                  challenge.ID,
		Title:               challenge.Title,
		Description:         challenge.Description,
		Category:            challenge.Category,
		Level:               challenge.Level,
		Points:              challenge.Points,
		SolveCount:          challenge.SolveCount,
		LevelVoteCounts:     challenge.LevelVotes,
		CreatedByUserID:     challenge.CreatedByUserID,
		CreatedByUsername:   challenge.CreatedByUsername,
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

func newLockedChallengeResponse(challenge *models.Challenge, previous *models.Challenge, isSolved bool) lockedChallengeResponse {
	var prevTitle *string
	var prevCategory *string
	if previous != nil {
		prevTitle = &previous.Title
		prevCategory = &previous.Category
	}
	return lockedChallengeResponse{
		ID:                        challenge.ID,
		Title:                     challenge.Title,
		Category:                  challenge.Category,
		Level:                     challenge.Level,
		Points:                    challenge.Points,
		SolveCount:                challenge.SolveCount,
		CreatedByUserID:           challenge.CreatedByUserID,
		CreatedByUsername:         challenge.CreatedByUsername,
		PreviousChallengeID:       challenge.PreviousChallengeID,
		PreviousChallengeTitle:    prevTitle,
		PreviousChallengeCategory: prevCategory,
		IsActive:                  challenge.IsActive,
		IsLocked:                  true,
		IsSolved:                  isSolved,
	}
}
