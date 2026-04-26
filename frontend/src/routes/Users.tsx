import { useEffect, useMemo, useState } from 'react'
import type { PaginationMeta, UserListItem } from '../lib/types'
import { formatApiError } from '../lib/utils'
import { navigate } from '../lib/router'
import { getRoleKey, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import UserAvatar from '../components/UserAvatar'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const USER_SKELETON_ROWS = 5
const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const Users = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const [users, setUsers] = useState<UserListItem[]>([])
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const readQueryState = () => {
        if (typeof window === 'undefined') return { q: '', page: 1 }
        const params = new URLSearchParams(window.location.search)
        return {
            q: (params.get('q') ?? '').trim(),
            page: parsePositiveInt(params.get('page'), 1),
        }
    }
    const initialQueryState = readQueryState()
    const [searchQuery, setSearchQuery] = useState(initialQueryState.q)
    const [appliedSearch, setAppliedSearch] = useState(initialQueryState.q)
    const [page, setPage] = useState(initialQueryState.page)
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const pushQueryState = (next: { q: string; page: number }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }

    const loadUsers = async () => {
        setLoading(true)
        setErrorMessage('')

        try {
            const keyword = appliedSearch.trim()
            const response = keyword ? await api.searchUsers(keyword, page, PAGE_SIZE) : await api.users(page, PAGE_SIZE)
            setUsers(response.users)
            setPagination(response.pagination)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setPagination(EMPTY_PAGINATION)
        } finally {
            setLoading(false)
        }
    }

    const sortedUsers = useMemo(() => [...users].sort((a, b) => a.id - b.id), [users])

    useEffect(() => {
        void loadUsers()
    }, [page, appliedSearch])

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setPage(state.page)
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    return (
        <section className='animate space-y-4'>
            <div className='space-y-2 bg-transparent shadow-none md:bg-surface dark:bg-surface'>
                <h2 className='text-2xl font-semibold text-text dark:text-text'>{t('users.title')}</h2>

                <div className='mt-1'>
                    <input
                        type='text'
                        placeholder={t('users.searchPlaceholder')}
                        value={searchQuery}
                        onChange={(event) => setSearchQuery(event.target.value)}
                        className='w-full rounded-lg border border-border/70 bg-surface px-3 py-2.5 text-sm text-text placeholder:text-text-subtle focus:border-accent focus:outline-none dark:border-border/70 dark:bg-surface dark:text-text dark:placeholder:text-text-subtle'
                    />
                    <div className='mt-2 flex flex-wrap gap-2'>
                        <button
                            type='button'
                            className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle'
                            onClick={() => {
                                const nextQ = searchQuery.trim()
                                setAppliedSearch(nextQ)
                                setPage(1)
                                pushQueryState({ q: nextQ, page: 1 })
                            }}
                        >
                            {t('common.search')}
                        </button>
                        <button
                            type='button'
                            className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle dark:border-border/70 dark:hover:bg-surface-muted'
                            onClick={() => {
                                setSearchQuery('')
                                setAppliedSearch('')
                                setPage(1)
                                pushQueryState({ q: '', page: 1 })
                            }}
                        >
                            {t('common.reset')}
                        </button>
                    </div>
                </div>
            </div>

            <div className='-mx-4 md:mx-0 overflow-hidden rounded-none md:rounded-xl bg-transparent md:bg-surface md:shadow-sm'>
                {loading ? (
                    <>
                        <div className='divide-y divide-border/60 md:hidden'>
                            {Array.from({ length: USER_SKELETON_ROWS }, (_, idx) => (
                                <div key={`users-mobile-skeleton-${idx}`} className='px-4 py-3'>
                                    <div className='flex items-center justify-between gap-3 animate-pulse'>
                                        <div className='min-w-0 flex items-center gap-3.75'>
                                            <div className='h-10 w-10 shrink-0 rounded-full bg-surface-muted' />
                                            <div className='min-w-0 space-y-2'>
                                                <div className='h-4 w-28 rounded bg-surface-muted' />
                                                <div className='h-4 w-16 rounded bg-surface-muted' />
                                                <div className='h-3 w-32 rounded bg-surface-muted' />
                                                <div className='h-3 w-40 rounded bg-surface-muted' />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div className='hidden md:block'>
                            <div className='grid grid-cols-[80px_minmax(0,1fr)_180px] bg-surface-muted px-4 py-3 text-[12px] text-text-muted dark:bg-surface-muted dark:text-text-muted'>
                                <p className='font-medium'>{t('common.id')}</p>
                                <p className='font-medium'>{t('common.username')}</p>
                                <p className='font-medium'>{t('common.role')}</p>
                            </div>
                            {Array.from({ length: USER_SKELETON_ROWS }, (_, idx) => (
                                <div key={`users-desktop-skeleton-${idx}`} className='grid grid-cols-[80px_minmax(0,1fr)_180px] items-center px-4 py-4'>
                                    <div className='h-3 w-10 rounded bg-surface-muted animate-pulse' />
                                    <div className='flex items-center gap-3.75'>
                                        <div className='h-8 w-8 shrink-0 rounded-full bg-surface-muted animate-pulse' />
                                        <div className='min-w-0 flex-1 space-y-2'>
                                            <div className='h-3 w-24 rounded bg-surface-muted animate-pulse' />
                                            <div className='h-3 w-36 rounded bg-surface-muted animate-pulse' />
                                        </div>
                                    </div>
                                    <div className='h-4 w-14 rounded bg-surface-muted animate-pulse' />
                                </div>
                            ))}
                        </div>
                    </>
                ) : null}
                {!loading && errorMessage ? <p className='px-4 py-8 text-sm text-danger'>{errorMessage}</p> : null}
                {!loading && !errorMessage ? (
                    <>
                        <div className='divide-y divide-border/60 md:hidden'>
                            {sortedUsers.map((user) => (
                                <div key={user.id} className='px-4 py-3 cursor-pointer' onClick={() => navigate(`/users/${user.id}${window.location.search}`)}>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div className='min-w-0 flex items-center gap-3.75'>
                                            <UserAvatar username={user.username} size='md' />
                                            <div className='min-w-0'>
                                                <p className='truncate text-sm font-semibold text-text'>{user.username}</p>
                                                <p className='mt-1 text-xs text-text-muted bg-accent/10 inline-block rounded px-1.5 py-0.5 dark:bg-accent/20 dark:text-accent'>{t(getRoleKey(user.role))}</p>
                                                <p className='mt-1 truncate text-xs text-text-subtle'>{user.affiliation?.trim() ? user.affiliation : ''}</p>
                                                <p className='truncate text-xs text-text-subtle'>{user.bio ?? t('profile.noBio')}</p>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                            {sortedUsers.length === 0 ? <p className='px-6 py-8 text-center text-sm text-text-muted'>{appliedSearch ? t('users.noResults') : t('users.noUsers')}</p> : null}
                        </div>

                        <div className='hidden md:block'>
                            <div className='grid grid-cols-[80px_minmax(0,1fr)_180px] bg-surface-muted px-4 py-3 text-[12px] text-text-muted dark:bg-surface-muted dark:text-text-muted'>
                                <p className='font-medium'>{t('common.id')}</p>
                                <p className='font-medium'>{t('common.username')}</p>
                                <p className='font-medium'>{t('common.role')}</p>
                            </div>
                            {sortedUsers.map((user) => (
                                <div
                                    key={user.id}
                                    className='grid grid-cols-[80px_minmax(0,1fr)_180px] items-center px-4 py-4 transition hover:bg-surface-muted/40 dark:hover:bg-surface-muted cursor-pointer'
                                    onClick={() => navigate(`/users/${user.id}${window.location.search}`)}
                                >
                                    <p className='text-sm text-text dark:text-text'>{user.id}</p>
                                    <div className='flex items-center gap-3.75 truncate'>
                                        <UserAvatar username={user.username} size='sm' />
                                        <div className='min-w-0 pr-3'>
                                            <p className='truncate text-sm text-text dark:text-text'>{user.username}</p>
                                            <p className='truncate text-xs text-text-subtle'>{user.affiliation?.trim() ? user.affiliation : ''}</p>
                                            <p className='truncate text-xs text-text-subtle'>{user.bio ?? t('profile.noBio')}</p>
                                        </div>
                                    </div>
                                    <p className='text-xs text-text-muted dark:text-text-muted bg-accent/10 inline-block rounded px-1.5 py-0.5 w-max max-w-full whitespace-nowrap dark:bg-accent/20'>{t(getRoleKey(user.role))}</p>
                                </div>
                            ))}
                            {sortedUsers.length === 0 ? <p className='px-6 py-8 text-center text-sm text-text-muted dark:text-text-muted'>{appliedSearch ? t('users.noResults') : t('users.noUsers')}</p> : null}
                        </div>
                    </>
                ) : null}
            </div>

            <div className='mt-3 flex flex-wrap items-center justify-between gap-3 px-1 text-xs text-text-muted dark:text-text-muted'>
                <span>{t('common.totalCount', { count: pagination.total_count })}</span>
                <div className='flex items-center gap-2'>
                    <button
                        type='button'
                        className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50 dark:text-text dark:hover:bg-surface-muted'
                        disabled={!pagination.has_prev}
                        onClick={() => {
                            const nextPage = Math.max(1, page - 1)
                            setPage(nextPage)
                            pushQueryState({ q: appliedSearch, page: nextPage })
                        }}
                    >
                        {t('common.previous')}
                    </button>
                    <span className='text-xs text-text-muted'>
                        {pagination.page} / {pagination.total_pages || 1}
                    </span>
                    <button
                        type='button'
                        className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50 dark:text-text dark:hover:bg-surface-muted'
                        disabled={!pagination.has_next}
                        onClick={() => {
                            const nextPage = page + 1
                            setPage(nextPage)
                            pushQueryState({ q: appliedSearch, page: nextPage })
                        }}
                    >
                        {t('common.next')}
                    </button>
                </div>
            </div>
        </section>
    )
}

export default Users
