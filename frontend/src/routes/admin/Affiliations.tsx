import { useCallback, useEffect, useState } from 'react'
import { useApi } from '../../lib/useApi'
import { useT } from '../../lib/i18n'
import { formatApiError } from '../../lib/utils'
import type { Affiliation, PaginationMeta } from '../../lib/types'
import FormMessage from '../../components/FormMessage'

const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const AFFILIATION_SKELETON_ROWS = 5

const AdminAffiliations = () => {
    const t = useT()
    const api = useApi()
    const [rows, setRows] = useState<Affiliation[]>([])
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)
    const [page, setPage] = useState(1)
    const [name, setName] = useState('')
    const [loading, setLoading] = useState(false)
    const [creating, setCreating] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')

    const loadRows = useCallback(
        async (targetPage: number) => {
            setLoading(true)
            setErrorMessage('')
            try {
                const data = await api.affiliations(targetPage, 20)
                setRows(data.affiliations)
                setPagination(data.pagination)
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
                setRows([])
                setPagination(EMPTY_PAGINATION)
            } finally {
                setLoading(false)
            }
        },
        [api, t],
    )

    useEffect(() => {
        void loadRows(page)
    }, [loadRows, page])

    const create = async () => {
        const trimmed = name.trim()
        if (!trimmed) {
            setErrorMessage(t('errors.required'))
            return
        }

        setCreating(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const created = await api.createAffiliation(trimmed)
            setSuccessMessage(t('admin.affiliations.created', { name: created.name }))
            setName('')
            if (page === 1) {
                await loadRows(1)
            } else {
                setPage(1)
            }
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setCreating(false)
        }
    }

    return (
        <section className='space-y-4'>
            <div className='space-y-2 rounded-lg border border-border bg-surface p-4'>
                <h3 className='text-base text-text'>{t('admin.affiliations.title')}</h3>
                <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
                    <input
                        type='text'
                        value={name}
                        onChange={(event) => setName(event.target.value)}
                        className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                        placeholder={t('admin.affiliations.namePlaceholder')}
                        disabled={creating}
                    />
                    <button type='button' className='rounded-md border border-border bg-surface-muted px-4 py-2 text-sm text-text transition hover:bg-surface-subtle disabled:opacity-50' disabled={creating} onClick={create}>
                        {creating ? t('admin.affiliations.creating') : t('admin.affiliations.create')}
                    </button>
                </div>
            </div>

            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            {loading ? (
                <div className='rounded-lg border border-border bg-surface'>
                    <div className='grid grid-cols-[80px_minmax(0,1fr)] gap-3 border-b border-border bg-surface-muted px-4 py-2 text-xs text-text-muted'>
                        <span>{t('common.id')}</span>
                        <span>{t('common.affiliation')}</span>
                    </div>
                    <div className='divide-y divide-border/70'>
                        {Array.from({ length: AFFILIATION_SKELETON_ROWS }, (_, idx) => (
                            <div key={`admin-affiliation-skeleton-${idx}`} className='grid grid-cols-[80px_minmax(0,1fr)] gap-3 px-4 py-3'>
                                <div className='h-3 w-8 rounded bg-surface-muted animate-pulse' />
                                <div className='h-4 w-2/3 rounded bg-surface-muted animate-pulse' />
                            </div>
                        ))}
                    </div>
                </div>
            ) : (
                <div className='rounded-lg border border-border bg-surface'>
                    <div className='grid grid-cols-[80px_minmax(0,1fr)] gap-3 border-b border-border bg-surface-muted px-4 py-2 text-xs text-text-muted'>
                        <span>{t('common.id')}</span>
                        <span>{t('common.affiliation')}</span>
                    </div>
                    <div className='divide-y divide-border/70'>
                        {rows.map((row) => (
                            <div key={row.id} className='grid grid-cols-[80px_minmax(0,1fr)] gap-3 px-4 py-3 text-sm text-text'>
                                <span>{row.id}</span>
                                <span className='truncate'>{row.name}</span>
                            </div>
                        ))}
                    </div>
                    {rows.length === 0 ? <p className='px-4 py-6 text-center text-sm text-text-muted'>{t('admin.affiliations.empty')}</p> : null}
                    <div className='flex items-center justify-end gap-2 border-t border-border px-4 py-2 text-xs text-text-subtle'>
                        <button className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50' disabled={!pagination.has_prev} onClick={() => setPage((prev) => Math.max(1, prev - 1))}>
                            {t('common.previous')}
                        </button>
                        <span>
                            {pagination.page} / {pagination.total_pages || 1}
                        </span>
                        <button className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50' disabled={!pagination.has_next} onClick={() => setPage((prev) => prev + 1)}>
                            {t('common.next')}
                        </button>
                    </div>
                </div>
            )}
        </section>
    )
}

export default AdminAffiliations
