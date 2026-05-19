import { useEffect, useState } from 'react'
import { useApi } from '../lib/useApi'
import { useT } from '../lib/i18n'
import { formatApiError } from '../lib/utils'
import type { ChallengeSeries as ChallengeSeriesItem, PaginationMeta } from '../lib/types'
import { navigate } from '../lib/router'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }

const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const ChallengeSeries = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const api = useApi()
    const t = useT()
    const [series, setSeries] = useState<ChallengeSeriesItem[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const readQueryState = () => {
        if (typeof window === 'undefined') return { page: 1 }
        const params = new URLSearchParams(window.location.search)
        return { page: parsePositiveInt(params.get('page'), 1) }
    }

    const [page, setPage] = useState(readQueryState().page)

    const pushQueryState = (nextPage: number) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams()
        if (nextPage > 1) params.set('page', String(nextPage))
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) window.history.pushState({}, '', nextURL)
    }

    useEffect(() => {
        const onPopState = () => setPage(readQueryState().page)
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    useEffect(() => {
        const load = async () => {
            setLoading(true)
            setErrorMessage('')
            try {
                const data = await api.challengeSeries(page, PAGE_SIZE)
                setSeries(data.series)
                setPagination(data.pagination)
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
                setPagination(EMPTY_PAGINATION)
            } finally {
                setLoading(false)
            }
        }

        void load()
    }, [page])

    return (
        <section className='animate space-y-4'>
            <div className='flex flex-wrap items-end justify-between gap-3'>
                <div>
                    <h2 className='text-3xl text-text'>{t('challengeSeries.title')}</h2>
                    <p className='mt-1 text-sm text-text-muted'>{t('challengeSeries.subtitle')}</p>
                </div>
                <span className='text-sm text-text-muted'>{t('common.totalCount', { count: pagination.total_count })}</span>
            </div>

            <div className='rounded-lg border border-border/70 bg-surface'>
                {loading ? (
                    <div className='space-y-2 p-4'>
                        {Array.from({ length: 6 }, (_, idx) => (
                            <div key={`series-skeleton-${idx}`} className='rounded-xl border border-border/60 bg-surface p-4'>
                                <div className='animate-pulse space-y-2'>
                                    <div className='h-4 w-2/5 rounded bg-surface-muted' />
                                    <div className='h-3 w-4/5 rounded bg-surface-muted' />
                                </div>
                            </div>
                        ))}
                    </div>
                ) : errorMessage ? (
                    <div className='px-4 py-8 text-sm text-danger'>{errorMessage}</div>
                ) : series.length === 0 ? (
                    <div className='px-4 py-8 text-sm text-text-muted'>{t('challengeSeries.empty')}</div>
                ) : (
                    <div className='divide-y divide-border/60'>
                        {series.map((item) => (
                            <button key={item.id} type='button' className='bg-background rounded-none block w-full px-4 py-4 text-left transition hover:bg-surface' onClick={() => navigate(`/series/${item.id}`)}>
                                <p className='text-base font-semibold text-text'>{item.title}</p>
                                <p className='mt-1 line-clamp-2 text-sm text-text-muted'>{item.description}</p>
                            </button>
                        ))}
                    </div>
                )}
            </div>

            <div className='flex items-center justify-end gap-2 text-sm text-text-muted'>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50'
                    disabled={!pagination.has_prev}
                    onClick={() => {
                        const nextPage = Math.max(1, page - 1)
                        setPage(nextPage)
                        pushQueryState(nextPage)
                    }}
                >
                    {t('common.previous')}
                </button>
                <span>
                    {pagination.page} / {pagination.total_pages || 1}
                </span>
                <button
                    type='button'
                    className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text transition hover:bg-surface-subtle disabled:opacity-50'
                    disabled={!pagination.has_next}
                    onClick={() => {
                        const nextPage = page + 1
                        setPage(nextPage)
                        pushQueryState(nextPage)
                    }}
                >
                    {t('common.next')}
                </button>
            </div>
        </section>
    )
}

export default ChallengeSeries
