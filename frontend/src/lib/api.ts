import type {
    AuthResponse,
    AuthUser,
    Challenge,
    ChallengeDetail,
    ChallengesResponse,
    ChallengeCreatePayload,
    ChallengeCreateResponse,
    ChallengeUpdatePayload,
    ChallengeFileUploadResponse,
    ChallengeMyVoteResponse,
    ChallengeVotesResponse,
    ChallengeWriteupsResponse,
    AdminChallengeDetail,
    AdminStackDeleteResponse,
    AdminStackListItem,
    AdminStacksResponse,
    AffiliationRankingResponse,
    AffiliationsResponse,
    Affiliation,
    FlagSubmissionResult,
    LeaderboardResponse,
    PresignedURL,
    Stack,
    StacksResponse,
    LoginPayload,
    PaginationMeta,
    RegisterPayload,
    RegisterResponse,
    UserSolvedResponse,
    ChallengeSolversResponse,
    TimelineResponse,
    UserDetail,
    UserRankingResponse,
    UsersResponse,
    Writeup,
    WriteupDetailResponse,
} from './types'
import type { AuthState } from './auth'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

export interface ApiErrorDetail {
    field: string
    reason: string
}
export interface RateLimitInfo {
    limit: number
    remaining: number
    reset_seconds: number
}

export class ApiError extends Error {
    status: number
    details?: ApiErrorDetail[]
    rateLimit?: RateLimitInfo

    constructor(message: string, status: number, details?: ApiErrorDetail[], rateLimit?: RateLimitInfo) {
        super(message)
        this.name = 'ApiError'
        this.status = status
        this.details = details
        this.rateLimit = rateLimit
    }
}

interface ApiDeps {
    getAuth: () => AuthState
    setAuthTokens: (accessToken: string, refreshToken: string) => void
    setAuthUser: (user: AuthUser | null) => void
    clearAuth: () => void
    translate: (key: string, vars?: Record<string, string | number>) => string
}

const parseJson = async (response: Response) => {
    const contentType = response.headers.get('content-type') ?? ''
    if (!contentType.includes('application/json')) return null

    return response.json()
}

const extractRateLimit = (response: Response, data: any): RateLimitInfo | undefined => {
    if (data?.rate_limit) return data.rate_limit as RateLimitInfo

    const limit = Number(response.headers.get('x-ratelimit-limit'))
    const remaining = Number(response.headers.get('x-ratelimit-remaining'))
    const resetSeconds = Number(response.headers.get('x-ratelimit-reset'))

    if (Number.isFinite(limit) && Number.isFinite(remaining) && Number.isFinite(resetSeconds)) {
        return { limit, remaining, reset_seconds: resetSeconds }
    }

    return undefined
}

