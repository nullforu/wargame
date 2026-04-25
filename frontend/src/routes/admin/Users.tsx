import { useCallback, useEffect, useMemo, useState } from 'react'
import type { AuthUser, PaginationMeta, UserListItem } from '../../lib/types'
import { useApi } from '../../lib/useApi'
import { formatApiError, formatDateTime } from '../../lib/utils'
import { getLocaleTag, getRoleKey, useLocale, useT } from '../../lib/i18n'
import FormMessage from '../../components/FormMessage'

const AdminUsers = () => {
    const t = useT()
    const api = useApi()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    type RoleFilter = 'all' | 'admin' | 'user' | 'blocked'
    const [users, setUsers] = useState<UserListItem[]>([])
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const readQueryState = useCallback((): { q: string; page: number; role: RoleFilter } => {
        if (typeof window === 'undefined') return { q: '', page: 1, role: 'all' as RoleFilter }
        const params = new URLSearchParams(window.location.search)
        const parsedPage = Number(params.get('page'))
        const roleParam = params.get('role')
        return {
            q: (params.get('q') ?? '').trim(),
            page: Number.isInteger(parsedPage) && parsedPage > 0 ? parsedPage : 1,
            role: roleParam === 'admin' || roleParam === 'user' || roleParam === 'blocked' ? roleParam : 'all',
        }
    }, [])
    const initialQueryState = readQueryState()
    const [searchQuery, setSearchQuery] = useState(initialQueryState.q)
    const [appliedSearch, setAppliedSearch] = useState(initialQueryState.q)
    const [roleFilter, setRoleFilter] = useState<RoleFilter>(initialQueryState.role)
    const [page, setPage] = useState(initialQueryState.page)
    const [pagination, setPagination] = useState<PaginationMeta>({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
    const [blockReasons, setBlockReasons] = useState<Record<number, string>>({})
    const [rowErrors, setRowErrors] = useState<Record<number, string>>({})
    const [blockingUserId, setBlockingUserId] = useState<number | null>(null)
    const [unblockingUserId, setUnblockingUserId] = useState<number | null>(null)

    const filteredUsers = useMemo(() => {
        const sorted = [...users].sort((a, b) => a.id - b.id)
        if (roleFilter === 'all') return sorted
        return sorted.filter((user) => user.role === roleFilter)
    }, [users, roleFilter])

    const pushQueryState = useCallback((next: { q: string; page: number; role: RoleFilter }) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (next.q.trim() !== '') params.set('q', next.q.trim())
        if (next.page > 1) params.set('page', String(next.page))
        if (next.role !== 'all') params.set('role', next.role)
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }, [])

    const formatOptionalDate = useCallback((value?: string | null) => (value ? formatDateTime(value, localeTag) : t('common.na')), [localeTag, t])

    const loadData = useCallback(async () => {
        setLoading(true)
        setErrorMessage('')
        setSuccessMessage('')
        setRowErrors({})

        try {
            const response = appliedSearch ? await api.searchUsers(appliedSearch, page, 20) : await api.users(page, 20)
            setUsers(response.users)
            setPagination(response.pagination)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setPagination({ page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
        } finally {
            setLoading(false)
        }
    }, [api, t, appliedSearch, page])

    const updateUserRow = useCallback((updated: AuthUser) => {
        setUsers((prev) => prev.map((user) => (user.id === updated.id ? { ...user, ...updated } : user)))
    }, [])

    const handleBlockUser = useCallback(
        async (user: UserListItem) => {
            if (blockingUserId !== null) return
            if (user.role === 'admin') return

            const reason = blockReasons[user.id]?.trim()
            if (!reason) {
                setRowErrors((prev) => ({ ...prev, [user.id]: t('errors.required') }))
                return
            }

            setBlockingUserId(user.id)
            setRowErrors((prev) => ({ ...prev, [user.id]: '' }))
            setErrorMessage('')
            setSuccessMessage('')

            try {
                const updated = await api.blockUser(user.id, reason)
                updateUserRow(updated)
                setBlockReasons((prev) => ({ ...prev, [user.id]: '' }))
                setSuccessMessage(t('admin.users.blockedSuccess', { username: updated.username }))
            } catch (error) {
                setRowErrors((prev) => ({ ...prev, [user.id]: formatApiError(error, t).message }))
            } finally {
                setBlockingUserId(null)
            }
        },
        [api, blockingUserId, blockReasons, t, updateUserRow],
    )

    const handleUnblockUser = useCallback(
        async (user: UserListItem) => {
            if (unblockingUserId !== null) return
            if (user.role === 'admin') return

            setUnblockingUserId(user.id)
            setRowErrors((prev) => ({ ...prev, [user.id]: '' }))
            setErrorMessage('')
            setSuccessMessage('')

            try {
                const updated = await api.unblockUser(user.id)
                updateUserRow(updated)
                setSuccessMessage(t('admin.users.unblockedSuccess', { username: updated.username }))
            } catch (error) {
                setRowErrors((prev) => ({ ...prev, [user.id]: formatApiError(error, t).message }))
            } finally {
                setUnblockingUserId(null)
            }
        },
        [api, t, unblockingUserId, updateUserRow],
    )

    useEffect(() => {
        loadData()
    }, [loadData])

    useEffect(() => {
        const onPopState = () => {
            const state = readQueryState()
            setSearchQuery(state.q)
            setAppliedSearch(state.q)
            setRoleFilter(state.role as RoleFilter)
            setPage(state.page)
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [readQueryState])

    return (
        <section className='space-y-4'>
            <div className='flex items-center justify-between'>
                <button className='text-xs uppercase tracking-wide text-text-subtle hover:text-text cursor-pointer' onClick={loadData} disabled={loading}>
                    {loading ? t('common.loading') : t('common.refresh')}
                </button>
            </div>

            <div className='space-y-2 bg-transparent shadow-none md:bg-surface md:p-3 dark:bg-surface'>
                <input
                    type='text'
                    placeholder={t('admin.users.searchPlaceholder')}
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    className='w-full rounded-lg border border-border/70 bg-surface px-4 py-2.5 text-sm text-text placeholder-text-subtle transition focus:border-accent focus:outline-none'
                />
                <div className='flex gap-2'>
                    <button
                        type='button'
                        className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle'
                        onClick={() => {
                            const nextQ = searchQuery.trim()
                            setAppliedSearch(nextQ)
                            setPage(1)
                            pushQueryState({ q: nextQ, page: 1, role: roleFilter })
                        }}
                    >
                        {t('common.search')}
                    </button>
                    <button
                        type='button'
                        className='rounded-md border border-border/70 bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle'
                        onClick={() => {
                            setSearchQuery('')
                            setAppliedSearch('')
                            setRoleFilter('all')
                            setPage(1)
                            pushQueryState({ q: '', page: 1, role: 'all' })
                        }}
                    >
                        {t('common.reset')}
                    </button>
                </div>
                <div className='flex flex-wrap items-center gap-2 pt-1'>
                    <span className='w-14 text-xs text-text-muted'>{t('common.role')}</span>
                    {(['all', 'admin', 'user', 'blocked'] as const).map((key) => (
                        <button
                            key={key}
                            type='button'
                            className={`rounded-md border px-3 py-1 text-xs ${roleFilter === key ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                            onClick={() => {
                                setRoleFilter(key)
                                setPage(1)
                                pushQueryState({ q: appliedSearch, page: 1, role: key })
                            }}
                        >
                            {key === 'all' ? t('common.all') : t(getRoleKey(key))}
                        </button>
                    ))}
                </div>
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            {loading ? (
                <p className='text-sm text-text-muted'>{t('admin.users.loading')}</p>
            ) : (
                <div className='-mx-4 space-y-2 px-4 md:mx-0 md:space-y-0 md:px-0'>
                    <div className='space-y-2 md:hidden'>
                        {filteredUsers.map((user) => {
                            const isBlocked = user.role === 'blocked'
                            const rowError = rowErrors[user.id]
                            const pendingBlock = blockingUserId === user.id
                            const pendingUnblock = unblockingUserId === user.id

                            return (
                                <div key={user.id} className='rounded-xl border border-border/70 bg-surface p-3'>
                                    <div className='flex items-start justify-between gap-3'>
                                        <div className='min-w-0'>
                                            <p className='text-xs text-text-subtle'>#{user.id}</p>
                                            <p className='truncate text-sm font-semibold text-text'>{user.username}</p>
                                        </div>
                                        <span
                                            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase ${
                                                user.role === 'admin' ? 'bg-secondary/20 text-secondary' : user.role === 'blocked' ? 'bg-danger/20 text-danger' : 'bg-accent/20 text-accent-strong'
                                            }`}
                                        >
                                            {t(getRoleKey(user.role))}
                                        </span>
                                    </div>

                                    <div className='mt-2 text-sm text-text'>
                                        {isBlocked ? (
                                            <div className='space-y-1'>
                                                <p className='text-sm font-medium text-danger'>{t('admin.users.blockedStatus')}</p>
                                                <p className='text-xs text-text-subtle'>
                                                    {t('admin.users.blockedReasonLabel')}: {user.blocked_reason ?? t('common.na')}
                                                </p>
                                                <p className='text-xs text-text-subtle'>
                                                    {t('admin.users.blockedAtLabel')}: {formatOptionalDate(user.blocked_at)}
                                                </p>
                                            </div>
                                        ) : (
                                            <p className='text-xs text-text-subtle'>{t('admin.users.activeStatus')}</p>
                                        )}
                                    </div>

                                    <div className='mt-3 text-sm text-text'>
                                        {user.role === 'admin' ? (
                                            <p className='text-xs text-text-subtle'>{t('admin.users.adminLocked')}</p>
                                        ) : isBlocked ? (
                                            <button
                                                className='w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-accent/40 hover:text-accent disabled:cursor-not-allowed disabled:opacity-60'
                                                onClick={() => handleUnblockUser(user)}
                                                disabled={pendingUnblock}
                                                type='button'
                                            >
                                                {pendingUnblock ? t('admin.users.unblocking') : t('admin.users.unblockUser')}
                                            </button>
                                        ) : (
                                            <div className='space-y-2'>
                                                <input
                                                    type='text'
                                                    placeholder={t('admin.users.reasonPlaceholder')}
                                                    value={blockReasons[user.id] ?? ''}
                                                    onChange={(event) =>
                                                        setBlockReasons((prev) => ({
                                                            ...prev,
                                                            [user.id]: event.target.value,
                                                        }))
                                                    }
                                                    className='w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text placeholder-text-subtle focus:border-accent focus:outline-none'
                                                />
                                                <button
                                                    className='w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-accent/40 hover:text-accent disabled:cursor-not-allowed disabled:opacity-60'
                                                    onClick={() => handleBlockUser(user)}
                                                    disabled={pendingBlock}
                                                    type='button'
                                                >
                                                    {pendingBlock ? t('admin.users.blocking') : t('admin.users.blockUser')}
                                                </button>
                                            </div>
                                        )}
                                        {rowError ? <p className='mt-2 text-xs text-danger'>{rowError}</p> : null}
                                    </div>
                                </div>
                            )
                        })}
                        {filteredUsers.length === 0 ? <p className='px-2 py-6 text-center text-sm text-text-muted'>{t('admin.users.noUsers')}</p> : null}
                    </div>

                    <div className='hidden overflow-visible rounded-none bg-transparent md:block md:overflow-hidden md:rounded-xl md:bg-surface md:shadow-sm'>
                        <div className='grid grid-cols-[80px_minmax(0,1fr)_140px_minmax(220px,1fr)_320px] bg-surface-muted px-6 py-3 text-[12px] text-text-muted'>
                            <p className='font-medium'>{t('common.id')}</p>
                            <p className='font-medium'>{t('common.user')}</p>
                            <p className='font-medium'>{t('common.role')}</p>
                            <p className='font-medium'>{t('admin.users.blockedLabel')}</p>
                            <p className='font-medium'>{t('common.action')}</p>
                        </div>
                        {filteredUsers.map((user) => {
                            const isBlocked = user.role === 'blocked'
                            const rowError = rowErrors[user.id]
                            const pendingBlock = blockingUserId === user.id
                            const pendingUnblock = unblockingUserId === user.id

                            return (
                                <div key={user.id} className='grid grid-cols-[80px_minmax(0,1fr)_140px_minmax(220px,1fr)_320px] items-start px-6 py-5 transition hover:bg-surface-muted/40'>
                                    <p className='whitespace-nowrap text-sm text-text'>{user.id}</p>
                                    <div className='min-w-0 pr-4 text-sm text-text'>
                                        <p className='truncate font-medium'>{user.username}</p>
                                    </div>
                                    <div className='text-sm'>
                                        <span
                                            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase ${
                                                user.role === 'admin' ? 'bg-secondary/20 text-secondary' : user.role === 'blocked' ? 'bg-danger/20 text-danger' : 'bg-accent/20 text-accent-strong'
                                            }`}
                                        >
                                            {t(getRoleKey(user.role))}
                                        </span>
                                    </div>
                                    <div className='pr-4 text-sm text-text'>
                                        {isBlocked ? (
                                            <div className='space-y-1'>
                                                <p className='text-sm font-medium text-danger'>{t('admin.users.blockedStatus')}</p>
                                                <p className='text-xs text-text-subtle'>
                                                    {t('admin.users.blockedReasonLabel')}: {user.blocked_reason ?? t('common.na')}
                                                </p>
                                                <p className='text-xs text-text-subtle'>
                                                    {t('admin.users.blockedAtLabel')}: {formatOptionalDate(user.blocked_at)}
                                                </p>
                                            </div>
                                        ) : (
                                            <p className='text-xs text-text-subtle'>{t('admin.users.activeStatus')}</p>
                                        )}
                                    </div>
                                    <div className='text-sm text-text'>
                                        {user.role === 'admin' ? (
                                            <p className='text-xs text-text-subtle'>{t('admin.users.adminLocked')}</p>
                                        ) : isBlocked ? (
                                            <button
                                                className='rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-accent/40 hover:text-accent disabled:cursor-not-allowed disabled:opacity-60'
                                                onClick={() => handleUnblockUser(user)}
                                                disabled={pendingUnblock}
                                                type='button'
                                            >
                                                {pendingUnblock ? t('admin.users.unblocking') : t('admin.users.unblockUser')}
                                            </button>
                                        ) : (
                                            <div className='space-y-2'>
                                                <input
                                                    type='text'
                                                    placeholder={t('admin.users.reasonPlaceholder')}
                                                    value={blockReasons[user.id] ?? ''}
                                                    onChange={(event) =>
                                                        setBlockReasons((prev) => ({
                                                            ...prev,
                                                            [user.id]: event.target.value,
                                                        }))
                                                    }
                                                    className='w-full min-w-50 rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text placeholder-text-subtle focus:border-accent focus:outline-none'
                                                />
                                                <button
                                                    className='rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-accent/40 hover:text-accent disabled:cursor-not-allowed disabled:opacity-60'
                                                    onClick={() => handleBlockUser(user)}
                                                    disabled={pendingBlock}
                                                    type='button'
                                                >
                                                    {pendingBlock ? t('admin.users.blocking') : t('admin.users.blockUser')}
                                                </button>
                                            </div>
                                        )}
                                        {rowError ? <p className='mt-2 text-xs text-danger'>{rowError}</p> : null}
                                    </div>
                                </div>
                            )
                        })}
                        {filteredUsers.length === 0 ? <p className='px-6 py-8 text-center text-sm text-text-muted'>{t('admin.users.noUsers')}</p> : null}
                    </div>
                </div>
            )}
            <div className='mt-3 flex items-center justify-end gap-2 px-1 text-xs text-text-muted'>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-50'
                    disabled={!pagination.has_prev}
                    onClick={() => {
                        const nextPage = Math.max(1, page - 1)
                        setPage(nextPage)
                        pushQueryState({ q: appliedSearch, page: nextPage, role: roleFilter })
                    }}
                >
                    {t('common.previous')}
                </button>
                <span className='text-xs text-text-muted'>
                    {pagination.page} / {pagination.total_pages || 1}
                </span>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-50'
                    disabled={!pagination.has_next}
                    onClick={() => {
                        const nextPage = page + 1
                        setPage(nextPage)
                        pushQueryState({ q: appliedSearch, page: nextPage, role: roleFilter })
                    }}
                >
                    {t('common.next')}
                </button>
            </div>
        </section>
    )
}

export default AdminUsers
