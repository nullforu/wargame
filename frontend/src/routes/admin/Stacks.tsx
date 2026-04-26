import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import type { AdminStackListItem, Stack } from '../../lib/types'
import { useApi } from '../../lib/useApi'
import { formatApiError, formatDateTime } from '../../lib/utils'
import { getLocaleTag, useLocale, useT } from '../../lib/i18n'
import FormMessage from '../../components/FormMessage'

const STACK_SKELETON_ROWS = 5

const AdminStacks = () => {
    const t = useT()
    const api = useApi()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const [stacks, setStacks] = useState<AdminStackListItem[]>([])
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const [detailById, setDetailById] = useState<Record<string, Stack>>({})
    const [detailLoadingId, setDetailLoadingId] = useState<string | null>(null)
    const [detailErrorById, setDetailErrorById] = useState<Record<string, string>>({})
    const [deleteLoadingId, setDeleteLoadingId] = useState<string | null>(null)
    const formatTargetPorts = useCallback((ports: Stack['ports']) => (ports.length > 0 ? ports.map((port) => `${port.container_port}/${port.protocol}`).join(', ') : t('common.pending')), [t])
    const formatNodePorts = useCallback((ports: Stack['ports']) => (ports.length > 0 ? ports.map((port) => `${port.protocol} ${port.node_port}`).join(', ') : t('common.pending')), [t])
    const formatEndpoints = useCallback((detail: Stack) => (detail.node_public_ip && detail.ports.length > 0 ? detail.ports.map((port) => `${port.protocol} ${detail.node_public_ip}:${port.node_port}`).join(', ') : t('common.pending')), [t])

    const formatOptionalDate = useCallback((value?: string | null) => (value ? formatDateTime(value, localeTag) : t('common.na')), [localeTag, t])
    const formatCompactDateTime = useCallback(
        (value: string) =>
            new Intl.DateTimeFormat(localeTag, {
                year: '2-digit',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                hour12: false,
            }).format(new Date(value)),
        [localeTag],
    )

    const loadStacks = useCallback(async () => {
        setLoading(true)
        setErrorMessage('')
        setSuccessMessage('')

        try {
            const response = await api.adminStacks()
            const sorted = [...response.stacks].sort((a, b) => b.created_at.localeCompare(a.created_at))
            setStacks(sorted)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }, [api, t])

    const toggleDetails = useCallback(
        async (stackId: string) => {
            if (detailLoadingId) return
            if (detailById[stackId]) {
                setDetailById((prev) => {
                    const next = { ...prev }
                    delete next[stackId]
                    return next
                })
                return
            }

            setDetailLoadingId(stackId)
            setDetailErrorById((prev) => ({ ...prev, [stackId]: '' }))

            try {
                const detail = await api.adminStack(stackId)
                setDetailById((prev) => ({ ...prev, [stackId]: detail }))
            } catch (error) {
                setDetailErrorById((prev) => ({ ...prev, [stackId]: formatApiError(error, t).message }))
            } finally {
                setDetailLoadingId(null)
            }
        },
        [api, detailById, detailLoadingId, t],
    )

    const deleteStack = useCallback(
        async (stackId: string) => {
            if (deleteLoadingId) return
            const confirmed = window.confirm(t('admin.stacks.confirmDelete', { stack_id: stackId }))
            if (!confirmed) return

            setDeleteLoadingId(stackId)
            setErrorMessage('')
            setSuccessMessage('')

            try {
                await api.deleteAdminStack(stackId)
                setSuccessMessage(t('admin.stacks.deleted', { stack_id: stackId }))
                setStacks((prev) => prev.filter((stack) => stack.stack_id !== stackId))
                setDetailById((prev) => {
                    const next = { ...prev }
                    delete next[stackId]
                    return next
                })
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                setDeleteLoadingId(null)
            }
        },
        [api, deleteLoadingId, t],
    )

    useEffect(() => {
        loadStacks()
    }, [loadStacks])

    return (
        <section className='space-y-4'>
            <div className='flex flex-wrap items-center justify-between gap-3'>
                <button className='text-xs uppercase tracking-wide text-text-subtle hover:text-text disabled:opacity-60 cursor-pointer' onClick={loadStacks} disabled={loading}>
                    {loading ? t('common.loading') : t('common.refresh')}
                </button>
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            {loading ? (
                <div className='-mx-4 space-y-2 px-4 md:mx-0 md:space-y-0 md:px-0'>
                    <div className='space-y-2 md:hidden'>
                        {Array.from({ length: STACK_SKELETON_ROWS }, (_, idx) => (
                            <div key={`admin-stacks-mobile-skeleton-${idx}`} className='rounded-xl border border-border/70 bg-surface p-3'>
                                <div className='animate-pulse space-y-2'>
                                    <div className='h-3 w-32 rounded bg-surface-muted' />
                                    <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                    <div className='h-3 w-1/2 rounded bg-surface-muted' />
                                    <div className='h-8 w-full rounded bg-surface-muted' />
                                </div>
                            </div>
                        ))}
                    </div>

                    <div className='hidden overflow-visible rounded-none bg-transparent md:block md:overflow-hidden md:rounded-xl md:bg-surface md:shadow-sm'>
                        <div className='overflow-x-auto'>
                            <div className='min-w-280'>
                                <div className='grid min-w-280 grid-cols-[150px_minmax(170px,1fr)_170px_160px_160px_160px_120px] bg-surface-muted px-6 py-3 text-[12px] text-text-muted'>
                                    <p className='font-medium whitespace-nowrap'>{t('common.id')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.challengeLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.userLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.ttlLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.createdAt')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.updatedAt')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.action')}</p>
                                </div>
                                {Array.from({ length: STACK_SKELETON_ROWS }, (_, idx) => (
                                    <div key={`admin-stacks-desktop-skeleton-${idx}`} className='grid min-w-280 grid-cols-[150px_minmax(170px,1fr)_170px_160px_160px_160px_120px] items-start px-6 py-4'>
                                        <div className='h-3 w-24 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-4 w-2/3 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-3 w-1/2 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-3 w-16 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-3 w-16 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-3 w-16 rounded bg-surface-muted animate-pulse' />
                                        <div className='h-7 w-full rounded bg-surface-muted animate-pulse' />
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                </div>
            ) : stacks.length === 0 ? (
                <p className='text-sm text-text-muted'>{t('admin.stacks.noStacks')}</p>
            ) : (
                <div className='-mx-4 space-y-2 px-4 md:mx-0 md:space-y-0 md:px-0'>
                    <div className='space-y-2 md:hidden'>
                        {stacks.map((stack) => {
                            const detail = detailById[stack.stack_id]
                            const detailError = detailErrorById[stack.stack_id]
                            const detailsOpen = !!detail
                            const detailLoading = detailLoadingId === stack.stack_id
                            const deleteLoading = deleteLoadingId === stack.stack_id

                            return (
                                <div key={stack.stack_id} className='rounded-xl border border-border/70 bg-surface p-3'>
                                    <p className='break-all font-mono text-xs text-text'>#{stack.stack_id}</p>
                                    <p className='mt-1 truncate text-sm font-medium text-text'>{stack.challenge_title}</p>
                                    <p className='text-xs text-text-subtle'>
                                        {stack.challenge_category} · #{stack.challenge_id}
                                    </p>
                                    <p className='mt-2 text-sm text-text'>{stack.username}</p>
                                    <p className='text-xs text-text-subtle truncate'>{stack.email}</p>
                                    <div className='mt-2 space-y-1 text-xs text-text-subtle'>
                                        <p>
                                            {t('admin.stacks.ttlLabel')}: {stack.ttl_expires_at ? formatCompactDateTime(stack.ttl_expires_at) : t('common.na')}
                                        </p>
                                        <p>
                                            {t('common.createdAt')}: {formatCompactDateTime(stack.created_at)}
                                        </p>
                                        <p>
                                            {t('common.updatedAt')}: {formatCompactDateTime(stack.updated_at)}
                                        </p>
                                    </div>
                                    <div className='mt-3 flex gap-2'>
                                        <button
                                            className='flex-1 rounded-md bg-surface-muted px-3 py-1.5 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-60'
                                            type='button'
                                            onClick={() => toggleDetails(stack.stack_id)}
                                            disabled={detailLoading}
                                        >
                                            {detailLoading ? t('admin.stacks.detailsLoading') : detailsOpen ? t('common.close') : t('common.view')}
                                        </button>
                                        <button
                                            className='flex-1 rounded-md border border-danger/30 px-3 py-1.5 text-xs text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60'
                                            type='button'
                                            onClick={() => deleteStack(stack.stack_id)}
                                            disabled={deleteLoading}
                                        >
                                            {deleteLoading ? t('admin.stacks.deleting') : t('common.delete')}
                                        </button>
                                    </div>
                                    {detailError ? <p className='mt-2 text-xs text-danger'>{detailError}</p> : null}
                                    {detailLoading || detail ? (
                                        <div className='mt-3 rounded-lg bg-surface-muted/60 p-3'>
                                            {detailLoading ? (
                                                <p className='text-xs text-text-subtle'>{t('admin.stacks.detailsLoading')}</p>
                                            ) : detail ? (
                                                <div className='grid gap-3 sm:grid-cols-2'>
                                                    <div>
                                                        <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.statusLabel')}</p>
                                                        <p className='mt-1 text-sm text-text'>{detail.status}</p>
                                                    </div>
                                                    <div>
                                                        <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.runtimeLabel')}</p>
                                                        <p className='mt-1 break-all text-sm text-text'>{formatEndpoints(detail)}</p>
                                                    </div>
                                                    <div>
                                                        <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.targetPortLabel')}</p>
                                                        <p className='mt-1 text-sm text-text'>{formatTargetPorts(detail.ports)}</p>
                                                    </div>
                                                    <div>
                                                        <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.portLabel')}</p>
                                                        <p className='mt-1 text-sm text-text'>{formatNodePorts(detail.ports)}</p>
                                                    </div>
                                                </div>
                                            ) : null}
                                        </div>
                                    ) : null}
                                </div>
                            )
                        })}
                    </div>

                    <div className='hidden overflow-visible rounded-none bg-transparent md:block md:overflow-hidden md:rounded-xl md:bg-surface md:shadow-sm'>
                        <div className='overflow-x-auto'>
                            <div className='min-w-280'>
                                <div className='grid min-w-280 grid-cols-[150px_minmax(170px,1fr)_170px_160px_160px_160px_120px] bg-surface-muted px-6 py-3 text-[12px] text-text-muted'>
                                    <p className='font-medium whitespace-nowrap'>{t('common.id')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.challengeLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.userLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('admin.stacks.ttlLabel')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.createdAt')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.updatedAt')}</p>
                                    <p className='font-medium whitespace-nowrap'>{t('common.action')}</p>
                                </div>
                                {stacks.map((stack) => {
                                    const detail = detailById[stack.stack_id]
                                    const detailError = detailErrorById[stack.stack_id]
                                    const detailsOpen = !!detail
                                    const detailLoading = detailLoadingId === stack.stack_id
                                    const deleteLoading = deleteLoadingId === stack.stack_id

                                    return (
                                        <Fragment key={stack.stack_id}>
                                            <div className='grid min-w-280 grid-cols-[150px_minmax(170px,1fr)_170px_160px_160px_160px_120px] items-start px-6 py-4 transition hover:bg-surface-muted/40'>
                                                <p className='whitespace-nowrap font-mono text-xs text-text'>{stack.stack_id}</p>
                                                <div className='min-w-0 pr-3'>
                                                    <p className='truncate text-sm font-medium text-text'>{stack.challenge_title}</p>
                                                    <p className='truncate text-xs text-text-subtle'>
                                                        {stack.challenge_category} · #{stack.challenge_id}
                                                    </p>
                                                </div>
                                                <div className='min-w-0 pr-3'>
                                                    <p className='truncate text-sm font-medium text-text'>{stack.username}</p>
                                                    <p className='truncate text-xs text-text-subtle'>{stack.email}</p>
                                                </div>
                                                <p className='truncate text-xs text-text-subtle' title={formatOptionalDate(stack.ttl_expires_at)}>
                                                    {stack.ttl_expires_at ? formatCompactDateTime(stack.ttl_expires_at) : t('common.na')}
                                                </p>
                                                <p className='truncate text-xs text-text-subtle' title={formatDateTime(stack.created_at, localeTag)}>
                                                    {formatCompactDateTime(stack.created_at)}
                                                </p>
                                                <p className='truncate text-xs text-text-subtle' title={formatDateTime(stack.updated_at, localeTag)}>
                                                    {formatCompactDateTime(stack.updated_at)}
                                                </p>
                                                <div className='flex items-center gap-2 whitespace-nowrap'>
                                                    <button
                                                        className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-60'
                                                        type='button'
                                                        onClick={() => toggleDetails(stack.stack_id)}
                                                        disabled={detailLoading}
                                                    >
                                                        {detailLoading ? t('admin.stacks.detailsLoading') : detailsOpen ? t('common.close') : t('common.view')}
                                                    </button>
                                                    <button
                                                        className='rounded-md border border-danger/30 px-3 py-1 text-xs text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60'
                                                        type='button'
                                                        onClick={() => deleteStack(stack.stack_id)}
                                                        disabled={deleteLoading}
                                                    >
                                                        {deleteLoading ? t('admin.stacks.deleting') : t('common.delete')}
                                                    </button>
                                                </div>
                                            </div>
                                            {detailError ? <p className='bg-surface/40 px-6 py-4 text-xs text-danger'>{detailError}</p> : null}
                                            {detailLoading || detail ? (
                                                <div className='bg-surface/40 px-6 py-4'>
                                                    {detailLoading ? (
                                                        <p className='text-xs text-text-subtle'>{t('admin.stacks.detailsLoading')}</p>
                                                    ) : detail ? (
                                                        <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-4'>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.statusLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{detail.status}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.runtimeLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatEndpoints(detail)}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.targetPortLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatTargetPorts(detail.ports)}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.ttlLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatOptionalDate(detail.ttl_expires_at)}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('common.createdAt')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatDateTime(detail.created_at, localeTag)}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('common.updatedAt')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatDateTime(detail.updated_at, localeTag)}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.nodeLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{detail.node_public_ip ?? t('common.pending')}</p>
                                                            </div>
                                                            <div>
                                                                <p className='text-xs uppercase tracking-wide text-text-muted'>{t('admin.stacks.portLabel')}</p>
                                                                <p className='mt-1 text-sm text-text'>{formatNodePorts(detail.ports)}</p>
                                                            </div>
                                                        </div>
                                                    ) : null}
                                                </div>
                                            ) : null}
                                        </Fragment>
                                    )
                                })}
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </section>
    )
}

export default AdminStacks
