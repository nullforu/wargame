import { useEffect, useRef, useState } from 'react'
import { formatApiError } from '../lib/utils'
import type { Challenge, PaginationMeta, ScoreEntry } from '../lib/types'
import LoginRequired from '../components/LoginRequired'
import { getCategoryKey, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import { CHALLENGE_CATEGORIES } from '../lib/constants'
import { useAuth } from '../lib/auth'
import { navigate } from '../lib/router'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }

type SolveFilter = 'all' | 'solved' | 'unsolved'

const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const parseSolveFilter = (value: string | null): SolveFilter => {
    if (value === 'solved' || value === 'unsolved') return value
    return 'all'
}

const getDifficultyBadgeClass = (level: number) => {
    if (level >= 9) return 'bg-rose-300 text-rose-900 dark:bg-rose-700 dark:text-rose-50'
    if (level >= 7) return 'bg-blue-300 text-blue-900 dark:bg-blue-700 dark:text-blue-50'
    if (level >= 4) return 'bg-sky-200 text-sky-900 dark:bg-sky-700 dark:text-sky-50'
    return 'bg-green-200 text-green-900 dark:bg-green-600 dark:text-green-50'
}

const PRIMARY_CHALLENGE_CATEGORIES = ['Web', 'Pwnable', 'Reversing', 'Crypto', 'Forensics', 'Programming', 'Misc'] as const
const PRIMARY_CATEGORY_SET = new Set<string>(PRIMARY_CHALLENGE_CATEGORIES)
const EXTRA_CHALLENGE_CATEGORIES = CHALLENGE_CATEGORIES.filter((category) => !PRIMARY_CATEGORY_SET.has(category))

const DifficultyBadge = ({ level, active }: { level: number; active?: boolean }) => {
    return (
        <span
            className={`
                inline-flex items-center justify-center
                rounded-full bg-white
                border border-border
                ${active ? 'ring-1 ring-accent' : ''}
                transition
                h-8 w-8
            `}
        >
            <span
                className={`
                    inline-flex items-center justify-center
                    rounded-full text-[11px] font-bold
                    ${getDifficultyBadgeClass(level)}
                    h-6.5 w-6.5
                `}
            >
                {level}
            </span>
        </span>
    )
}

