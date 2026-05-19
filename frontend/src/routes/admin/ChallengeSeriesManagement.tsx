import { useEffect, useMemo, useState } from 'react'
import { useApi } from '../../lib/useApi'
import { useT } from '../../lib/i18n'
import { formatApiError } from '../../lib/utils'
import type { Challenge, ChallengeSeries, PaginationMeta } from '../../lib/types'
import FormMessage from '../../components/FormMessage'

const ChallengeSeriesManagement = () => {
    const api = useApi()
    const t = useT()

    const [series, setSeries] = useState<ChallengeSeries[]>([])
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [selectedSeriesID, setSelectedSeriesID] = useState<number | null>(null)
    const [selectedChallengeIDs, setSelectedChallengeIDs] = useState<number[]>([])
    const [selectedChallengeTitles, setSelectedChallengeTitles] = useState<Record<number, string>>({})

    const [title, setTitle] = useState('')
    const [description, setDescription] = useState('')
    const [loadingSeries, setLoadingSeries] = useState(false)
    const [loadingChallenges, setLoadingChallenges] = useState(false)
    const [saving, setSaving] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [successMessage, setSuccessMessage] = useState('')
    const [challengeSearch, setChallengeSearch] = useState('')
    const [availablePage, setAvailablePage] = useState(1)
    const [availablePagination, setAvailablePagination] = useState<PaginationMeta>({ page: 1, page_size: 10, total_count: 0, total_pages: 0, has_prev: false, has_next: false })
    const [draggingIndex, setDraggingIndex] = useState<number | null>(null)
    const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)
    const AVAILABLE_PAGE_SIZE = 10

    const selectedSeries = useMemo(() => series.find((row) => row.id === selectedSeriesID) ?? null, [series, selectedSeriesID])

    const loadSeries = async () => {
        const first = await api.challengeSeries(1, 10)
        setSeries(first.series)
    }

    const loadChallenges = async (page: number, q: string) => {
        const res = await api.searchChallenges(q.trim(), page, AVAILABLE_PAGE_SIZE)
        setChallenges(res.challenges)
        setAvailablePagination(res.pagination)
    }

    const loadSeriesDetail = async (id: number) => {
        const detail = await api.challengeSeriesDetail(id)
        setSelectedChallengeIDs(detail.challenges.map((item) => item.id))
        const titles: Record<number, string> = {}
        detail.challenges.forEach((item) => {
            titles[item.id] = item.title
        })
        setSelectedChallengeTitles(titles)
    }

    useEffect(() => {
        const load = async () => {
            setLoadingSeries(true)
            setErrorMessage('')
            try {
                await loadSeries()
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                setLoadingSeries(false)
            }
        }
        void load()
    }, [])

    useEffect(() => {
        const load = async () => {
            setLoadingChallenges(true)
            try {
                await loadChallenges(availablePage, challengeSearch)
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                setLoadingChallenges(false)
            }
        }
        void load()
    }, [availablePage, challengeSearch])

    useEffect(() => {
        if (!selectedSeries) {
            setTitle('')
            setDescription('')
            setSelectedChallengeIDs([])
            setSelectedChallengeTitles({})
            return
        }
        setTitle(selectedSeries.title)
        setDescription(selectedSeries.description)
        void loadSeriesDetail(selectedSeries.id)
    }, [selectedSeriesID])

    const challengeMap = useMemo(() => new Map(challenges.map((item) => [item.id, item])), [challenges])

    const availableChallengeOptions = useMemo(() => {
        const selected = new Set(selectedChallengeIDs)
        return challenges.filter((item) => {
            if (selected.has(item.id)) return false
            return true
        })
    }, [challenges, selectedChallengeIDs])

    useEffect(() => {
        setAvailablePage(1)
    }, [challengeSearch])

    const moveItem = (index: number, direction: -1 | 1) => {
        const next = index + direction
        if (next < 0 || next >= selectedChallengeIDs.length) return
        const copied = [...selectedChallengeIDs]
        ;[copied[index], copied[next]] = [copied[next], copied[index]]
        setSelectedChallengeIDs(copied)
    }

    const reorderByDrag = (from: number, to: number) => {
        if (from === to || from < 0 || to < 0 || from >= selectedChallengeIDs.length || to >= selectedChallengeIDs.length) return
        const copied = [...selectedChallengeIDs]
        const [moved] = copied.splice(from, 1)
        copied.splice(to, 0, moved)
        setSelectedChallengeIDs(copied)
    }

    const createSeries = async () => {
        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            const created = await api.createChallengeSeries({ title: title.trim(), description: description.trim() })
            if (selectedChallengeIDs.length > 0) {
                await api.replaceChallengeSeriesChallenges(created.id, selectedChallengeIDs)
            }
            await loadSeries()
            setSelectedSeriesID(created.id)
            setSuccessMessage(t('admin.series.created'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
    }

    const updateSeries = async () => {
        if (!selectedSeries) return
        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            await api.updateChallengeSeries(selectedSeries.id, { title: title.trim(), description: description.trim() })
            await api.replaceChallengeSeriesChallenges(selectedSeries.id, selectedChallengeIDs)
            await loadSeries()
            setSuccessMessage(t('admin.series.saved'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
    }

    const removeSeries = async () => {
        if (!selectedSeries) return
        setSaving(true)
        setErrorMessage('')
        setSuccessMessage('')
        try {
            await api.deleteChallengeSeries(selectedSeries.id)
            await loadSeries()
            setSelectedSeriesID(null)
            setSuccessMessage(t('admin.series.deleted'))
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
    }

    return (
        <section className='space-y-4'>
            {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
            {successMessage ? <FormMessage variant='success' message={successMessage} /> : null}

            <div className='grid gap-4 lg:grid-cols-[320px_1fr]'>
                <div className='rounded-lg border border-border bg-surface'>
                    <div className='border-b border-border bg-surface-muted px-4 py-2 text-xs text-text-muted'>{t('admin.series.listTitle')}</div>
                    <div className='max-h-110 overflow-y-auto'>
                        {loadingSeries ? (
                            <div className='p-4 text-sm text-text-muted'>{t('common.loading')}</div>
                        ) : series.length === 0 ? (
                            <div className='p-4 text-sm text-text-muted'>{t('admin.series.empty')}</div>
                        ) : (
                            series.map((row) => (
                                <button
                                    key={row.id}
                                    type='button'
                                    className={`rounded-none block w-full border-b border-border/60 px-4 py-3 text-left transition hover:bg-surface-muted ${selectedSeriesID === row.id ? 'bg-surface-muted' : ''}`}
                                    onClick={() => setSelectedSeriesID(row.id)}
                                >
                                    <p className='truncate text-sm font-medium text-text'>{row.title}</p>
                                    <p className='mt-1 line-clamp-1 text-xs text-text-muted'>{row.description}</p>
                                </button>
                            ))
                        )}
                    </div>
                </div>

                <div className='space-y-4 rounded-lg border border-border bg-surface p-3 md:p-4'>
                    <div className='grid gap-4 md:grid-cols-2'>
                        <div className='md:col-span-2'>
                            <label className='text-xs uppercase tracking-wide text-text-muted'>{t('common.title')}</label>
                            <input className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-2.5 text-sm text-text focus:border-accent focus:outline-none' value={title} onChange={(event) => setTitle(event.target.value)} />
                        </div>
                        <div className='md:col-span-2'>
                            <label className='text-xs uppercase tracking-wide text-text-muted'>{t('common.description')}</label>
                            <textarea
                                className='mt-2 min-h-24 w-full rounded-xl border border-border bg-surface px-4 py-2.5 text-sm text-text focus:border-accent focus:outline-none'
                                value={description}
                                onChange={(event) => setDescription(event.target.value)}
                            />
                        </div>
                    </div>

                    <div className='rounded-lg border border-border/70 bg-surface-muted p-2.5 md:p-3'>
                        <p className='text-sm font-semibold text-text'>{t('admin.series.orderTitle')}</p>
                        <div className='mt-3 space-y-2'>
                            {selectedChallengeIDs.map((id, index) => {
                                const item = challengeMap.get(id)
                                return (
                                    <div
                                        key={`${id}-${index}`}
                                        className={`border bg-surface px-2.5 py-2 ${dragOverIndex === index ? 'border-accent/60' : 'border-border'}`}
                                        draggable
                                        onDragStart={(event) => {
                                            setDraggingIndex(index)
                                            event.dataTransfer.effectAllowed = 'move'
                                        }}
                                        onDragOver={(event) => {
                                            event.preventDefault()
                                            if (dragOverIndex !== index) setDragOverIndex(index)
                                        }}
                                        onDrop={(event) => {
                                            event.preventDefault()
                                            if (draggingIndex !== null) reorderByDrag(draggingIndex, index)
                                            setDraggingIndex(null)
                                            setDragOverIndex(null)
                                        }}
                                        onDragEnd={() => {
                                            setDraggingIndex(null)
                                            setDragOverIndex(null)
                                        }}
                                    >
                                        <div className='flex items-start justify-between gap-2'>
                                            <div className='min-w-0 flex items-start gap-2'>
                                                <span className='w-5 shrink-0 pt-0.5 text-xs text-text-subtle'>{index + 1}</span>
                                                <span className='line-clamp-2 text-sm text-text'>{item?.title ?? selectedChallengeTitles[id] ?? `#${id}`}</span>
                                            </div>
                                            <span className='cursor-grab shrink-0 pt-0.5 text-xs text-text-subtle'>⋮⋮</span>
                                        </div>
                                        <div className='mt-2 flex flex-wrap justify-end gap-1.5'>
                                            <button type='button' className='rounded border border-border px-2 py-0.5 text-xs text-text' onClick={() => moveItem(index, -1)}>
                                                ↑
                                            </button>
                                            <button type='button' className='rounded border border-border px-2 py-0.5 text-xs text-text' onClick={() => moveItem(index, 1)}>
                                                ↓
                                            </button>
                                            <button type='button' className='rounded border border-danger/40 px-2 py-0.5 text-xs text-danger' onClick={() => setSelectedChallengeIDs((prev) => prev.filter((_, i) => i !== index))}>
                                                {t('common.remove')}
                                            </button>
                                        </div>
                                    </div>
                                )
                            })}
                            {selectedChallengeIDs.length === 0 ? <p className='text-xs text-text-muted'>{t('admin.series.orderEmpty')}</p> : null}
                        </div>

                        <input
                            type='text'
                            value={challengeSearch}
                            onChange={(event) => setChallengeSearch(event.target.value)}
                            placeholder={t('common.search')}
                            className='mt-3 w-full border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                        />
                        <div className='mt-3 overflow-hidden border border-border/60 bg-surface'>
                            <div className='hidden grid-cols-[70px_minmax(0,1fr)_70px] bg-surface-muted px-3 py-2 text-[11px] text-text-muted md:grid'>
                                <span>{t('common.id')}</span>
                                <span>{t('challenges.tableProblem')}</span>
                                <span>{t('common.add')}</span>
                            </div>
                            <div className='divide-y divide-border/60'>
                                {loadingChallenges ? (
                                    <p className='px-3 py-4 text-xs text-text-muted'>{t('common.loading')}</p>
                                ) : availableChallengeOptions.length === 0 ? (
                                    <p className='px-3 py-4 text-xs text-text-muted'>{t('users.noResults')}</p>
                                ) : (
                                    availableChallengeOptions.map((item) => (
                                        <div key={item.id} className='px-3 py-2'>
                                            <div className='md:hidden'>
                                                <div className='flex items-center justify-between gap-2'>
                                                    <div className='min-w-0'>
                                                        <p className='text-xs text-text-subtle'>#{item.id}</p>
                                                        <p className='line-clamp-2 text-sm text-text'>{item.title}</p>
                                                    </div>
                                                    <button
                                                        type='button'
                                                        className='shrink-0 border border-border px-2 py-1 text-xs text-text hover:bg-surface-subtle'
                                                        onClick={() => {
                                                            setSelectedChallengeIDs((prev) => [...prev, item.id])
                                                            setSelectedChallengeTitles((prev) => ({ ...prev, [item.id]: item.title }))
                                                        }}
                                                    >
                                                        +
                                                    </button>
                                                </div>
                                            </div>
                                            <div className='hidden md:grid md:grid-cols-[70px_minmax(0,1fr)_70px] md:items-center md:gap-2'>
                                                <span className='text-xs text-text-subtle'>#{item.id}</span>
                                                <span className='truncate text-sm text-text'>{item.title}</span>
                                                <button
                                                    type='button'
                                                    className='border border-border px-2 py-1 text-xs text-text hover:bg-surface-subtle'
                                                    onClick={() => {
                                                        setSelectedChallengeIDs((prev) => [...prev, item.id])
                                                        setSelectedChallengeTitles((prev) => ({ ...prev, [item.id]: item.title }))
                                                    }}
                                                >
                                                    +
                                                </button>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                            <div className='flex items-center justify-end gap-2 border-t border-border px-3 py-2 text-xs text-text-subtle'>
                                <span className='mr-auto'>{t('common.totalCount', { count: availablePagination.total_count })}</span>
                                <button
                                    type='button'
                                    className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                    disabled={!availablePagination.has_prev}
                                    onClick={() => setAvailablePage((prev) => Math.max(1, prev - 1))}
                                >
                                    {t('common.previous')}
                                </button>
                                <span>
                                    {availablePagination.page} / {availablePagination.total_pages || 1}
                                </span>
                                <button type='button' className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50' disabled={!availablePagination.has_next} onClick={() => setAvailablePage((prev) => prev + 1)}>
                                    {t('common.next')}
                                </button>
                            </div>
                        </div>
                    </div>

                    <div className='grid grid-cols-1 gap-2 sm:flex sm:flex-wrap'>
                        <button
                            type='button'
                            className='rounded-md bg-accent px-4 py-2 text-sm text-contrast-foreground disabled:opacity-60'
                            disabled={saving || !title.trim() || !description.trim()}
                            onClick={selectedSeries ? updateSeries : createSeries}
                        >
                            {saving ? t('common.loading') : selectedSeries ? t('common.save') : t('common.add')}
                        </button>
                        {selectedSeries ? (
                            <button type='button' className='rounded-md border border-danger/40 px-4 py-2 text-sm text-danger disabled:opacity-60' disabled={saving} onClick={removeSeries}>
                                {t('common.delete')}
                            </button>
                        ) : null}
                        {selectedSeries ? (
                            <button
                                type='button'
                                className='rounded-md border border-border px-4 py-2 text-sm text-text disabled:opacity-60'
                                disabled={saving}
                                onClick={() => {
                                    setSelectedSeriesID(null)
                                    setTitle('')
                                    setDescription('')
                                    setSelectedChallengeIDs([])
                                }}
                            >
                                {t('common.cancel')}
                            </button>
                        ) : null}
                    </div>
                </div>
            </div>
        </section>
    )
}

export default ChallengeSeriesManagement
