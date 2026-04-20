import { useEffect, useMemo, useState } from 'react'
import LoginRequired from '../components/LoginRequired'
import type { PaginationMeta, UserListItem } from '../lib/types'
import { formatApiError } from '../lib/utils'
import { navigate } from '../lib/router'
import { getRoleKey, useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const Users = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()
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
        if (!auth.user) return
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
        if (!auth.user) return
        void loadUsers()
    }, [auth.user, page, appliedSearch])

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

    if (!auth.user) {
        return <LoginRequired title={t('users.title')} />
    }

    return (
        <section className='animate space-y-4'>
            <div className='space-y-2 bg-transparent shadow-none md:bg-surface md:p-3 dark:bg-surface'>
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
                {loading ? <p className='px-4 py-8 text-sm text-text-muted dark:text-text-muted'>{t('common.loading')}</p> : null}
                {!loading && errorMessage ? <p className='px-4 py-8 text-sm text-danger'>{errorMessage}</p> : null}
                {!loading && !errorMessage ? (
                    <>
                        <div className='divide-y divide-border/60 md:hidden'>
                            {sortedUsers.map((user) => (
                                <div key={user.id} className='px-4 py-3'>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div className='min-w-0'>
                                            <p className='truncate text-sm font-semibold text-text'>{user.username}</p>
                                            <p className='mt-1 text-xs text-text-muted'>
                                                #{user.id} · {t(getRoleKey(user.role))}
                                            </p>
                                        </div>
                                        <button
                                            className='rounded-md border border-border bg-surface px-3 py-1 text-xs text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                                            onClick={() => navigate(`/users/${user.id}${window.location.search}`)}
                                            type='button'
                                        >
                                            {t('common.view')}
                                        </button>
                                    </div>
                                </div>
                            ))}
                            {sortedUsers.length === 0 ? <p className='px-6 py-8 text-center text-sm text-text-muted'>{appliedSearch ? t('users.noResults') : t('users.noUsers')}</p> : null}
                        </div>

                        <div className='hidden md:block'>
                            <div className='grid grid-cols-[110px_minmax(0,1fr)_180px_120px] bg-surface-muted px-4 py-3 text-[12px] text-text-muted dark:bg-surface-muted dark:text-text-muted'>
                                <p className='font-medium'>{t('common.id')}</p>
                                <p className='font-medium'>{t('common.username')}</p>
                                <p className='font-medium'>{t('common.role')}</p>
                                <p className='text-right font-medium'>{t('common.action')}</p>
                            </div>
                            {sortedUsers.map((user) => (
                                <div key={user.id} className='grid grid-cols-[110px_minmax(0,1fr)_180px_120px] items-center px-4 py-4 transition hover:bg-surface-muted/40 dark:hover:bg-surface-muted'>
                                    <p className='text-sm text-text dark:text-text'>{user.id}</p>
                                    <p className='truncate pr-3 text-sm text-text dark:text-text'>{user.username}</p>
                                    <p className='text-xs text-text-muted dark:text-text-muted'>{t(getRoleKey(user.role))}</p>
                                    <div className='text-right'>
                                        <button
                                            className='rounded-md border border-border bg-surface px-3 py-1 text-xs text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                                            onClick={() => navigate(`/users/${user.id}${window.location.search}`)}
                                            type='button'
                                        >
                                            {t('common.view')}
                                        </button>
                                    </div>
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