const Challenges = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const readQueryState = () => {
        if (typeof window === 'undefined') {
            return { q: '', page: 1, category: 'all', level: 0, solved: 'all' as SolveFilter }
        }
        const params = new URLSearchParams(window.location.search)
        const categoryParam = params.get('category')
        return {
            q: (params.get('q') ?? '').trim(),
            page: parsePositiveInt(params.get('page'), 1),
            category: categoryParam && categoryParam.trim() !== '' ? categoryParam : 'all',
            level: Math.max(0, Math.min(10, parsePositiveInt(params.get('level'), 0))),
            solved: parseSolveFilter(params.get('solved')),
        }
    }

    const initialQueryState = readQueryState()
    const [searchQuery, setSearchQuery] = useState(initialQueryState.q)
    const [appliedSearch, setAppliedSearch] = useState(initialQueryState.q)
    const [page, setPage] = useState(initialQueryState.page)
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const [categoryFilter, setCategoryFilter] = useState<string>(initialQueryState.category)
    const [levelFilter, setLevelFilter] = useState<number>(initialQueryState.level)
    const [solveFilter, setSolveFilter] = useState<SolveFilter>(initialQueryState.solved)

    const [topUsers, setTopUsers] = useState<ScoreEntry[]>([])
    const [isExtraCategoryOpen, setIsExtraCategoryOpen] = useState(false)
    const extraCategoryMenuRef = useRef<HTMLDivElement | null>(null)
    const isExtraCategorySelected = categoryFilter !== 'all' && !PRIMARY_CATEGORY_SET.has(categoryFilter)

    const pushQueryState = (next: { q: string; page: number; category: string; level: number; solved: SolveFilter }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        if (next.category !== 'all') params.set('category', next.category)
        if (next.level > 0) params.set('level', String(next.level))
        if (next.solved !== 'all') params.set('solved', next.solved)
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }

    const loadChallenges = async (targetPage: number) => {
        setLoading(true)
        setErrorMessage('')

        try {
            const data = await api.searchChallenges(appliedSearch, targetPage, PAGE_SIZE, {
                category: categoryFilter === 'all' ? undefined : categoryFilter,
                level: levelFilter > 0 ? levelFilter : undefined,
                solved: solveFilter === 'all' ? undefined : solveFilter === 'solved',
            })
            setChallenges(data.challenges)
            setPagination(data.pagination)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setPagination(EMPTY_PAGINATION)
        } finally {
            setLoading(false)
        }
    }

    const loadTopUsers = async () => {
        try {
            const leaderboard = await api.leaderboard()
            setTopUsers(leaderboard.entries.slice(0, 10))
        } catch {
            setTopUsers([])
        }
    }

    useEffect(() => {
        if (!auth.user) return
        void Promise.all([loadChallenges(page), loadTopUsers()])
    }, [auth.user?.id, page, appliedSearch, categoryFilter, levelFilter, solveFilter])

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setPage(state.page)
            setCategoryFilter(state.category)
            setLevelFilter(state.level)
            setSolveFilter(state.solved)
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    useEffect(() => {
        if (!isExtraCategoryOpen) return

        const onPointerDown = (event: MouseEvent) => {
            if (extraCategoryMenuRef.current && !extraCategoryMenuRef.current.contains(event.target as Node)) {
                setIsExtraCategoryOpen(false)
            }
        }

        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') setIsExtraCategoryOpen(false)
        }

        window.addEventListener('mousedown', onPointerDown)
        window.addEventListener('keydown', onKeyDown)
        return () => {
            window.removeEventListener('mousedown', onPointerDown)
            window.removeEventListener('keydown', onKeyDown)
        }
    }, [isExtraCategoryOpen])

    if (!auth.user) {
        return <LoginRequired title={t('challenges.title')} />
    }

    return (
        <section className='animate space-y-4'>
            <div className='grid gap-4 lg:grid-cols-[1.9fr_0.9fr]'>
                <div className='space-y-3'>
                    <div className='space-y-2 bg-transparent shadow-none md:bg-surface md:p-3 dark:bg-surface'>
                        <div className='flex items-center gap-2 mb-4'>
                            <div className='relative flex-1'>
                                <span className='pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle dark:text-text-subtle'>⌕</span>
                                <input
                                    type='text'
                                    placeholder={t('challenges.searchPlaceholder')}
                                    value={searchQuery}
                                    onChange={(event) => setSearchQuery(event.target.value)}
                                    className='w-full rounded-lg border border-border/70 bg-surface py-2.5 pl-9 pr-3 text-sm text-text placeholder:text-text-subtle dark:border-border/70 dark:bg-surface dark:text-text dark:placeholder:text-text-subtle'
                                />
                            </div>
                            <button
                                type='button'
                                className='rounded-md border border-border/70 bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle dark:border-border/70 dark:text-text dark:hover:bg-surface-muted h-10'
                                onClick={() => {
                                    const nextQ = searchQuery.trim()
                                    setAppliedSearch(nextQ)
                                    setPage(1)
                                    pushQueryState({ q: nextQ, page: 1, category: categoryFilter, level: levelFilter, solved: solveFilter })
                                }}
                            >
                                {t('common.search')}
                            </button>
                        </div>
                        <div className='flex flex-wrap items-center gap-2'>
                            <span className='w-14 text-xs text-text-muted dark:text-text-muted'>{t('challenges.filterCategory')}</span>
                            <button
                                type='button'
                                className={`rounded-md border px-3 py-1 text-xs ${categoryFilter === 'all' ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                onClick={() => {
                                    setCategoryFilter('all')
                                    setPage(1)
                                    setIsExtraCategoryOpen(false)
                                    pushQueryState({ q: appliedSearch, page: 1, category: 'all', level: levelFilter, solved: solveFilter })
                                }}
                            >
                                {t('common.all')}
                            </button>
                            {PRIMARY_CHALLENGE_CATEGORIES.map((category) => (
                                <button
                                    key={category}
                                    type='button'
                                    className={`rounded-md border px-3 py-1 text-xs ${categoryFilter === category ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                    onClick={() => {
                                        setCategoryFilter(category)
                                        setPage(1)
                                        setIsExtraCategoryOpen(false)
                                        pushQueryState({ q: appliedSearch, page: 1, category, level: levelFilter, solved: solveFilter })
                                    }}
                                >
                                    {t(getCategoryKey(category))}
                                </button>
                            ))}
                            {EXTRA_CHALLENGE_CATEGORIES.length > 0 ? (
                                <div className='relative' ref={extraCategoryMenuRef}>
                                    <button
                                        type='button'
                                        className={`inline-flex items-center gap-1 px-3 py-1 text-xs transition ${isExtraCategorySelected ? 'border-accent/60 bg-accent/12 text-accent rounded-md border' : 'text-text-muted'}`}
                                        onClick={() => setIsExtraCategoryOpen((prev) => !prev)}
                                        aria-haspopup='menu'
                                        aria-expanded={isExtraCategoryOpen}
                                    >
                                        {isExtraCategorySelected ? t(getCategoryKey(categoryFilter)) : t('challenges.filterCategoryMore')}
                                        <svg className={`h-3 w-3 transition-transform ${isExtraCategoryOpen ? 'rotate-180' : ''}`} viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2'>
                                            <path d='m6 9 6 6 6-6' />
                                        </svg>
                                    </button>

                                    {isExtraCategoryOpen ? (
                                        <div className='absolute left-0 top-full z-20 mt-1 min-w-40 rounded-md border border-border bg-surface p-1 shadow-lg'>
                                            {EXTRA_CHALLENGE_CATEGORIES.map((category) => (
                                                <button
                                                    key={category}
                                                    type='button'
                                                    className={`block w-full rounded px-2 py-1.5 text-left text-xs ${categoryFilter === category ? 'bg-accent/12 text-accent' : 'text-text-muted hover:bg-surface-muted hover:text-text'}`}
                                                    onClick={() => {
                                                        setCategoryFilter(category)
                                                        setPage(1)
                                                        pushQueryState({ q: appliedSearch, page: 1, category, level: levelFilter, solved: solveFilter })
                                                        setIsExtraCategoryOpen(false)
                                                    }}
                                                >
                                                    {t(getCategoryKey(category))}
                                                </button>
                                            ))}
                                        </div>
                                    ) : null}
                                </div>
                            ) : null}
                        </div>

                        <div className='flex flex-wrap items-center gap-2'>
                            <span className='w-14 text-xs text-text-muted dark:text-text-muted'>{t('challenges.filterLevel')}</span>
                            <button
                                type='button'
                                className={`rounded-md border px-3 py-1 text-xs ${levelFilter === 0 ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                onClick={() => {
                                    setLevelFilter(0)
                                    setPage(1)
                                    pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: 0, solved: solveFilter })
                                }}
                            >
                                {t('common.all')}
                            </button>
                            {Array.from({ length: 10 }, (_, idx) => idx + 1).map((level) => (
                                <button
                                    key={level}
                                    type='button'
                                    onClick={() => {
                                        setLevelFilter(level)
                                        setPage(1)
                                        pushQueryState({
                                            q: appliedSearch,
                                            page: 1,
                                            category: categoryFilter,
                                            level,
                                            solved: solveFilter,
                                        })
                                    }}
                                    className='transition hover:scale-105'
                                >
                                    <DifficultyBadge level={level} active={levelFilter === level} />
                                </button>
                            ))}
                        </div>

                        <div className='flex flex-wrap items-center gap-2'>
                            <span className='w-14 text-xs text-text-muted dark:text-text-muted'>{t('challenges.filterSolve')}</span>
                            {(['all', 'solved', 'unsolved'] as const).map((key) => (
                                <button
                                    key={key}
                                    type='button'
                                    className={`rounded-md border px-3 py-1 text-xs ${solveFilter === key ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                    onClick={() => {
                                        setSolveFilter(key)
                                        setPage(1)
                                        pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: levelFilter, solved: key })
                                    }}
                                >
                                    {key === 'all' ? t('challenges.filterAll') : key === 'solved' ? t('challenges.filterSolved') : t('challenges.filterUnsolved')}
                                </button>
                            ))}
                        </div>
                    </div>

                    <div className='-mx-4 md:mx-0 overflow-hidden rounded-none md:rounded-xl bg-transparent md:bg-surface md:shadow-sm'>
                        {loading ? (
                            <div className='px-4 py-8 text-sm text-text-muted'>{t('common.loading')}</div>
                        ) : errorMessage ? (
                            <div className='px-4 py-8 text-sm text-danger'>{errorMessage}</div>
                        ) : challenges.length === 0 ? (
                            <div className='px-4 py-8 text-sm text-text-muted'>{t('users.noResults')}</div>
                        ) : (
                            <div className='overflow-x-auto'>
                                <div className='min-w-150'>
                                    <div className='grid grid-cols-[minmax(160px,2fr)_1fr_70px_90px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_100px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_110px] bg-surface-muted px-4 py-2 text-[12px] text-text-muted'>
                                        <span>{t('challenges.tableProblem')}</span>
                                        <span>{t('common.category')}</span>
                                        <span>{t('challenges.tableSolveCount')}</span>
                                        <span>{t('common.status')}</span>
                                    </div>

                                    <div>
                                        {challenges.map((challenge) => {
                                            const category = 'category' in challenge ? challenge.category : t('common.na')
                                            const solveCount = 'solve_count' in challenge ? challenge.solve_count : 0
                                            const solved = challenge.is_solved === true
                                            const locked = challenge.is_locked === true
                                            const inactive = challenge.is_active === false

                                            const statusLabel = locked ? t('challenge.lockedLabel') : inactive ? t('challenge.inactiveLabel') : solved ? t('challenge.solvedLabel') : t('challenge.unsolvedLabel')

                                            const statusColor = locked || inactive ? 'text-text-subtle' : solved ? 'text-success' : 'text-warning'

                                            return (
                                                <button
                                                    key={challenge.id}
                                                    type='button'
                                                    className='grid w-full grid-cols-[minmax(160px,2fr)_1fr_70px_90px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_100px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_110px] items-center px-4 py-3 text-left transition hover:bg-surface-muted disabled:cursor-not-allowed disabled:opacity-70'
                                                    disabled={inactive}
                                                    onClick={() => {
                                                        if (!inactive) {
                                                            navigate(`/challenges/${challenge.id}${window.location.search}`)
                                                        }
                                                    }}
                                                >
                                                    <div className='flex items-center gap-3 min-w-0'>
                                                        <DifficultyBadge level={challenge.level} />
                                                        <div className='flex items-center min-w-0 flex-1'>
                                                            <span className='truncate text-[14px] sm:text-[16px] font-semibold pr-4'>{challenge.title}</span>
                                                            {challenge.is_solved && (
                                                                <span className='shrink-0 w-4 h-4 text-accent -ml-1.5'>
                                                                    <svg viewBox='0 0 24 24' className='w-full h-full'>
                                                                        <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity='0.7' />
                                                                        <path
                                                                            d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z'
                                                                            fill='none'
                                                                            stroke='currentColor'
                                                                            strokeLinecap='round'
                                                                            strokeLinejoin='round'
                                                                        />
                                                                    </svg>
                                                                </span>
                                                            )}
                                                        </div>
                                                    </div>

                                                    <span className='text-xs text-text-muted wrap-break-words'>{t(getCategoryKey(category))}</span>
                                                    <span className='text-sm text-text-muted'>{solveCount}</span>
                                                    <span className={`text-xs font-medium ${statusColor}`}>{statusLabel}</span>
                                                </button>
                                            )
                                        })}
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>

                    <div className='flex flex-wrap items-center justify-between gap-2 px-0 py-2 text-sm text-text-muted md:px-4 dark:text-text-muted'>
                        <span>{t('common.totalCount', { count: pagination.total_count })}</span>
                        <div className='flex items-center gap-2'>
                            <button
                                type='button'
                                className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50 dark:text-text dark:hover:bg-surface-muted'
                                disabled={!pagination.has_prev}
                                onClick={() => {
                                    const nextPage = Math.max(1, page - 1)
                                    setPage(nextPage)
                                    pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, solved: solveFilter })
                                }}
                            >
                                {t('common.previous')}
                            </button>
                            <span>
                                {pagination.page} / {pagination.total_pages || 1}
                            </span>
                            <button
                                type='button'
                                className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50 dark:text-text dark:hover:bg-surface-muted'
                                disabled={!pagination.has_next}
                                onClick={() => {
                                    const nextPage = page + 1
                                    setPage(nextPage)
                                    pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, solved: solveFilter })
                                }}
                            >
                                {t('common.next')}
                            </button>
                        </div>
                    </div>
                </div>

                <aside className='space-y-3'>
                    <div className='rounded-none bg-transparent p-0 shadow-none md:rounded-xl md:bg-surface-muted md:p-4 md:shadow-sm dark:bg-surface'>
                        <p className='text-sm font-semibold text-text dark:text-text'>{t('challenges.sidebarIdeaTitle')}</p>
                        <p className='mt-1 text-sm text-text-muted dark:text-text-muted'>{t('challenges.sidebarIdeaBody')}</p>
                        <button type='button' className='mt-3 rounded-md text-xs text-accent'>
                            {t('challenges.sidebarIdeaCta')}
                        </button>
                    </div>

                    <div className='rounded-none bg-transparent p-0 shadow-none md:rounded-xl md:bg-surface md:p-4 md:shadow-sm dark:bg-surface'>
                        <p className='text-xl leading-none text-text dark:text-text'>
                            {t('challenges.sidebarTopUsersLine1Prefix')} <span className='text-accent'>{topUsers.length}</span> {t('challenges.sidebarTopUsersLine1Suffix')} {t('challenges.sidebarTopUsersLine2')}
                        </p>
                        <div className='mt-4 space-y-2'>
                            {topUsers.length === 0 ? (
                                <p className='text-sm text-text-muted dark:text-text-muted'>{t('leaderboard.noScores')}</p>
                            ) : (
                                topUsers.map((entry, index) => (
                                    <button
                                        key={`top-user-${entry.username}-${index}`}
                                        className='flex w-full items-center gap-3 rounded px-3 py-2 text-left hover:bg-surface-muted dark:hover:bg-surface-muted'
                                        onClick={() => navigate(`/users/${entry.user_id}`)}
                                    >
                                        <span className='text-xs text-text-subtle'>#{index + 1}</span>
                                        <span className='text-sm font-semibold text-text'>{entry.username}</span>
                                        <span className='ml-auto text-xs text-text-muted'>{t('common.pointsShort', { points: entry.score })}</span>
                                    </button>
                                ))
                            )}
                        </div>
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default Challenges
