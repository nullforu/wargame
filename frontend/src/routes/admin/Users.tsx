import { useCallback, useEffect, useMemo, useState } from 'react'
import type { AuthUser, UserListItem } from '../../lib/types'
import { useApi } from '../../lib/useApi'
import { formatApiError, formatDateTime } from '../../lib/utils'
import { getLocaleTag, getRoleKey, useLocale, useT } from '../../lib/i18n'
import FormMessage from '../../components/FormMessage'

const AdminUsers = () => {
    const t = useT()
    const api = useApi()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const [users, setUsers] = useState<UserListItem[]>([])
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const [searchQuery, setSearchQuery] = useState('')
    const [blockReasons, setBlockReasons] = useState<Record<number, string>>({})
    const [rowErrors, setRowErrors] = useState<Record<number, string>>({})
    const [blockingUserId, setBlockingUserId] = useState<number | null>(null)
    const [unblockingUserId, setUnblockingUserId] = useState<number | null>(null)

    const normalizedQuery = useMemo(() => searchQuery.trim().toLowerCase(), [searchQuery])
    const filteredUsers = useMemo(() => {
        const sorted = [...users].sort((a, b) => a.id - b.id)
        if (!normalizedQuery) return sorted
        return sorted.filter((user) => user.username.toLowerCase().includes(normalizedQuery) || user.id.toString().includes(normalizedQuery))
    }, [normalizedQuery, users])

    const formatOptionalDate = useCallback((value?: string | null) => (value ? formatDateTime(value, localeTag) : t('common.na')), [localeTag, t])

    const loadData = useCallback(async () => {
        setLoading(true)
        setErrorMessage('')
        setSuccessMessage('')
        setRowErrors({})

        try {
            const userRows = await api.users()
            setUsers(userRows)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }, [api, t])

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

    return (
        <section className='space-y-4'>
            <div className='flex items-center justify-between'>
                <button className='text-xs uppercase tracking-wide text-text-subtle hover:text-text cursor-pointer' onClick={loadData} disabled={loading}>
                    {loading ? t('common.loading') : t('common.refresh')}
                </button>
            </div>

            <div>
                <input
                    type='text'
                    placeholder={t('admin.users.searchPlaceholder')}
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    className='w-full rounded-xl border border-border bg-surface px-4 py-2.5 text-sm text-text placeholder-text-subtle transition focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20'
                />
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            {loading ? (
                <p className='text-sm text-text-muted'>{t('admin.users.loading')}</p>
            ) : (
                <div className='overflow-hidden rounded-2xl border border-border bg-surface'>
                    <div className='overflow-x-auto'>
                        <table className='w-full'>
                            <thead className='border-b border-border bg-surface-muted'>
                                <tr>
                                    <th className='px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted'>{t('common.id')}</th>
                                    <th className='px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted'>{t('common.user')}</th>
                                    <th className='px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted'>{t('common.role')}</th>
                                    <th className='px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted'>{t('admin.users.blockedLabel')}</th>
                                    <th className='px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted'>{t('common.action')}</th>
                                </tr>
                            </thead>
                            <tbody className='divide-y divide-border'>
                                {filteredUsers.map((user) => {
                                    const isBlocked = user.role === 'blocked'
                                    const rowError = rowErrors[user.id]
                                    const pendingBlock = blockingUserId === user.id
                                    const pendingUnblock = unblockingUserId === user.id

                                    return (
                                        <tr key={user.id} className='align-top'>
                                            <td className='whitespace-nowrap px-6 py-4 text-sm text-text'>{user.id}</td>
                                            <td className='px-6 py-4 text-sm text-text'>
                                                <div className='font-medium'>{user.username}</div>
                                            </td>
                                            <td className='px-6 py-4 text-sm'>
                                                <span
                                                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase ${
                                                        user.role === 'admin' ? 'bg-secondary/20 text-secondary' : user.role === 'blocked' ? 'bg-danger/20 text-danger' : 'bg-accent/20 text-accent-strong'
                                                    }`}
                                                >
                                                    {t(getRoleKey(user.role))}
                                                </span>
                                            </td>
                                            <td className='px-6 py-4 text-sm text-text'>
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
                                            </td>
                                            <td className='px-6 py-4 text-sm text-text'>
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
                                            </td>
                                        </tr>
                                    )
                                })}
                                {filteredUsers.length === 0 ? (
                                    <tr>
                                        <td colSpan={5} className='px-6 py-8 text-center text-sm text-text-muted'>
                                            {t('admin.users.noUsers')}
                                        </td>
                                    </tr>
                                ) : null}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </section>
    )
}

export default AdminUsers
