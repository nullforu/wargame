import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import type { AdminStackListItem, Stack } from '../../lib/types'
import { useApi } from '../../lib/useApi'
import { formatApiError, formatDateTime } from '../../lib/utils'
import { getLocaleTag, useLocale, useT } from '../../lib/i18n'
import FormMessage from '../../components/FormMessage'

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
                <p className='text-sm text-text-muted'>{t('admin.stacks.loading')}</p>
            ) : stacks.length === 0 ? (
                <p className='text-sm text-text-muted'>{t('admin.stacks.noStacks')}</p>
            ) : (
                <div className='overflow-hidden rounded-2xl border border-border bg-surface'>
                    <div className='overflow-x-auto'>
                        <table className='w-full text-left text-sm text-text'>
                            <thead className='border-b border-border bg-surface-muted text-xs uppercase tracking-wide text-text-muted'>
                                <tr>
                                    <th className='px-6 py-3'>{t('common.id')}</th>
                                    <th className='px-6 py-3'>{t('admin.stacks.challengeLabel')}</th>
                                    <th className='px-6 py-3'>{t('admin.stacks.userLabel')}</th>
                                    <th className='px-6 py-3'>{t('admin.stacks.ttlLabel')}</th>
                                    <th className='px-6 py-3'>{t('common.createdAt')}</th>
                                    <th className='px-6 py-3'>{t('common.updatedAt')}</th>
                                    <th className='px-6 py-3'>{t('common.action')}</th>
                                </tr>
                            </thead>
                            <tbody className='divide-y divide-border'>
                                {stacks.map((stack) => {
                                    const detail = detailById[stack.stack_id]
                                    const detailError = detailErrorById[stack.stack_id]
                                    const detailsOpen = !!detail
                                    const detailLoading = detailLoadingId === stack.stack_id
                                    const deleteLoading = deleteLoadingId === stack.stack_id

                                    return (
                                        <Fragment key={stack.stack_id}>
                                            <tr className='align-top'>
                                                <td className='whitespace-nowrap px-6 py-4 font-mono text-xs text-text'>{stack.stack_id}</td>
                                                <td className='px-6 py-4'>
                                                    <div className='font-medium'>{stack.challenge_title}</div>
                                                    <div className='text-xs text-text-subtle'>
                                                        {stack.challenge_category} · #{stack.challenge_id}
                                                    </div>
                                                </td>
                                                <td className='px-6 py-4'>
                                                    <div className='font-medium'>{stack.username}</div>
                                                    <div className='text-xs text-text-subtle'>{stack.email}</div>
                                                </td>
                                                <td className='px-6 py-4 text-xs text-text-subtle'>{formatOptionalDate(stack.ttl_expires_at)}</td>
                                                <td className='px-6 py-4 text-xs text-text-subtle'>{formatDateTime(stack.created_at, localeTag)}</td>
                                                <td className='px-6 py-4 text-xs text-text-subtle'>{formatDateTime(stack.updated_at, localeTag)}</td>
                                                <td className='px-6 py-4'>
                                                    <div className='flex flex-wrap items-center gap-2'>
                                                        <button
                                                            className='rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-text transition hover:border-accent hover:text-accent disabled:opacity-60 cursor-pointer'
                                                            type='button'
                                                            onClick={() => toggleDetails(stack.stack_id)}
                                                            disabled={detailLoading}
                                                        >
                                                            {detailLoading ? t('admin.stacks.detailsLoading') : detailsOpen ? t('common.close') : t('common.view')}
                                                        </button>
                                                        <button
                                                            className='rounded-lg border border-danger/30 px-3 py-1.5 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60 cursor-pointer'
                                                            type='button'
                                                            onClick={() => deleteStack(stack.stack_id)}
                                                            disabled={deleteLoading}
                                                        >
                                                            {deleteLoading ? t('admin.stacks.deleting') : t('common.delete')}
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                            {detailError ? (
                                                <tr className='bg-surface/40'>
                                                    <td className='px-6 py-4 text-xs text-danger' colSpan={7}>
                                                        {detailError}
                                                    </td>
                                                </tr>
                                            ) : null}
                                            {detailLoading || detail ? (
                                                <tr className='bg-surface/40'>
                                                    <td className='px-6 py-4' colSpan={7}>
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
                                                    </td>
                                                </tr>
                                            ) : null}
                                        </Fragment>
                                    )
                                })}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </section>
    )
}

export default AdminStacks
