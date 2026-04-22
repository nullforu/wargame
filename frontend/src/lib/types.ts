export interface AuthUser {
    id: number
    email: string
    username: string
    role: string
    stack_count: number
    stack_limit: number
    blocked_reason: string | null
    blocked_at: string | null
}

export interface RegisterPayload {
    email: string
    username: string
    password: string
}

export interface RegisterResponse {
    id: number
    email: string
    username: string
}

export interface LoginPayload {
    email: string
    password: string
}

export interface AuthResponse {
    access_token: string
    refresh_token: string
    user: AuthUser
}

export interface ChallengeDetail {
    id: number
    title: string
    description: string
    category: string
    level: number
    level_vote_counts?: LevelVoteCount[]
    points: number
    solve_count: number
    created_by_user_id?: number | null
    created_by_username?: string
    is_active: boolean
    has_file: boolean
    file_name?: string | null
    stack_enabled: boolean
    stack_target_ports: TargetPortSpec[]
    previous_challenge_id?: number | null
    is_locked?: false
    is_solved: boolean
}

export interface LockedChallenge {
    id: number
    title: string
    category: string
    level: number
    points: number
    solve_count: number
    created_by_user_id?: number | null
    created_by_username?: string
    is_active: boolean
    previous_challenge_id?: number | null
    previous_challenge_title?: string | null
    previous_challenge_category?: string | null
    is_locked: true
    is_solved: boolean
}

export type Challenge = ChallengeDetail | LockedChallenge

export interface LevelVoteCount {
    level: number
    count: number
}

export interface ChallengeVote {
    user_id: number
    username: string
    level: number
    updated_at: string
}

export interface ChallengeVotesResponse {
    votes: ChallengeVote[]
    pagination: PaginationMeta
}

export interface AdminChallengeDetail extends ChallengeDetail {
    stack_pod_spec?: string | null
}

export type PortProtocol = 'TCP' | 'UDP'

export interface TargetPortSpec {
    container_port: number
    protocol: PortProtocol
}

export interface PortMapping extends TargetPortSpec {
    node_port: number
}

export interface ChallengeCreatePayload {
    title: string
    description: string
    category: string
    points: number
    flag: string
    is_active: boolean
    previous_challenge_id?: number | null
    stack_enabled?: boolean
    stack_target_ports?: TargetPortSpec[]
    stack_pod_spec?: string
}

export interface ChallengeCreateResponse extends ChallengeDetail {}

export interface ChallengeUpdatePayload {
    title?: string
    description?: string
    category?: string
    points?: number
    flag?: string
    is_active?: boolean
    previous_challenge_id?: number | null
    stack_enabled?: boolean
    stack_target_ports?: TargetPortSpec[]
    stack_pod_spec?: string
}

export interface Stack {
    stack_id: string
    challenge_id: number
    challenge_title: string
    status: string
    node_public_ip?: string | null
    ports: PortMapping[]
    ttl_expires_at?: string | null
    created_at: string
    updated_at: string
    created_by_user_id: number
    created_by_username: string
}

export interface AdminStackListItem {
    stack_id: string
    ttl_expires_at?: string | null
    created_at: string
    updated_at: string
    user_id: number
    username: string
    email: string
    challenge_id: number
    challenge_title: string
    challenge_category: string
}

export interface PresignedPost {
    url: string
    fields: Record<string, string>
    expires_at: string
}

export interface PresignedURL {
    url: string
    expires_at: string
}

export interface ChallengeFileUploadResponse {
    challenge: ChallengeDetail
    upload: PresignedPost
}

export interface FlagSubmissionPayload {
    flag: string
}

export interface ChallengesResponse {
    challenges: Challenge[]
    pagination: PaginationMeta
}

export interface StacksResponse {
    stacks: Stack[]
}

export interface AdminStacksResponse {
    stacks: AdminStackListItem[]
}

export interface AdminStackDeleteResponse {
    deleted: boolean
    stack_id: string
}

export interface FlagSubmissionResult {
    correct?: boolean
}

export interface SolvedChallenge {
    challenge_id: number
    title: string
    points: number
    solved_at: string
}

export interface UserSolvedResponse {
    solved: SolvedChallenge[]
    pagination: PaginationMeta
}

export interface ChallengeSolver {
    user_id: number
    username: string
    solved_at: string
    is_first_blood: boolean
}

export interface ChallengeSolversResponse {
    solvers: ChallengeSolver[]
    pagination: PaginationMeta
}

export interface LeaderboardChallenge {
    id: number
    title: string
    category: string
    points: number
}

export interface LeaderboardSolve {
    challenge_id: number
    solved_at: string
    is_first_blood: boolean
}

export interface ScoreEntry {
    user_id: number
    username: string
    score: number
    solves: LeaderboardSolve[]
}

export interface LeaderboardResponse {
    challenges: LeaderboardChallenge[]
    entries: ScoreEntry[]
    pagination: PaginationMeta
}

export interface TimelineSubmission {
    timestamp: string
    user_id: number
    username: string
    points: number
    challenge_count: number
}

export interface TimelineResponse {
    submissions: TimelineSubmission[]
}

export interface UserListItem {
    id: number
    username: string
    role: string
    blocked_reason: string | null
    blocked_at: string | null
}

export interface UserDetail {
    id: number
    username: string
    role: string
    blocked_reason: string | null
    blocked_at: string | null
}

export interface PaginationMeta {
    page: number
    page_size: number
    total_count: number
    total_pages: number
    has_prev: boolean
    has_next: boolean
}

export interface UsersResponse {
    users: UserListItem[]
    pagination: PaginationMeta
}
