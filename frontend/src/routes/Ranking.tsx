import { useEffect, useMemo, useState } from 'react'
import ScoreboardTimeline from '../components/ScoreboardTimeline'
import UserAvatar from '../components/UserAvatar'
import { useApi } from '../lib/useApi'
import { useT } from '../lib/i18n'
import { formatApiError } from '../lib/utils'
import type { AffiliationRankingEntry, PaginationMeta, UserRankingEntry } from '../lib/types'
import { navigate } from '../lib/router'

interface RouteProps {
    routeParams?: Record<string, string>
}

const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
type RankingTabId = 'overall' | 'affiliations'
const TAB_PARAM = 'tab'
const RANKING_TAB_IDS: RankingTabId[] = ['overall', 'affiliations']

const getTabFromUrl = (): RankingTabId | null => {
    const params = new URLSearchParams(window.location.search)
    const value = params.get(TAB_PARAM)
    return RANKING_TAB_IDS.includes(value as RankingTabId) ? (value as RankingTabId) : null
}

const rankToneClass = (rank: number) => {
    if (rank === 1) return 'bg-warning/20 text-warning'
    if (rank === 2) return 'bg-info/20 text-info'
    if (rank === 3) return 'bg-accent/20 text-accent'
    return 'bg-surface-muted text-text-subtle'
}

