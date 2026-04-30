export interface AuthUser {
    id: number
    email: string
    username: string
    role: string
    affiliation_id: number | null
    affiliation: string | null
    bio: string | null
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
    user: AuthUser
}

export interface ChallengeDetail {
    id: number
    title: string
    description: string
    category: string
    created_at: string
    level: number
    level_vote_counts?: LevelVoteCount[]
    first_blood?: ChallengeSolver | null
    points: number
    solve_count: number
    created_by?: ChallengeCreator | null
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
    created_at: string
    level: number
    points: number
    solve_count: number
    first_blood?: ChallengeSolver | null
    created_by?: ChallengeCreator | null
    is_active: boolean
    previous_challenge_id?: number | null
    previous_challenge_title?: string | null
    previous_challenge_category?: string | null
    is_locked: true
    is_solved: boolean
}

export type Challenge = ChallengeDetail | LockedChallenge

export interface ChallengeCreator {
    user_id?: number | null
    username?: string
    affiliation_id?: number | null
    affiliation?: string | null
    bio?: string | null
}

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

export interface ChallengeMyVoteResponse {
    level: number | null
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
    affiliation?: string | null
    bio?: string | null
    solved_at: string
    is_first_blood: boolean
}

export interface ChallengeSolversResponse {
    solvers: ChallengeSolver[]
    pagination: PaginationMeta
}

export interface WriteupAuthor {
    user_id: number
    username: string
    affiliation_id?: number | null
    affiliation?: string | null
    bio?: string | null
}

export interface WriteupChallenge {
    id: number
    title: string
    category: string
    points: number
    level: number
}

export interface Writeup {
    id: number
    content?: string | null
    created_at: string
    updated_at: string
    author: WriteupAuthor
    challenge: WriteupChallenge
}

export interface ChallengeWriteupsResponse {
    writeups: Writeup[]
    can_view_content: boolean
    pagination: PaginationMeta
}

export interface WriteupDetailResponse {
    writeup: Writeup
    can_view_content: boolean
}

export interface ChallengeCommentItemAuthor {
    user_id: number
    username: string
    affiliation_id?: number | null
    affiliation?: string | null
    bio?: string | null
}

export interface ChallengeCommentItem {
    id: number
    content: string
    created_at: string
    updated_at: string
    author: ChallengeCommentItemAuthor
    challenge: {
        id: number
        title: string
    }
}

export interface ChallengeCommentPageResponse {
    comments: ChallengeCommentItem[]
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
    affiliation_id?: number | null
    affiliation?: string | null
    bio?: string | null
    blocked_reason: string | null
    blocked_at: string | null
}

export interface UserDetail {
    id: number
    username: string
    role: string
    affiliation_id: number | null
    affiliation: string | null
    bio: string | null
    blocked_reason: string | null
    blocked_at: string | null
}

export interface Affiliation {
    id: number
    name: string
}

export interface AffiliationsResponse {
    affiliations: Affiliation[]
    pagination: PaginationMeta
}

export interface UserRankingEntry {
    user_id: number
    username: string
    score: number
    solved_count: number
    affiliation_id: number | null
    affiliation_name: string | null
    bio?: string | null
}

export interface UserRankingResponse {
    entries: UserRankingEntry[]
    pagination: PaginationMeta
}

export interface AffiliationRankingEntry {
    affiliation_id: number
    name: string
    score: number
    solved_count: number
    user_count: number
}

export interface AffiliationRankingResponse {
    entries: AffiliationRankingEntry[]
    pagination: PaginationMeta
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
