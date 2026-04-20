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
        <section className='animate space-y-3'>
            <div className='rounded-none border-0 bg-transparent p-0 shadow-none'>
                <h2 className='text-2xl font-semibold text-text dark:text-text'>{t('users.title')}</h2>

                <div className='mt-3 flex flex-wrap gap-2'>
                    <input
                        type='text'
                        placeholder={t('users.searchPlaceholder')}
                        value={searchQuery}
                        onChange={(event) => setSearchQuery(event.target.value)}
                        className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-subtle focus:border-accent focus:outline-none sm:max-w-96 dark:border-border dark:bg-surface dark:text-text dark:placeholder:text-text-subtle'
                    />
                    <button
                        type='button'
                        className='flex-1 rounded-md border border-accent bg-accent px-4 py-2 text-sm text-white transition hover:bg-accent-strong sm:flex-none'
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
                        className='flex-1 rounded-md border border-border bg-surface px-4 py-2 text-sm text-text-muted transition hover:bg-surface-muted sm:flex-none dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
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

            <div className='overflow-hidden rounded-none border-0 bg-transparent shadow-none md:rounded-xl md:border md:border-border md:bg-surface dark:bg-surface dark:border-border'>
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

                        <div className='hidden overflow-x-auto md:block'>
                            <table className='w-full'>
                                <thead className='border-b border-border bg-surface-muted dark:border-border dark:bg-surface-muted'>
                                    <tr>
                                        <th className='px-4 py-2 text-left text-[12px] font-medium text-text-muted dark:text-text-muted'>{t('common.id')}</th>
                                        <th className='px-4 py-2 text-left text-[12px] font-medium text-text-muted dark:text-text-muted'>{t('common.username')}</th>
                                        <th className='px-4 py-2 text-left text-[12px] font-medium text-text-muted dark:text-text-muted'>{t('common.role')}</th>
                                        <th className='px-4 py-2 text-right text-[12px] font-medium text-text-muted dark:text-text-muted'>{t('common.action')}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {sortedUsers.map((user) => (
                                        <tr key={user.id} className='border-b border-border hover:bg-surface-muted dark:border-border dark:hover:bg-surface-muted'>
                                            <td className='px-4 py-3 text-sm text-text dark:text-text'>{user.id}</td>
                                            <td className='px-4 py-3 text-sm text-text dark:text-text'>{user.username}</td>
                                            <td className='px-4 py-3 text-xs text-text-muted dark:text-text-muted'>{t(getRoleKey(user.role))}</td>
                                            <td className='px-4 py-3 text-right'>
                                                <button
                                                    className='rounded-md border border-border bg-surface px-3 py-1 text-xs text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                                                    onClick={() => navigate(`/users/${user.id}${window.location.search}`)}
                                                    type='button'
                                                >
                                                    {t('common.view')}
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                    {sortedUsers.length === 0 ? (
                                        <tr>
                                            <td colSpan={4} className='px-6 py-8 text-center text-sm text-text-muted dark:text-text-muted'>
                                                {appliedSearch ? t('users.noResults') : t('users.noUsers')}
                                            </td>
                                        </tr>
                                    ) : null}
                                </tbody>
                            </table>
                        </div>
                    </>
                ) : null}
            </div>

            <div className='flex flex-wrap items-center justify-between gap-2 rounded-none border-0 bg-transparent px-0 py-2 text-sm text-text-muted shadow-none md:rounded-xl md:border md:border-border md:bg-surface md:px-4 dark:bg-surface dark:border-border dark:text-text-muted'>
                <span>{t('common.totalCount', { count: pagination.total_count })}</span>
                <div className='flex items-center gap-2'>
                    <button
                        type='button'
                        className='rounded-md border border-border bg-surface px-3 py-1 text-xs text-text-muted transition hover:bg-surface-muted disabled:opacity-50 dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                        disabled={!pagination.has_prev}
                        onClick={() => {
                            const nextPage = Math.max(1, page - 1)
                            setPage(nextPage)
                            pushQueryState({ q: appliedSearch, page: nextPage })
                        }}
                    >
                        {t('common.previous')}
                    </button>
                    <span className='text-xs'>
                        {pagination.page} / {pagination.total_pages || 1}
                    </span>
                    <button
                        type='button'
                        className='rounded-md border border-border bg-surface px-3 py-1 text-xs text-text-muted transition hover:bg-surface-muted disabled:opacity-50 dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
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