const Ranking = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const [userRows, setUserRows] = useState<UserRankingEntry[]>([])
    const [affiliationRows, setAffiliationRows] = useState<AffiliationRankingEntry[]>([])
    const [affiliationUserRows, setAffiliationUserRows] = useState<UserRankingEntry[]>([])
    const [userPagination, setUserPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)
    const [affiliationPagination, setAffiliationPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)
    const [affiliationUserPagination, setAffiliationUserPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)
    const [userPage, setUserPage] = useState(1)
    const [affiliationPage, setAffiliationPage] = useState(1)
    const [affiliationUserPage, setAffiliationUserPage] = useState(1)
    const [selectedAffiliation, setSelectedAffiliation] = useState<AffiliationRankingEntry | null>(null)
    const [loadingUsers, setLoadingUsers] = useState(true)
    const [loadingAffiliations, setLoadingAffiliations] = useState(true)
    const [loadingAffiliationUsers, setLoadingAffiliationUsers] = useState(false)
    const [userErrorMessage, setUserErrorMessage] = useState('')
    const [affiliationErrorMessage, setAffiliationErrorMessage] = useState('')
    const [affiliationUserErrorMessage, setAffiliationUserErrorMessage] = useState('')
    const [activeTab, setActiveTab] = useState<RankingTabId>(() => getTabFromUrl() ?? 'overall')

    const rankingTabs = useMemo(
        () => [
            { id: 'overall', label: t('ranking.tab.overall') },
            { id: 'affiliations', label: t('ranking.tab.affiliations') },
        ],
        [t],
    )

    useEffect(() => {
        const handlePopState = () => {
            const nextTab = getTabFromUrl()
            if (nextTab && nextTab !== activeTab) {
                setActiveTab(nextTab)
            }
        }

        window.addEventListener('popstate', handlePopState)
        return () => window.removeEventListener('popstate', handlePopState)
    }, [activeTab])

    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        params.set(TAB_PARAM, activeTab)
        const nextQuery = params.toString()
        const nextUrl = nextQuery ? `${window.location.pathname}?${nextQuery}` : window.location.pathname
        window.history.replaceState(null, '', nextUrl)
    }, [activeTab])

    useEffect(() => {
        if (activeTab !== 'overall') return
        let active = true
        setLoadingUsers(true)
        setUserErrorMessage('')
        const load = async () => {
            try {
                const data = await api.rankingUsers(userPage, PAGE_SIZE)
                if (!active) return
                setUserRows(data.entries)
                setUserPagination(data.pagination)
                setUserErrorMessage('')
            } catch (error) {
                if (!active) return
                setUserErrorMessage(formatApiError(error, t).message)
                setUserRows([])
                setUserPagination(EMPTY_PAGINATION)
            } finally {
                if (active) setLoadingUsers(false)
            }
        }
        load()
        return () => {
            active = false
        }
    }, [activeTab, api, userPage, t])

    useEffect(() => {
        if (activeTab !== 'affiliations') return
        let active = true
        setLoadingAffiliations(true)
        setAffiliationErrorMessage('')
        const load = async () => {
            try {
                const data = await api.rankingAffiliations(affiliationPage, PAGE_SIZE)
                if (!active) return
                setAffiliationRows(data.entries)
                setAffiliationPagination(data.pagination)
                setAffiliationErrorMessage('')
            } catch (error) {
                if (!active) return
                setAffiliationErrorMessage(formatApiError(error, t).message)
                setAffiliationRows([])
                setAffiliationPagination(EMPTY_PAGINATION)
            } finally {
                if (active) setLoadingAffiliations(false)
            }
        }
        load()
        return () => {
            active = false
        }
    }, [activeTab, affiliationPage, api, t])

    useEffect(() => {
        if (activeTab !== 'affiliations') return
        if (!selectedAffiliation) {
            setAffiliationUserRows([])
            setAffiliationUserPagination(EMPTY_PAGINATION)
            setAffiliationUserErrorMessage('')
            return
        }
        let active = true
        setLoadingAffiliationUsers(true)
        setAffiliationUserErrorMessage('')
        const load = async () => {
            try {
                const data = await api.rankingAffiliationUsers(selectedAffiliation.affiliation_id, affiliationUserPage, PAGE_SIZE)
                if (!active) return
                setAffiliationUserRows(data.entries)
                setAffiliationUserPagination(data.pagination)
                setAffiliationUserErrorMessage('')
            } catch (error) {
                if (!active) return
                setAffiliationUserErrorMessage(formatApiError(error, t).message)
                setAffiliationUserRows([])
                setAffiliationUserPagination(EMPTY_PAGINATION)
            } finally {
                if (active) setLoadingAffiliationUsers(false)
            }
        }
        load()
        return () => {
            active = false
        }
    }, [activeTab, affiliationUserPage, api, selectedAffiliation, t])

    const selectedTitle = useMemo(() => selectedAffiliation?.name ?? '', [selectedAffiliation])
    const errorMessage = useMemo(() => {
        if (activeTab === 'overall') {
            return userErrorMessage
        }

        return affiliationUserErrorMessage || affiliationErrorMessage
    }, [activeTab, userErrorMessage, affiliationErrorMessage, affiliationUserErrorMessage])

    return (
        <section className='animate space-y-4'>
            <h2 className='text-2xl font-semibold text-text'>{t('ranking.title')}</h2>

            {errorMessage ? <p className='text-sm text-danger'>{errorMessage}</p> : null}

            <div className='lg:hidden'>
                <select className='w-full border border-border px-3 py-2 text-sm text-text focus:border-accent focus:outline-none' value={activeTab} onChange={(event) => setActiveTab(event.target.value as RankingTabId)}>
                    {rankingTabs.map((tab) => (
                        <option key={tab.id} value={tab.id}>
                            {tab.label}
                        </option>
                    ))}
                </select>
            </div>

            <div className='flex flex-col gap-4 lg:flex-row'>
                <nav className='hidden w-56 shrink-0 lg:block'>
                    {rankingTabs.map((tab) => (
                        <button
                            key={tab.id}
                            className={`flex w-full items-center rounded-none border-b border-border px-4 py-3 text-left text-sm ${activeTab === tab.id ? 'bg-surface-muted font-semibold text-accent' : 'text-text-muted hover:bg-surface-muted'}`}
                            onClick={() => setActiveTab(tab.id as RankingTabId)}
                            type='button'
                        >
                            {tab.label}
                        </button>
                    ))}
                </nav>

                <div className='min-w-0 flex-1 space-y-4'>
                    {activeTab === 'overall' ? (
                        <>
                            <ScoreboardTimeline />

                            <div className='rounded-xl border border-border bg-surface p-4'>
                                <div className='mb-3 flex items-center justify-between'>
                                    <h3 className='text-lg text-text'>{t('ranking.usersTitle')}</h3>
                                </div>
                                {loadingUsers ? (
                                    <p className='text-sm text-text-muted'>{t('common.loading')}</p>
                                ) : (
                                    <>
                                        <div className='space-y-2'>
                                            {userRows.map((row, idx) =>
                                                (() => {
                                                    const rank = (userPagination.page - 1) * userPagination.page_size + idx + 1
                                                    return (
                                                        <button
                                                            key={`ranking-user-${row.user_id}`}
                                                            className='flex w-full flex-wrap items-center gap-3 rounded-md px-3 py-2.5 text-left transition hover:bg-surface-muted cursor-pointer sm:flex-nowrap'
                                                            onClick={() => navigate(`/users/${row.user_id}`)}
                                                        >
                                                            <span className={`inline-flex h-7 min-w-9 items-center justify-center rounded-full px-2 text-xs font-semibold md:mr-4 ${rankToneClass(rank)}`}>#{rank}</span>
                                                            <UserAvatar username={row.username} size='sm' />
                                                            <div className='min-w-0 flex-1'>
                                                                <p className='truncate text-sm text-text'>{row.username}</p>
                                                                <p className='truncate text-xs text-text-subtle'>{row.affiliation_name?.trim() ? row.affiliation_name : ''}</p>
                                                                <p className='truncate text-xs text-text-subtle'>{row.bio ?? t('profile.noBio')}</p>
                                                            </div>
                                                            <div className='w-full text-left sm:w-auto sm:text-right'>
                                                                <p className='text-sm font-semibold text-text'>{t('common.pointsShort', { points: row.score })}</p>
                                                                <p className='text-xs text-text-subtle'>{t('ranking.solvedCount', { count: row.solved_count })}</p>
                                                            </div>
                                                        </button>
                                                    )
                                                })(),
                                            )}
                                        </div>
                                        {userRows.length === 0 ? <p className='py-3 text-sm text-text-muted'>{t('leaderboard.noScores')}</p> : null}
                                        <div className='mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-text-subtle'>
                                            <span>{t('common.totalCount', { count: userPagination.total_count })}</span>
                                            <div className='flex flex-wrap items-center gap-2'>
                                                <button
                                                    className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                                    disabled={!userPagination.has_prev}
                                                    onClick={() => setUserPage((prev) => Math.max(1, prev - 1))}
                                                >
                                                    {t('common.previous')}
                                                </button>
                                                <span>
                                                    {userPagination.page} / {userPagination.total_pages || 1}
                                                </span>
                                                <button className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50' disabled={!userPagination.has_next} onClick={() => setUserPage((prev) => prev + 1)}>
                                                    {t('common.next')}
                                                </button>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>
                        </>
                    ) : (
                        <>
                            <div className='rounded-xl border border-border bg-surface p-4'>
                                <div className='mb-3 flex items-center justify-between'>
                                    <h3 className='text-lg text-text'>{t('ranking.affiliationsTitle')}</h3>
                                </div>
                                {loadingAffiliations ? (
                                    <p className='text-sm text-text-muted'>{t('common.loading')}</p>
                                ) : (
                                    <>
                                        <div className='space-y-2'>
                                            {affiliationRows.map((row, idx) =>
                                                (() => {
                                                    const rank = (affiliationPagination.page - 1) * affiliationPagination.page_size + idx + 1
                                                    return (
                                                        <button
                                                            key={`ranking-affiliation-${row.affiliation_id}`}
                                                            className={`flex w-full flex-wrap items-center gap-3 rounded-md px-3 py-2.5 text-left transition hover:bg-surface-muted cursor-pointer sm:flex-nowrap ${
                                                                selectedAffiliation?.affiliation_id === row.affiliation_id ? 'bg-accent/10 ring-1 ring-accent/30' : ''
                                                            }`}
                                                            onClick={() => {
                                                                setSelectedAffiliation(row)
                                                                setAffiliationUserPage(1)
                                                            }}
                                                        >
                                                            <span className={`inline-flex h-7 min-w-9 items-center justify-center rounded-full px-2 text-xs font-semibold md:mr-4 ${rankToneClass(rank)}`}>#{rank}</span>
                                                            <div className='min-w-0 flex-1'>
                                                                <p className='truncate text-sm text-text'>{row.name}</p>
                                                                <p className='text-xs text-text-subtle'>{t('ranking.members', { count: row.user_count })}</p>
                                                            </div>
                                                            <div className='w-full text-left sm:w-auto sm:text-right'>
                                                                <p className='text-sm font-semibold text-text'>{t('common.pointsShort', { points: row.score })}</p>
                                                                <p className='text-xs text-text-subtle'>{t('ranking.solvedCount', { count: row.solved_count })}</p>
                                                            </div>
                                                        </button>
                                                    )
                                                })(),
                                            )}
                                        </div>
                                        {affiliationRows.length === 0 ? <p className='py-3 text-sm text-text-muted'>{t('leaderboard.noScores')}</p> : null}
                                        <div className='mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-text-subtle'>
                                            <span>{t('common.totalCount', { count: affiliationPagination.total_count })}</span>
                                            <div className='flex flex-wrap items-center gap-2'>
                                                <button
                                                    className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                                    disabled={!affiliationPagination.has_prev}
                                                    onClick={() => setAffiliationPage((prev) => Math.max(1, prev - 1))}
                                                >
                                                    {t('common.previous')}
                                                </button>
                                                <span>
                                                    {affiliationPagination.page} / {affiliationPagination.total_pages || 1}
                                                </span>
                                                <button
                                                    className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                                    disabled={!affiliationPagination.has_next}
                                                    onClick={() => setAffiliationPage((prev) => prev + 1)}
                                                >
                                                    {t('common.next')}
                                                </button>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>

                            {selectedAffiliation ? (
                                <div className='rounded-xl border border-border bg-surface p-4'>
                                    <div className='mb-3 flex items-center justify-between'>
                                        <h3 className='text-lg text-text'>{t('ranking.affiliationUsersTitle', { name: selectedTitle })}</h3>
                                    </div>
                                    {loadingAffiliationUsers ? (
                                        <p className='text-sm text-text-muted'>{t('common.loading')}</p>
                                    ) : (
                                        <>
                                            <div className='space-y-2'>
                                                {affiliationUserRows.map((row, idx) =>
                                                    (() => {
                                                        const rank = (affiliationUserPagination.page - 1) * affiliationUserPagination.page_size + idx + 1
                                                        return (
                                                            <button
                                                                key={`ranking-affiliation-user-${row.user_id}`}
                                                                className='flex w-full flex-wrap items-center gap-3 rounded-md px-3 py-2.5 text-left transition hover:bg-surface-muted cursor-pointer sm:flex-nowrap'
                                                                onClick={() => navigate(`/users/${row.user_id}`)}
                                                            >
                                                                <span className={`inline-flex h-7 min-w-9 items-center justify-center rounded-full px-2 text-xs font-semibold md:mr-4 ${rankToneClass(rank)}`}>#{rank}</span>
                                                                <UserAvatar username={row.username} size='sm' />
                                                                <div className='min-w-0 flex-1'>
                                                                    <p className='truncate text-sm text-text'>{row.username}</p>
                                                                    <p className='truncate text-xs text-text-subtle'>{selectedTitle}</p>
                                                                    <p className='truncate text-xs text-text-subtle'>{row.bio ?? t('profile.noBio')}</p>
                                                                </div>
                                                                <div className='w-full text-left sm:w-auto sm:text-right'>
                                                                    <p className='text-sm font-semibold text-text'>{t('common.pointsShort', { points: row.score })}</p>
                                                                    <p className='text-xs text-text-subtle'>{t('ranking.solvedCount', { count: row.solved_count })}</p>
                                                                </div>
                                                            </button>
                                                        )
                                                    })(),
                                                )}
                                            </div>
                                            {affiliationUserRows.length === 0 ? <p className='py-3 text-sm text-text-muted'>{t('leaderboard.noScores')}</p> : null}
                                            <div className='mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-text-subtle'>
                                                <span>{t('common.totalCount', { count: affiliationUserPagination.total_count })}</span>
                                                <div className='flex flex-wrap items-center gap-2'>
                                                    <button
                                                        className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                                        disabled={!affiliationUserPagination.has_prev}
                                                        onClick={() => setAffiliationUserPage((prev) => Math.max(1, prev - 1))}
                                                    >
                                                        {t('common.previous')}
                                                    </button>
                                                    <span>
                                                        {affiliationUserPagination.page} / {affiliationUserPagination.total_pages || 1}
                                                    </span>
                                                    <button
                                                        className='rounded-md border border-border/70 px-2.5 py-1 text-text disabled:opacity-50'
                                                        disabled={!affiliationUserPagination.has_next}
                                                        onClick={() => setAffiliationUserPage((prev) => prev + 1)}
                                                    >
                                                        {t('common.next')}
                                                    </button>
                                                </div>
                                            </div>
                                        </>
                                    )}
                                </div>
                            ) : null}
                        </>
                    )}
                </div>
            </div>
        </section>
    )
}

export default Ranking
