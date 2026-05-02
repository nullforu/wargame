import { useEffect, useMemo, useRef, useState } from 'react'
import { useApi } from '../lib/useApi'
import { useAuth } from '../lib/auth'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import type { CommunityPost, PaginationMeta } from '../lib/types'
import { formatApiError, formatDateTime } from '../lib/utils'

const PAGE_SIZE = 20
const NOTICE_SIZE = 3
const POPULAR_POST_LIKE_THRESHOLD = 5
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const COMMUNITY_SKELETON_ROWS = 5

const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const stringsEqualTrue = (value: string | null) => {
    const v = (value ?? '').trim().toLowerCase()
    return v === '1' || v === 'true'
}

export const categoryTextKey = (category: number) => {
    switch (category) {
        case 0:
            return 'community.category.notice'
        case 1:
            return 'community.category.free'
        case 2:
            return 'community.category.qna'
        case 3:
            return 'community.category.humor'
        default:
            return 'common.na'
    }
}

export const categoryBadgeClass = (category: number) => {
    switch (category) {
        case 0:
            return 'border-warning/40 bg-warning/10 text-warning'
        case 1:
            return 'border-info/30 bg-info/10 text-info'
        case 2:
            return 'border-danger/30 bg-danger/10 text-danger'
        case 3:
            return 'border-success/30 bg-success/12 text-success'
        default:
            return 'border-border/60 bg-surface-muted text-text-muted'
    }
}