export const createApi = ({ getAuth, setAuthTokens, setAuthUser, clearAuth, translate }: ApiDeps) => {
    const defaultPagination = (): PaginationMeta => ({
        page: 1,
        page_size: 20,
        total_count: 0,
        total_pages: 0,
        has_prev: false,
        has_next: false,
    })

    const normalizePagination = (pagination?: PaginationMeta): PaginationMeta => {
        if (!pagination || typeof pagination !== 'object') return defaultPagination()
        return {
            page: typeof pagination.page === 'number' ? pagination.page : 1,
            page_size: typeof pagination.page_size === 'number' ? pagination.page_size : 20,
            total_count: typeof pagination.total_count === 'number' ? pagination.total_count : 0,
            total_pages: typeof pagination.total_pages === 'number' ? pagination.total_pages : 0,
            has_prev: Boolean(pagination.has_prev),
            has_next: Boolean(pagination.has_next),
        }
    }

    const withPagination = (path: string, page?: number, pageSize?: number) => {
        const params = new URLSearchParams()
        if (typeof page === 'number') params.set('page', String(page))
        if (typeof pageSize === 'number') params.set('page_size', String(pageSize))
        const query = params.toString()
        return query ? `${path}?${query}` : path
    }

    const withSearchAndPagination = (path: string, q: string, page?: number, pageSize?: number) => {
        const params = new URLSearchParams()
        params.set('q', q)
        if (typeof page === 'number') params.set('page', String(page))
        if (typeof pageSize === 'number') params.set('page_size', String(pageSize))
        return `${path}?${params.toString()}`
    }

    const withChallengeFilters = (
        path: string,
        {
            q,
            page,
            pageSize,
            category,
            level,
            solved,
            sort,
        }: {
            q?: string
            page?: number
            pageSize?: number
            category?: string
            level?: number
            solved?: boolean
            sort?: string
        },
    ) => {
        const params = new URLSearchParams()
        if (typeof q === 'string' && q.trim() !== '') params.set('q', q.trim())
        if (typeof page === 'number') params.set('page', String(page))
        if (typeof pageSize === 'number') params.set('page_size', String(pageSize))
        if (typeof category === 'string' && category.trim() !== '') params.set('category', category.trim())
        if (typeof level === 'number') params.set('level', String(level))
        if (typeof solved === 'boolean') params.set('solved', String(solved))
        if (typeof sort === 'string' && sort.trim() !== '') params.set('sort', sort.trim())
        const query = params.toString()
        return query ? `${path}?${query}` : path
    }

    const buildHeaders = (withAuth: boolean, tokenOverride?: string) => {
        const headers: Record<string, string> = { Accept: 'application/json' }

        if (withAuth) {
            const token = tokenOverride ?? getAuth().accessToken
            if (token) headers.Authorization = `Bearer ${token}`
        }

        return headers
    }

    const refreshToken = async () => {
        const refreshTokenValue = getAuth().refreshToken
        if (!refreshTokenValue) throw new ApiError(translate('errors.missingRefreshToken'), 401)

        const response = await fetch(`${API_BASE}/api/auth/refresh`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Accept: 'application/json',
            },
            body: JSON.stringify({ refresh_token: refreshTokenValue }),
        })

        if (!response.ok) {
            const data = await parseJson(response)
            clearAuth()

            throw new ApiError(data?.error ?? translate('errors.invalidCredentials'), response.status, data?.details, extractRateLimit(response, data))
        }

        const data = await response.json()
        setAuthTokens(data.access_token, data.refresh_token)

        return data.access_token as string
    }

    let refreshInFlight: Promise<string> | null = null

    const getFreshToken = async () => {
        if (refreshInFlight) return refreshInFlight
        refreshInFlight = (async () => {
            try {
                return await refreshToken()
            } finally {
                refreshInFlight = null
            }
        })()
        return refreshInFlight
    }

    const request = async <T>(
        path: string,
        {
            method = 'GET',
            body,
            auth = false,
            retryOnAuth = true,
            noCache = false,
        }: {
            method?: string
            body?: unknown
            auth?: boolean
            retryOnAuth?: boolean
            noCache?: boolean
        } = {},
    ): Promise<T> => {
        const headers = buildHeaders(auth)
        if (body !== undefined) headers['Content-Type'] = 'application/json'
        if (noCache) {
            headers['Cache-Control'] = 'no-cache'
            headers.Pragma = 'no-cache'
        }

        const response = await fetch(`${API_BASE}${path}`, {
            method,
            headers,
            body: body !== undefined ? JSON.stringify(body) : undefined,
            cache: noCache ? 'no-store' : 'default',
        })

        if (response.ok) {
            if (response.status === 204) return null as T
            const data = await parseJson(response)
            return data as T
        }

        if (response.status === 401 && auth && retryOnAuth) {
            try {
                const newToken = await getFreshToken()
                const retryHeaders = buildHeaders(true, newToken)
                if (body !== undefined) retryHeaders['Content-Type'] = 'application/json'
                if (noCache) {
                    retryHeaders['Cache-Control'] = 'no-cache'
                    retryHeaders.Pragma = 'no-cache'
                }

                const retryResponse = await fetch(`${API_BASE}${path}`, {
                    method,
                    headers: retryHeaders,
                    body: body !== undefined ? JSON.stringify(body) : undefined,
                    cache: noCache ? 'no-store' : 'default',
                })

                if (retryResponse.ok) {
                    if (retryResponse.status === 204) return null as T
                    return (await parseJson(retryResponse)) as T
                }

                const retryData = await parseJson(retryResponse)
                throw new ApiError(retryData?.error ?? translate('errors.requestFailed'), retryResponse.status, retryData?.details, extractRateLimit(retryResponse, retryData))
            } catch (error) {
                if (error instanceof ApiError) throw error
                clearAuth()
                throw new ApiError(translate('errors.invalidCredentials'), 401)
            }
        }

        const data = await parseJson(response)

        throw new ApiError(data?.error ?? translate('errors.requestFailed'), response.status, data?.details, extractRateLimit(response, data))
    }

    return {
        register: (payload: RegisterPayload) => request<RegisterResponse>(`/api/auth/register`, { method: 'POST', body: payload }),
        login: async (payload: LoginPayload) => {
            const data = await request<AuthResponse>(`/api/auth/login`, { method: 'POST', body: payload })
            setAuthTokens(data.access_token, data.refresh_token)
            setAuthUser(data.user)
            return data
        },
        logout: async () => {
            const refreshTokenValue = getAuth().refreshToken
            if (!refreshTokenValue) {
                clearAuth()
                return
            }
            await request(`/api/auth/logout`, { method: 'POST', body: { refresh_token: refreshTokenValue } })
            clearAuth()
        },
        me: () => request<AuthUser>(`/api/me`, { auth: true }),
        updateMe: (payload: { username?: string; affiliation_id?: number | null; bio?: string | null }) => request<AuthUser>(`/api/me`, { method: 'PUT', body: payload, auth: true }),
        challenges: async (page?: number, pageSize?: number) => {
            const data = await request<{ challenges?: Challenge[]; pagination?: PaginationMeta }>(withPagination(`/api/challenges`, page, pageSize), { auth: true })
            return {
                challenges: Array.isArray(data?.challenges) ? data.challenges : [],
                pagination: normalizePagination(data?.pagination),
            } as ChallengesResponse
        },
        searchChallenges: async (
            q: string,
            page?: number,
            pageSize?: number,
            filters?: {
                category?: string
                level?: number
                solved?: boolean
                sort?: string
            },
        ) => {
            const trimmedQ = q.trim()
            const basePath = trimmedQ === '' ? `/api/challenges` : `/api/challenges/search`
            const data = await request<{ challenges?: Challenge[]; pagination?: PaginationMeta }>(
                withChallengeFilters(basePath, { q: trimmedQ, page, pageSize, category: filters?.category, level: filters?.level, solved: filters?.solved, sort: filters?.sort }),
                { auth: true },
            )
            return {
                challenges: Array.isArray(data?.challenges) ? data.challenges : [],
                pagination: normalizePagination(data?.pagination),
            } as ChallengesResponse
        },
        challenge: (id: number) => request<Challenge>(`/api/challenges/${id}`, { auth: true }),
        challengeSolvers: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<ChallengeSolversResponse>>(withPagination(`/api/challenges/${id}/solvers`, page, pageSize))
            return {
                solvers: Array.isArray(data?.solvers) ? data.solvers : [],
                pagination: normalizePagination(data?.pagination),
            } as ChallengeSolversResponse
        },
        challengeVotes: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<ChallengeVotesResponse>>(withPagination(`/api/challenges/${id}/votes`, page, pageSize))
            return {
                votes: Array.isArray(data?.votes) ? data.votes : [],
                pagination: normalizePagination(data?.pagination),
            } as ChallengeVotesResponse
        },
        challengeWriteups: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<ChallengeWriteupsResponse>>(withPagination(`/api/challenges/${id}/writeups`, page, pageSize), { auth: true })
            return {
                writeups: Array.isArray(data?.writeups) ? data.writeups : [],
                can_view_content: Boolean(data?.can_view_content),
                pagination: normalizePagination(data?.pagination),
            } as ChallengeWriteupsResponse
        },
        writeup: async (id: number) => {
            const data = await request<Partial<WriteupDetailResponse>>(`/api/writeups/${id}`, { auth: true })
            return {
                writeup: data?.writeup as Writeup,
                can_view_content: Boolean(data?.can_view_content),
            } as WriteupDetailResponse
        },
        createWriteup: (challengeID: number, content: string) =>
            request<Writeup>(`/api/challenges/${challengeID}/writeups`, {
                method: 'POST',
                body: { content },
                auth: true,
            }),
        updateWriteup: (writeupID: number, payload: { content?: string }) =>
            request<Writeup>(`/api/writeups/${writeupID}`, {
                method: 'PATCH',
                body: payload,
                auth: true,
            }),
        deleteWriteup: (writeupID: number) =>
            request<{ status?: string }>(`/api/writeups/${writeupID}`, {
                method: 'DELETE',
                auth: true,
            }),
        myWriteups: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<ChallengeWriteupsResponse>>(withPagination(`/api/me/writeups`, page, pageSize), { auth: true })
            return {
                writeups: Array.isArray(data?.writeups) ? data.writeups : [],
                can_view_content: true,
                pagination: normalizePagination(data?.pagination),
            } as ChallengeWriteupsResponse
        },
        challengeMyVote: async (id: number) => {
            const data = await request<Partial<ChallengeMyVoteResponse>>(`/api/challenges/${id}/my-vote`, { auth: true })
            return {
                level: typeof data?.level === 'number' ? data.level : null,
            } as ChallengeMyVoteResponse
        },
        voteChallengeLevel: (id: number, level: number) =>
            request<{ status: string }>(`/api/challenges/${id}/vote`, {
                method: 'POST',
                body: { level },
                auth: true,
            }),
        submitFlag: (id: number, flag: string) =>
            request<FlagSubmissionResult>(`/api/challenges/${id}/submit`, {
                method: 'POST',
                body: { flag },
                auth: true,
            }),
        leaderboard: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<LeaderboardResponse>>(withPagination(`/api/leaderboard`, page, pageSize))
            return {
                challenges: Array.isArray(data?.challenges) ? data.challenges : [],
                entries: Array.isArray(data?.entries) ? data.entries : [],
                pagination: normalizePagination(data?.pagination),
            } as LeaderboardResponse
        },
        legacyLeaderboard: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<LeaderboardResponse>>(withPagination(`/api/leaderboard`, page, pageSize))
            return {
                challenges: Array.isArray(data?.challenges) ? data.challenges : [],
                entries: Array.isArray(data?.entries) ? data.entries : [],
                pagination: normalizePagination(data?.pagination),
            } as LeaderboardResponse
        },
        timeline: async () => {
            const data = await request<Partial<TimelineResponse>>(`/api/timeline`, { auth: true })
            return {
                submissions: Array.isArray(data?.submissions) ? data.submissions : [],
            } as TimelineResponse
        },
        createChallenge: (payload: ChallengeCreatePayload) => request<ChallengeCreateResponse>(`/api/admin/challenges`, { method: 'POST', body: payload, auth: true }),
        adminChallenge: (id: number) => request<AdminChallengeDetail>(`/api/admin/challenges/${id}`, { auth: true }),
        updateChallenge: (id: number, payload: ChallengeUpdatePayload) => request<ChallengeDetail>(`/api/admin/challenges/${id}`, { method: 'PUT', body: payload, auth: true }),
        deleteChallenge: (id: number) => request<void>(`/api/admin/challenges/${id}`, { method: 'DELETE', auth: true }),
        requestChallengeFileUpload: (id: number, filename: string) =>
            request<ChallengeFileUploadResponse>(`/api/admin/challenges/${id}/file/upload`, {
                method: 'POST',
                body: { filename },
                auth: true,
            }),
        deleteChallengeFile: (id: number) => request<ChallengeDetail>(`/api/admin/challenges/${id}/file`, { method: 'DELETE', auth: true }),
        requestChallengeFileDownload: (id: number) =>
            request<PresignedURL>(`/api/challenges/${id}/file/download`, {
                method: 'POST',
                auth: true,
            }),
        createStack: (challengeID: number) => request<Stack>(`/api/challenges/${challengeID}/stack`, { method: 'POST', auth: true }),
        getStack: (challengeID: number) => request<Stack>(`/api/challenges/${challengeID}/stack`, { auth: true }),
        deleteStack: (challengeID: number) => request<{ status?: string }>(`/api/challenges/${challengeID}/stack`, { method: 'DELETE', auth: true }),
        stacks: async () => {
            const data = await request<{ stacks?: Stack[] }>(`/api/stacks`, { auth: true })
            return {
                stacks: Array.isArray(data?.stacks) ? data.stacks : [],
            } as StacksResponse
        },
        adminStacks: async () => {
            const data = await request<{ stacks?: AdminStackListItem[] }>(`/api/admin/stacks`, { auth: true })
            return {
                stacks: Array.isArray(data?.stacks) ? data.stacks : [],
            } as AdminStacksResponse
        },
        adminStack: (stackId: string) => request<Stack>(`/api/admin/stacks/${stackId}`, { auth: true }),
        deleteAdminStack: (stackId: string) => request<AdminStackDeleteResponse>(`/api/admin/stacks/${stackId}`, { method: 'DELETE', auth: true }),
        blockUser: (id: number, reason: string) => request<AuthUser>(`/api/admin/users/${id}/block`, { method: 'POST', body: { reason }, auth: true }),
        unblockUser: (id: number) => request<AuthUser>(`/api/admin/users/${id}/unblock`, { method: 'POST', auth: true }),
        users: async (page?: number, pageSize?: number) => {
            const data = await request<UsersResponse>(withPagination(`/api/users`, page, pageSize))
            return {
                users: Array.isArray(data?.users) ? data.users : [],
                pagination: normalizePagination(data?.pagination),
            } as UsersResponse
        },
        searchUsers: async (q: string, page?: number, pageSize?: number) => {
            const data = await request<UsersResponse>(withSearchAndPagination(`/api/users/search`, q, page, pageSize))
            return {
                users: Array.isArray(data?.users) ? data.users : [],
                pagination: normalizePagination(data?.pagination),
            } as UsersResponse
        },
        user: (id: number) => request<UserDetail>(`/api/users/${id}`),
        affiliations: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<AffiliationsResponse>>(withPagination(`/api/affiliations`, page, pageSize))
            return {
                affiliations: Array.isArray(data?.affiliations) ? data.affiliations : [],
                pagination: normalizePagination(data?.pagination),
            } as AffiliationsResponse
        },
        searchAffiliations: async (q: string, page?: number, pageSize?: number) => {
            const data = await request<Partial<AffiliationsResponse>>(withSearchAndPagination(`/api/affiliations/search`, q, page, pageSize))
            return {
                affiliations: Array.isArray(data?.affiliations) ? data.affiliations : [],
                pagination: normalizePagination(data?.pagination),
            } as AffiliationsResponse
        },
        affiliationUsers: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<UsersResponse>(withPagination(`/api/affiliations/${id}/users`, page, pageSize))
            return {
                users: Array.isArray(data?.users) ? data.users : [],
                pagination: normalizePagination(data?.pagination),
            } as UsersResponse
        },
        rankingUsers: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<UserRankingResponse>>(withPagination(`/api/rankings/users`, page, pageSize))
            return {
                entries: Array.isArray(data?.entries) ? data.entries : [],
                pagination: normalizePagination(data?.pagination),
            } as UserRankingResponse
        },
        rankingAffiliations: async (page?: number, pageSize?: number) => {
            const data = await request<Partial<AffiliationRankingResponse>>(withPagination(`/api/rankings/affiliations`, page, pageSize))
            return {
                entries: Array.isArray(data?.entries) ? data.entries : [],
                pagination: normalizePagination(data?.pagination),
            } as AffiliationRankingResponse
        },
        rankingAffiliationUsers: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<UserRankingResponse>>(withPagination(`/api/rankings/affiliations/${id}/users`, page, pageSize))
            return {
                entries: Array.isArray(data?.entries) ? data.entries : [],
                pagination: normalizePagination(data?.pagination),
            } as UserRankingResponse
        },
        createAffiliation: (name: string) => request<Affiliation>(`/api/admin/affiliations`, { method: 'POST', body: { name }, auth: true }),
        userSolved: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<UserSolvedResponse>>(withPagination(`/api/users/${id}/solved`, page, pageSize))
            return {
                solved: Array.isArray(data?.solved) ? data.solved : [],
                pagination: normalizePagination(data?.pagination),
            } as UserSolvedResponse
        },
        userWriteups: async (id: number, page?: number, pageSize?: number) => {
            const data = await request<Partial<ChallengeWriteupsResponse>>(withPagination(`/api/users/${id}/writeups`, page, pageSize), { auth: true })
            return {
                writeups: Array.isArray(data?.writeups) ? data.writeups : [],
                can_view_content: Boolean(data?.can_view_content),
                pagination: normalizePagination(data?.pagination),
            } as ChallengeWriteupsResponse
        },
    }
}

export const uploadPresignedPost = async (upload: { url: string; fields: Record<string, string> }, file: File) => {
    const formData = new FormData()
    Object.entries(upload.fields).forEach(([key, value]) => {
        formData.append(key, value)
    })
    formData.append('file', file)

    try {
        const response = await fetch(upload.url, { method: 'POST', body: formData })
        if (!response.ok) {
            throw new Error('File upload failed')
        }
    } catch (error) {
        throw error
    }
}
