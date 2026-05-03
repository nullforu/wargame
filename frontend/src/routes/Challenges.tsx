import { useEffect, useRef, useState } from 'react'
import { formatApiError } from '../lib/utils'
import type { Challenge, PaginationMeta, UserRankingEntry } from '../lib/types'
import { getCategoryKey, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import { CHALLENGE_CATEGORIES } from '../lib/constants'
import { navigate } from '../lib/router'
import UserAvatar from '../components/UserAvatar'
import { levelBadgeClass, normalizeLevel } from '../lib/level'
import FlagIcon from '../components/FlagIcon'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const CHALLENGE_SKELETON_ROWS = 5

type SolveFilter = 'all' | 'solved' | 'unsolved'
type SortFilter = 'latest' | 'oldest' | 'most_solved' | 'least_solved'
const ALL_LEVEL_FILTER = -1

const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const parseLevelFilter = (value: string | null): number => {
    if (value == null || value.trim() === '') return ALL_LEVEL_FILTER
    const parsed = Number(value)
    if (!Number.isInteger(parsed)) return ALL_LEVEL_FILTER
    if (parsed < 0 || parsed > 10) return ALL_LEVEL_FILTER
    return parsed
}

const parseSolveFilter = (value: string | null): SolveFilter => {
    if (value === 'solved' || value === 'unsolved') return value
    return 'all'
}

const parseSortFilter = (value: string | null): SortFilter => {
    if (value === 'oldest' || value === 'most_solved' || value === 'least_solved') return value
    return 'latest'
}

const PRIMARY_CHALLENGE_CATEGORIES = ['Web', 'Pwnable', 'Reversing', 'Crypto', 'Forensics', 'Programming', 'Misc'] as const
const PRIMARY_CATEGORY_SET = new Set<string>(PRIMARY_CHALLENGE_CATEGORIES)
const EXTRA_CHALLENGE_CATEGORIES = CHALLENGE_CATEGORIES.filter((category) => !PRIMARY_CATEGORY_SET.has(category))

export const LevelBadge = ({ level, active }: { level?: number; active?: boolean }) => {
    const normalized = normalizeLevel(level)
    return (
        <span
            className={`
                inline-flex items-center justify-center
                rounded-full bg-white dark:bg-surface
                border border-border dark:border-border/80
                ${active ? 'ring-1 ring-accent' : ''}
                transition
                h-8 w-8
            `}
        >
            <span
                className={`
                    inline-flex items-center justify-center
                    rounded-full text-[11px] font-bold
                    ${levelBadgeClass(normalized)}
                    h-6.5 w-6.5
                `}
            >
                {normalized > 0 ? normalized : '?'}
            </span>
        </span>
    )
}

const Challenges = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const readQueryState = () => {
        if (typeof window === 'undefined') {
            return { q: '', page: 1, category: 'all', level: ALL_LEVEL_FILTER, solved: 'all' as SolveFilter, sort: 'latest' as SortFilter }
        }
        const params = new URLSearchParams(window.location.search)
        const categoryParam = params.get('category')
        return {
            q: (params.get('q') ?? '').trim(),
            page: parsePositiveInt(params.get('page'), 1),
            category: categoryParam && categoryParam.trim() !== '' ? categoryParam : 'all',
            level: parseLevelFilter(params.get('level')),
            solved: parseSolveFilter(params.get('solved')),
            sort: parseSortFilter(params.get('sort')),
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
    const [sortFilter, setSortFilter] = useState<SortFilter>(initialQueryState.sort)

    const [topUsers, setTopUsers] = useState<UserRankingEntry[]>([])
    const [isExtraCategoryOpen, setIsExtraCategoryOpen] = useState(false)
    const [isSortMenuOpen, setIsSortMenuOpen] = useState(false)
    const extraCategoryMenuRef = useRef<HTMLDivElement | null>(null)
    const sortMenuRef = useRef<HTMLDivElement | null>(null)
    const isExtraCategorySelected = categoryFilter !== 'all' && !PRIMARY_CATEGORY_SET.has(categoryFilter)

    const pushQueryState = (next: { q: string; page: number; category: string; level: number; solved: SolveFilter; sort: SortFilter }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        if (next.category !== 'all') params.set('category', next.category)
        if (next.level >= 0) params.set('level', String(next.level))
        if (next.solved !== 'all') params.set('solved', next.solved)
        if (next.sort !== 'latest') params.set('sort', next.sort)
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
                level: levelFilter >= 0 ? levelFilter : undefined,
                solved: solveFilter === 'all' ? undefined : solveFilter === 'solved',
                sort: sortFilter,
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
            const ranking = await api.rankingUsers(1, 10)
            setTopUsers(ranking.entries.slice(0, 10))
        } catch {
            setTopUsers([])
        }
    }

    useEffect(() => {
        void Promise.all([loadChallenges(page), loadTopUsers()])
    }, [page, appliedSearch, categoryFilter, levelFilter, solveFilter, sortFilter])

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setPage(state.page)
            setCategoryFilter(state.category)
            setLevelFilter(state.level)
            setSolveFilter(state.solved)
            setSortFilter(state.sort)
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

    useEffect(() => {
        if (!isSortMenuOpen) return

        const onPointerDown = (event: MouseEvent) => {
            if (sortMenuRef.current && !sortMenuRef.current.contains(event.target as Node)) {
                setIsSortMenuOpen(false)
            }
        }

        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') setIsSortMenuOpen(false)
        }

        window.addEventListener('mousedown', onPointerDown)
        window.addEventListener('keydown', onKeyDown)
        return () => {
            window.removeEventListener('mousedown', onPointerDown)
            window.removeEventListener('keydown', onKeyDown)
        }
    }, [isSortMenuOpen])

    return (
        <section className='animate space-y-4'>
            <div className='grid gap-4 lg:grid-cols-[1.9fr_0.9fr]'>
                <div className='space-y-3'>
                    <div className='space-y-2 shadow-none md:p-3'>
                        <div className='flex items-center gap-2 mb-4'>
                            <div className='relative flex-1'>
                                <span className='pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle dark:text-text-subtle'>⌕</span>
                                <input
                                    type='text'
                                    name='search'
                                    placeholder={t('challenges.searchPlaceholder')}
                                    value={searchQuery}
                                    onChange={(event) => setSearchQuery(event.target.value)}
                                    className='w-full rounded-lg border border-border/70 bg-surface py-2.5 pl-9 pr-3 text-sm text-text placeholder:text-text-subtle dark:border-border/70 dark:bg-surface dark:text-text dark:placeholder:text-text-subtle'
                                    onKeyDown={(event) => {
                                        if (event.key === 'Enter') {
                                            const nextQ = searchQuery.trim()
                                            setAppliedSearch(nextQ)
                                            setPage(1)
                                            pushQueryState({ q: nextQ, page: 1, category: categoryFilter, level: levelFilter, solved: solveFilter, sort: sortFilter })
                                        }
                                    }}
                                />
                            </div>
                            <button
                                type='button'
                                className='rounded-md border border-border/70 bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle dark:border-border/70 dark:text-text dark:hover:bg-surface-muted h-10'
                                onClick={() => {
                                    const nextQ = searchQuery.trim()
                                    setAppliedSearch(nextQ)
                                    setPage(1)
                                    pushQueryState({ q: nextQ, page: 1, category: categoryFilter, level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                    pushQueryState({ q: appliedSearch, page: 1, category: 'all', level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                        pushQueryState({ q: appliedSearch, page: 1, category, level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                                        pushQueryState({ q: appliedSearch, page: 1, category, level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                className={`rounded-md border px-3 py-1 text-xs ${levelFilter === ALL_LEVEL_FILTER ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                onClick={() => {
                                    setLevelFilter(ALL_LEVEL_FILTER)
                                    setPage(1)
                                    pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: ALL_LEVEL_FILTER, solved: solveFilter, sort: sortFilter })
                                }}
                            >
                                {t('common.all')}
                            </button>
                            <button
                                type='button'
                                onClick={() => {
                                    setLevelFilter(0)
                                    setPage(1)
                                    pushQueryState({
                                        q: appliedSearch,
                                        page: 1,
                                        category: categoryFilter,
                                        level: 0,
                                        solved: solveFilter,
                                        sort: sortFilter,
                                    })
                                }}
                                className='transition hover:scale-105'
                                aria-label={t('level.unknown')}
                            >
                                <LevelBadge level={0} active={levelFilter === 0} />
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
                                            sort: sortFilter,
                                        })
                                    }}
                                    className='transition hover:scale-105'
                                >
                                    <LevelBadge level={level} active={levelFilter === level} />
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
                                        pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: levelFilter, solved: key, sort: sortFilter })
                                    }}
                                >
                                    {key === 'all' ? t('challenges.filterAll') : key === 'solved' ? t('challenges.filterSolved') : t('challenges.filterUnsolved')}
                                </button>
                            ))}
                        </div>
                    </div>

                    <div className='flex items-center justify-between px-0 py-1 text-sm text-text-muted md:px-4 dark:text-text-muted'>
                        <span>{t('common.totalCount', { count: pagination.total_count })}</span>
                        <div className='relative' ref={sortMenuRef}>
                            <button
                                type='button'
                                className='inline-flex min-w-28 items-center justify-between gap-2 rounded-md border border-accent/20 px-3 py-1.5 text-xs text-text'
                                onClick={() => setIsSortMenuOpen((prev) => !prev)}
                                aria-haspopup='menu'
                                aria-expanded={isSortMenuOpen}
                            >
                                <span>{t(`challenges.sort.${sortFilter}`)}</span>
                                <svg className={`h-3 w-3 transition-transform ${isSortMenuOpen ? 'rotate-180' : ''}`} viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2'>
                                    <path d='m6 9 6 6 6-6' />
                                </svg>
                            </button>

                            {isSortMenuOpen ? (
                                <div className='absolute right-0 top-full z-20 mt-1 min-w-44 rounded-md border border-border bg-surface p-1 shadow-lg'>
                                    {(['latest', 'oldest', 'most_solved', 'least_solved'] as const).map((key) => (
                                        <button
                                            key={key}
                                            type='button'
                                            className={`block w-full rounded px-2 py-1.5 text-left text-xs ${sortFilter === key ? 'bg-accent/12 text-accent' : 'text-text-muted hover:bg-surface-muted hover:text-text'}`}
                                            onClick={() => {
                                                setSortFilter(key)
                                                setPage(1)
                                                setIsSortMenuOpen(false)
                                                pushQueryState({ q: appliedSearch, page: 1, category: categoryFilter, level: levelFilter, solved: solveFilter, sort: key })
                                            }}
                                        >
                                            {t(`challenges.sort.${key}`)}
                                        </button>
                                    ))}
                                </div>
                            ) : null}
                        </div>
                    </div>

                    <div className='-mx-4 md:mx-0 overflow-visible md:overflow-hidden rounded-none md:rounded-xl bg-transparent'>
                        {loading ? (
                            <>
                                <div className='space-y-2 px-4 md:hidden'>
                                    {Array.from({ length: CHALLENGE_SKELETON_ROWS }, (_, idx) => (
                                        <div key={`challenge-mobile-skeleton-${idx}`} className='w-full rounded-xl border border-border/60 bg-surface p-3'>
                                            <div className='flex items-start gap-3 animate-pulse'>
                                                <div className='h-8 w-8 shrink-0 rounded-full bg-surface-muted' />
                                                <div className='min-w-0 flex-1 space-y-2'>
                                                    <div className='h-4 w-3/5 rounded bg-surface-muted' />
                                                    <div className='h-3 w-1/3 rounded bg-surface-muted' />
                                                    <div className='mt-1 flex flex-wrap gap-x-4 gap-y-1'>
                                                        <div className='h-3 w-20 rounded bg-surface-muted' />
                                                        <div className='h-3 w-28 rounded bg-surface-muted' />
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>

                                <div className='hidden overflow-x-auto md:block'>
                                    <div className='min-w-150'>
                                        <div className='grid grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] bg-surface px-4 py-2 text-[12px] text-text-muted'>
                                            <span>{t('challenges.tableProblem')}</span>
                                            <span>{t('common.category')}</span>
                                            <span>{t('challenges.tableSolveCount')}</span>
                                            <span>{t('challenges.tableAuthor')}</span>
                                        </div>
                                        <div>
                                            {Array.from({ length: CHALLENGE_SKELETON_ROWS }, (_, idx) => (
                                                <div
                                                    key={`challenge-desktop-skeleton-${idx}`}
                                                    className='grid w-full grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] items-center px-4 py-3'
                                                >
                                                    <div className='min-w-0 flex items-center gap-3 animate-pulse'>
                                                        <div className='h-8 w-8 shrink-0 rounded-full bg-surface-muted' />
                                                        <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                                    </div>
                                                    <div className='h-3 w-2/5 rounded bg-surface-muted animate-pulse' />
                                                    <div className='h-3 w-10 rounded bg-surface-muted animate-pulse' />
                                                    <div className='h-3 w-4/5 rounded bg-surface-muted animate-pulse' />
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            </>
                        ) : errorMessage ? (
                            <div className='px-4 py-8 text-sm text-danger'>{errorMessage}</div>
                        ) : challenges.length === 0 ? (
                            <div className='px-4 py-8 text-sm text-text-muted'>{t('users.noResults')}</div>
                        ) : (
                            <>
                                <div className='space-y-2 px-4 md:hidden'>
                                    {challenges.map((challenge) => {
                                        const category = 'category' in challenge ? challenge.category : t('common.na')
                                        const solveCount = 'solve_count' in challenge ? challenge.solve_count : 0
                                        const inactive = challenge.is_active === false
                                        const authorName = challenge.created_by?.username?.trim()
                                        const author = authorName && authorName !== '' ? authorName : t('common.na')

                                        return (
                                            <button
                                                key={challenge.id}
                                                type='button'
                                                className='w-full rounded-xl border border-border/60 bg-surface p-3 text-left transition hover:bg-surface-muted disabled:cursor-not-allowed disabled:opacity-70'
                                                disabled={inactive}
                                                onClick={() => {
                                                    if (!inactive) {
                                                        navigate(`/challenges/${challenge.id}${window.location.search}`)
                                                    }
                                                }}
                                            >
                                                <div className='flex items-start gap-3'>
                                                    <LevelBadge level={challenge.level} />
                                                    <div className='min-w-0 flex-1'>
                                                        <div className='flex items-center gap-1.5'>
                                                            <p className='truncate text-sm font-semibold text-text'>{challenge.title}</p>
                                                            {challenge.is_locked ? (
                                                                <span className='shrink-0 h-4 w-4 text-warning' title={t('challenge.lockedLabel')}>
                                                                    <svg viewBox='0 0 24 24' className='h-full w-full' fill='none' stroke='currentColor' strokeWidth='2'>
                                                                        <rect x='5' y='11' width='14' height='9' rx='2' />
                                                                        <path d='M8 11V8a4 4 0 1 1 8 0v3' />
                                                                    </svg>
                                                                </span>
                                                            ) : challenge.is_solved ? (
                                                                <FlagIcon className='shrink-0 h-4 w-4 text-accent' />
                                                            ) : null}
                                                        </div>
                                                        <p className='mt-1 text-xs text-text-muted'>{t(getCategoryKey(category))}</p>
                                                        <div className='mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-subtle'>
                                                            <span>
                                                                {t('challenges.tableSolveCount')}: {solveCount}
                                                            </span>
                                                            <span className='truncate'>
                                                                {t('challenges.tableAuthor')}: {author}
                                                            </span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </button>
                                        )
                                    })}
                                </div>

                                <div className='hidden overflow-x-auto md:block'>
                                    <div className='min-w-150'>
                                        <div className='grid grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] bg-surface px-4 py-2 text-[12px] text-text-muted'>
                                            <span>{t('challenges.tableProblem')}</span>
                                            <span>{t('common.category')}</span>
                                            <span>{t('challenges.tableSolveCount')}</span>
                                            <span>{t('challenges.tableAuthor')}</span>
                                        </div>

                                        <div>
                                            {challenges.map((challenge) => {
                                                const category = 'category' in challenge ? challenge.category : t('common.na')
                                                const solveCount = 'solve_count' in challenge ? challenge.solve_count : 0
                                                const inactive = challenge.is_active === false
                                                const authorName = challenge.created_by?.username?.trim()
                                                const author = authorName && authorName !== '' ? authorName : t('common.na')

                                                return (
                                                    <button
                                                        key={challenge.id}
                                                        type='button'
                                                        className='grid w-full grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] items-center px-4 py-3 text-left transition hover:bg-surface-muted disabled:cursor-not-allowed disabled:opacity-70'
                                                        disabled={inactive}
                                                        onClick={() => {
                                                            if (!inactive) {
                                                                navigate(`/challenges/${challenge.id}${window.location.search}`)
                                                            }
                                                        }}
                                                    >
                                                        <div className='min-w-0 flex items-center gap-3'>
                                                            <LevelBadge level={challenge.level} />
                                                            <div className='min-w-0 flex flex-1 items-center'>
                                                                <span className='truncate pr-4 text-[14px] font-semibold sm:text-[16px]'>{challenge.title}</span>
                                                                {challenge.is_locked ? (
                                                                    <span className='-ml-1.5 h-4 w-4 shrink-0 text-warning' title={t('challenge.lockedLabel')}>
                                                                        <svg viewBox='0 0 24 24' className='h-full w-full' fill='none' stroke='currentColor' strokeWidth='2'>
                                                                            <rect x='5' y='11' width='14' height='9' rx='2' />
                                                                            <path d='M8 11V8a4 4 0 1 1 8 0v3' />
                                                                        </svg>
                                                                    </span>
                                                                ) : challenge.is_solved ? (
                                                                    <FlagIcon className='-ml-1.5 h-4 w-4 shrink-0 text-accent' />
                                                                ) : null}
                                                            </div>
                                                        </div>

                                                        <span className='wrap-break-words text-xs text-text-muted'>{t(getCategoryKey(category))}</span>
                                                        <span className='text-sm text-text-muted'>{solveCount}</span>
                                                        <span className='truncate text-xs text-text-muted'>{author}</span>
                                                    </button>
                                                )
                                            })}
                                        </div>
                                    </div>
                                </div>
                            </>
                        )}
                    </div>

                    <div className='flex flex-wrap items-center justify-end gap-2 px-0 py-2 text-sm text-text-muted md:px-4 dark:text-text-muted'>
                        <div className='flex items-center gap-2'>
                            <button
                                type='button'
                                className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50 dark:text-text dark:hover:bg-surface-muted'
                                disabled={!pagination.has_prev}
                                onClick={() => {
                                    const nextPage = Math.max(1, page - 1)
                                    setPage(nextPage)
                                    pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                    pushQueryState({ q: appliedSearch, page: nextPage, category: categoryFilter, level: levelFilter, solved: solveFilter, sort: sortFilter })
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
                                topUsers.map((entry, index) => {
                                    const affiliation = entry.affiliation_name?.trim()
                                    const bio = entry.bio?.trim()
                                    const profileLine = `${affiliation ? `${affiliation} · ` : ''}${bio && bio.length > 0 ? bio : t('profile.noBio')}`

                                    return (
                                        <button
                                            key={`top-user-${entry.user_id}-${index}`}
                                            type='button'
                                            className='group flex w-full items-start justify-between gap-3 rounded-lg px-2 py-2 text-left transition hover:bg-surface-muted dark:hover:bg-surface-muted'
                                            onClick={() => navigate(`/users/${entry.user_id}`)}
                                        >
                                            <div className='min-w-0 flex flex-1 items-center gap-3'>
                                                <UserAvatar username={entry.username} profileImage={entry.profile_image ?? null} size='sm' />
                                                <div className='min-w-0 flex-1'>
                                                    <div className='flex min-w-0 items-center gap-1.5'>
                                                        <span className='shrink-0 text-xs text-text-subtle'>#{index + 1}</span>
                                                        <span className='min-w-0 truncate text-sm font-semibold text-text'>{entry.username}</span>
                                                    </div>

                                                    <p className='mt-1 text-xs text-text-subtle line-clamp-1 break-all'>{profileLine}</p>
                                                </div>
                                            </div>

                                            <span className='shrink-0 pt-0.5 text-xs text-text-muted'>{t('common.pointsShort', { points: entry.score })}</span>
                                        </button>
                                    )
                                })
                            )}
                        </div>
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default Challenges