const Community = () => {
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()

    const readQueryState = () => {
        if (typeof window === 'undefined') return { q: '', page: 1, category: '', sort: 'latest' as const, popularOnly: false }
        const params = new URLSearchParams(window.location.search)
        const sort = (params.get('sort') ?? '').trim()
        return {
            q: (params.get('q') ?? '').trim(),
            page: parsePositiveInt(params.get('page'), 1),
            category: (params.get('category') ?? '').trim(),
            sort: sort === 'oldest' || sort === 'popular' ? (sort as 'oldest' | 'popular') : ('latest' as const),
            popularOnly: stringsEqualTrue(params.get('popular')),
        }
    }

    const initialQueryState = readQueryState()
    const [posts, setPosts] = useState<CommunityPost[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    const [searchQuery, setSearchQuery] = useState(initialQueryState.q)
    const [appliedSearch, setAppliedSearch] = useState(initialQueryState.q)
    const [page, setPage] = useState(initialQueryState.page)
    const [category, setCategory] = useState(initialQueryState.category)
    const [sort, setSort] = useState<'latest' | 'oldest' | 'popular'>(initialQueryState.sort)
    const [popularOnly, setPopularOnly] = useState(initialQueryState.popularOnly)
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const [isSortMenuOpen, setIsSortMenuOpen] = useState(false)
    const sortMenuRef = useRef<HTMLDivElement | null>(null)

    const pushQueryState = (next: { q: string; page: number; category: string; sort: 'latest' | 'oldest' | 'popular'; popularOnly: boolean }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        if (next.category !== '') params.set('category', next.category)
        if (next.sort !== 'latest') params.set('sort', next.sort)
        if (next.popularOnly) params.set('popular', '1')
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) window.history.pushState({}, '', nextURL)
    }

    const scrollToTop = () => {
        if (typeof window !== 'undefined') {
            window.scrollTo(0, 0)
        }
    }

    const load = async (targetPage: number) => {
        setLoading(true)
        setError('')
        try {
            if (category === '' && !popularOnly) {
                const [noticeData, normalData] = await Promise.all([
                    api.communityPosts({ page: 1, pageSize: NOTICE_SIZE, sort: 'latest', category: 0 }),
                    api.communityPosts({ page: targetPage, pageSize: PAGE_SIZE, q: appliedSearch, sort, excludeNotice: true }),
                ])
                setPosts([...noticeData.posts, ...normalData.posts])
                setPagination({
                    ...normalData.pagination,
                    total_count: normalData.pagination.total_count + noticeData.pagination.total_count,
                })
            } else {
                const data = await api.communityPosts({ page: targetPage, pageSize: PAGE_SIZE, q: appliedSearch, sort, category: category === '' ? undefined : Number(category), popularOnly })
                setPosts(data.posts)
                setPagination(data.pagination)
            }
        } catch (e) {
            setError(formatApiError(e, t).message)
            setPagination(EMPTY_PAGINATION)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void load(page)
    }, [page, appliedSearch, category, sort, popularOnly])

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setPage(state.page)
            setCategory(state.category)
            setSort(state.sort)
            setPopularOnly(state.popularOnly)
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    useEffect(() => {
        if (!isSortMenuOpen) return
        const onPointerDown = (event: MouseEvent) => {
            if (sortMenuRef.current && !sortMenuRef.current.contains(event.target as Node)) setIsSortMenuOpen(false)
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

    const pageNumbers = useMemo(() => {
        const total = pagination.total_pages || 1
        const start = Math.max(1, page - 2)
        const end = Math.min(total, page + 2)
        const out: number[] = []
        for (let p = start; p <= end; p += 1) out.push(p)
        return out
    }, [page, pagination.total_pages])

    const changePage = (nextPage: number) => {
        setPage(nextPage)
        pushQueryState({ q: appliedSearch, page: nextPage, category, sort, popularOnly })
        scrollToTop()
    }

    return (
        <section className='animate space-y-4'>
            <div className='space-y-3'>
                <div className='space-y-2 shadow-none'>
                    <div className='flex items-center gap-2 mb-4'>
                        <div className='relative flex-1'>
                            <span className='pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle'>⌕</span>
                            <input
                                type='text'
                                name='search'
                                placeholder={t('community.searchPlaceholder')}
                                value={searchQuery}
                                onChange={(event) => setSearchQuery(event.target.value)}
                                className='w-full rounded-lg border border-border/70 bg-surface py-2.5 pl-9 pr-3 text-sm text-text placeholder:text-text-subtle'
                                onKeyDown={(event) => {
                                    if (event.key === 'Enter') {
                                        const nextQ = searchQuery.trim()
                                        setAppliedSearch(nextQ)
                                        setPage(1)
                                        pushQueryState({ q: nextQ, page: 1, category, sort, popularOnly })
                                    }
                                }}
                            />
                        </div>
                        <button
                            type='button'
                            className='rounded-md border border-border/70 bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle h-10'
                            onClick={() => {
                                const nextQ = searchQuery.trim()
                                setAppliedSearch(nextQ)
                                setPage(1)
                                pushQueryState({ q: nextQ, page: 1, category, sort, popularOnly })
                            }}
                        >
                            {t('common.search')}
                        </button>
                    </div>

                    <div className='flex items-end justify-between gap-2 flex-wrap'>
                        <div className='flex flex-wrap items-center gap-2'>
                            <span className='w-14 text-xs text-text-muted'>{t('common.category')}</span>
                            <button
                                type='button'
                                className={`rounded-md border px-3 py-1 text-xs ${category === '' && !popularOnly ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                onClick={() => {
                                    setCategory('')
                                    setPopularOnly(false)
                                    setPage(1)
                                    pushQueryState({ q: appliedSearch, page: 1, category: '', sort, popularOnly: false })
                                }}
                            >
                                {t('common.all')}
                            </button>
                            <button
                                type='button'
                                className={`rounded-md border px-3 py-1 text-xs ${popularOnly ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                onClick={() => {
                                    setPopularOnly(true)
                                    setCategory('')
                                    setPage(1)
                                    pushQueryState({ q: appliedSearch, page: 1, category: '', sort, popularOnly: true })
                                }}
                            >
                                {t('community.popularPosts')}
                            </button>
                            {[0, 1, 2, 3].map((c) => (
                                <button
                                    key={c}
                                    type='button'
                                    className={`rounded-md border px-3 py-1 text-xs ${category === String(c) ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                    onClick={() => {
                                        const next = String(c)
                                        setPopularOnly(false)
                                        setCategory(next)
                                        setPage(1)
                                        pushQueryState({ q: appliedSearch, page: 1, category: next, sort, popularOnly: false })
                                    }}
                                >
                                    {t(categoryTextKey(c))}
                                </button>
                            ))}
                        </div>

                        <div className='relative' ref={sortMenuRef}>
                            <button
                                type='button'
                                className='inline-flex min-w-28 items-center justify-between gap-2 rounded-md border border-accent/20 px-3 py-1.5 text-xs text-text'
                                onClick={() => setIsSortMenuOpen((prev) => !prev)}
                                aria-haspopup='menu'
                                aria-expanded={isSortMenuOpen}
                            >
                                <span>{t(`community.sort.${sort}`)}</span>
                                <svg className={`h-3 w-3 transition-transform ${isSortMenuOpen ? 'rotate-180' : ''}`} viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2'>
                                    <path d='m6 9 6 6 6-6' />
                                </svg>
                            </button>

                            {isSortMenuOpen ? (
                                <div className='absolute right-0 top-full z-20 mt-1 min-w-44 rounded-md border border-border bg-surface p-1 shadow-lg'>
                                    {(['latest', 'oldest', 'popular'] as const).map((key) => (
                                        <button
                                            key={key}
                                            type='button'
                                            className={`block w-full rounded px-2 py-1.5 text-left text-xs ${sort === key ? 'bg-accent/12 text-accent' : 'text-text-muted hover:bg-surface-muted hover:text-text'}`}
                                            onClick={() => {
                                                setSort(key)
                                                setPage(1)
                                                setIsSortMenuOpen(false)
                                                pushQueryState({ q: appliedSearch, page: 1, category, sort: key, popularOnly })
                                            }}
                                        >
                                            {t(`community.sort.${key}`)}
                                        </button>
                                    ))}
                                </div>
                            ) : null}
                        </div>
                    </div>
                </div>
            </div>

            <div className='-mx-4 md:mx-0 overflow-x-auto'>
                {loading ? (
                    <>
                        <div className='hidden md:grid grid-cols-[90px_60px_minmax(0,1fr)_100px_110px_90px_80px] items-center gap-3 px-4 py-3 text-xs font-medium text-text-muted'>
                            <p>{t('community.table.number')}</p>
                            <p>{t('common.category')}</p>
                            <p>{t('common.title')}</p>
                            <p>{t('common.username')}</p>
                            <p>{t('common.createdAt')}</p>
                            <p>{t('community.table.views')}</p>
                            <p>{t('community.table.likes')}</p>
                        </div>
                        {Array.from({ length: COMMUNITY_SKELETON_ROWS }, (_, idx) => (
                            <div key={`community-skeleton-${idx}`} className='hidden md:grid grid-cols-[90px_60px_minmax(0,1fr)_100px_110px_90px_80px] items-center gap-3 px-4 py-3 border-b border-border/60'>
                                <div className='h-4 w-10 rounded bg-surface-muted animate-pulse' />
                                <div className='h-5 w-18 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-3/4 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-2/3 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-2/3 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-8 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-6 rounded bg-surface-muted animate-pulse' />
                            </div>
                        ))}
                        <div className='divide-y divide-border/60 md:hidden'>
                            {Array.from({ length: COMMUNITY_SKELETON_ROWS }, (_, idx) => (
                                <div key={`community-mobile-skeleton-${idx}`} className='px-4 py-3'>
                                    <div className='animate-pulse space-y-2'>
                                        <div className='h-4 w-16 rounded bg-surface-muted' />
                                        <div className='h-4 w-11/12 rounded bg-surface-muted' />
                                        <div className='h-3 w-2/3 rounded bg-surface-muted' />
                                    </div>
                                </div>
                            ))}
                        </div>
                    </>
                ) : null}

                {!loading ? (
                    <>
                        <div className='hidden md:grid grid-cols-[90px_60px_minmax(0,1fr)_100px_110px_90px_80px] items-center gap-3 px-4 py-3 text-xs font-medium text-text-muted'>
                            <p>{t('community.table.number')}</p>
                            <p>{t('common.category')}</p>
                            <p>{t('common.title')}</p>
                            <p>{t('common.username')}</p>
                            <p>{t('common.createdAt')}</p>
                            <p>{t('community.table.views')}</p>
                            <p>{t('community.table.likes')}</p>
                        </div>

                        <div className='divide-y divide-border/60'>
                            {posts.map((post) => (
                                <div key={post.id}>
                                    <button
                                        className={`rounded-none hidden w-full md:grid grid-cols-[90px_60px_minmax(0,1fr)_100px_110px_90px_80px] items-center gap-3 px-4 py-3 text-left transition hover:bg-surface-muted/40 ${post.category === 0 ? 'bg-warning/10 hover:bg-warning/20' : ''}`}
                                        onClick={() => navigate(`/community/${post.id}${window.location.search}`)}
                                    >
                                        <p className='text-sm text-text-muted'>{post.id}</p>
                                        <p>
                                            <span className={`rounded-md inline-flex border px-2 py-0.5 text-[11px] font-medium ${categoryBadgeClass(post.category)}`}>{t(categoryTextKey(post.category))}</span>
                                        </p>
                                        <p className='flex min-w-0 items-center gap-1.5 text-sm font-medium text-text'>
                                            {post.like_count >= POPULAR_POST_LIKE_THRESHOLD ? (
                                                <svg viewBox='0 0 24 24' className='h-4 w-4 shrink-0 text-warning' fill='currentColor' aria-label='popular'>
                                                    <path d='M12 2.5l2.9 5.88 6.49.94-4.7 4.58 1.11 6.47L12 17.32l-5.8 3.05 1.1-6.47-4.69-4.58 6.49-.94L12 2.5Z' />
                                                </svg>
                                            ) : null}
                                            <span className='truncate'>{post.title}</span>
                                        </p>
                                        <p className='truncate text-sm text-text-muted'>{post.author.username}</p>
                                        <p className='text-sm text-text-muted'>{formatDateTime(post.created_at, localeTag).slice(0, 11)}</p>
                                        <p className='text-sm text-text'>{post.view_count}</p>
                                        <p className='text-sm text-text-muted'>{post.like_count}</p>
                                    </button>

                                    <button
                                        className={`rounded-none w-full px-4 py-3 text-left md:hidden ${post.category === 0 ? 'bg-warning/10 hover:bg-warning/20' : ''}`}
                                        onClick={() => navigate(`/community/${post.id}${window.location.search}`)}
                                    >
                                        <div className='flex items-center justify-between gap-2'>
                                            <span className={`rounded-md inline-flex border px-2 py-0.5 text-[11px] font-medium ${categoryBadgeClass(post.category)}`}>{t(categoryTextKey(post.category))}</span>
                                            <span className='text-xs text-text-subtle'>{formatDateTime(post.created_at, localeTag)}</span>
                                        </div>
                                        <p className='mt-1 flex items-center gap-1.5 text-sm font-semibold text-text'>
                                            {post.like_count >= POPULAR_POST_LIKE_THRESHOLD ? (
                                                <svg viewBox='0 0 24 24' className='h-4 w-4 shrink-0 text-warning' fill='currentColor' aria-label='popular'>
                                                    <path d='M12 2.5l2.9 5.88 6.49.94-4.7 4.58 1.11 6.47L12 17.32l-5.8 3.05 1.1-6.47-4.69-4.58 6.49-.94L12 2.5Z' />
                                                </svg>
                                            ) : null}
                                            <span className='line-clamp-1'>{post.title}</span>
                                        </p>
                                        <div className='mt-1 flex items-center gap-2 text-xs text-text-muted'>
                                            <span>{post.author.username}</span>
                                            <span>·</span>
                                            <span>{t('community.views', { count: post.view_count })}</span>
                                        </div>
                                    </button>
                                </div>
                            ))}
                        </div>

                        {error ? <p className='px-4 py-3 text-sm text-danger'>{error}</p> : null}
                        {!error && posts.length === 0 ? <p className='px-4 py-8 text-center text-sm text-text-muted'>{t('community.empty')}</p> : null}
                    </>
                ) : null}
            </div>

            <div className='space-y-3 pt-4'>
                <div className='flex items-center justify-between px-4'>
                    <span className='text-xs text-text-muted'>{t('common.totalCount', { count: pagination.total_count })}</span>
                    <div />
                </div>
                <div className='flex items-center justify-end mt-3'>
                    {auth.user ? (
                        <button className='rounded-md bg-accent px-3 py-2 text-sm font-medium text-white transition hover:bg-accent-strong' onClick={() => navigate(`/community/write${window.location.search}`)}>
                            {t('community.write')}
                        </button>
                    ) : (
                        <div className='text-sm text-text-muted'>{t('community.loginToWrite')}</div>
                    )}
                </div>
                <div className='flex items-center justify-center gap-1'>
                    <button
                        type='button'
                        className='px-3 py-1 text-xs text-text transition hover:bg-surface-muted disabled:opacity-50'
                        disabled={!pagination.has_prev}
                        onClick={() => {
                            if (!pagination.has_prev) return
                            const nextPage = Math.max(1, page - 1)
                            changePage(nextPage)
                        }}
                    >
                        {t('common.previous')}
                    </button>
                    {pageNumbers.map((p) => (
                        <button
                            key={`community-page-${p}`}
                            type='button'
                            className={`rounded-md px-2.5 py-1 text-xs ${p === page ? 'bg-accent text-white' : 'bg-surface-muted text-text hover:bg-surface-subtle'}`}
                            onClick={() => changePage(p)}
                        >
                            {p}
                        </button>
                    ))}
                    <button
                        type='button'
                        className='px-3 py-1 text-xs text-text transition hover:bg-surface-muted disabled:opacity-50'
                        disabled={!pagination.has_next}
                        onClick={() => {
                            if (!pagination.has_next) return
                            const nextPage = page + 1
                            changePage(nextPage)
                        }}
                    >
                        {t('common.next')}
                    </button>
                </div>
            </div>
        </section>
    )
}

export default Community
