import { useEffect, useRef, useState } from 'react'
import { formatApiError } from '../lib/utils'
import type { LeaderboardChallenge, LeaderboardSolve, PaginationMeta, ScoreEntry } from '../lib/types'
import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'
import UserAvatar from './UserAvatar'

interface LegacyLeaderboardProps {
    refreshTrigger?: number
}

type UserEntryView = ScoreEntry & { solveMap: Map<number, LeaderboardSolve> }
const PAGE_SIZE = 20
const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: PAGE_SIZE, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const LEGACY_LEADERBOARD_SKELETON_ROWS = 5

const parsePositiveInt = (value: string | null, fallback: number) => {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const pushQueryState = (nextPage: number) => {
    if (typeof window === 'undefined') return
    const params = new URLSearchParams(window.location.search)
    if (nextPage > 1) {
        params.set('board_page', String(nextPage))
    } else {
        params.delete('board_page')
    }
    const query = params.toString()
    const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
    const currentURL = `${window.location.pathname}${window.location.search}`
    if (nextURL !== currentURL) {
        window.history.pushState({}, '', nextURL)
    }
}

const LegacyLeaderboard = ({ refreshTrigger = 0 }: LegacyLeaderboardProps) => {
    const t = useT()
    const api = useApi()
    const [challenges, setChallenges] = useState<LeaderboardChallenge[]>([])
    const [scores, setScores] = useState<UserEntryView[]>([])
    const [page, setPage] = useState(() => {
        if (typeof window === 'undefined') return 1
        const params = new URLSearchParams(window.location.search)
        return parsePositiveInt(params.get('board_page'), 1)
    })
    const [pagination, setPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')
    const requestIdRef = useRef(0)
    const flagSize = 22
    const fixedCols = '48px 80px minmax(160px, 1fr)'

    const buildSolveMap = (solves: LeaderboardSolve[]) => {
        const map = new Map<number, LeaderboardSolve>()
        for (const solve of solves) {
            map.set(solve.challenge_id, solve)
        }
        return map
    }

    useEffect(() => {
        let active = true
        requestIdRef.current += 1
        const currentRequest = requestIdRef.current
        setLoading(scores.length === 0)
        setErrorMessage('')

        const loadLegacyLeaderboard = async () => {
            try {
                const payload = await api.legacyLeaderboard(page, PAGE_SIZE)
                if (!active || currentRequest !== requestIdRef.current) return

                setChallenges(payload.challenges)
                setPagination(payload.pagination)
                setScores(
                    payload.entries.map((entry) => ({
                        ...entry,
                        solveMap: buildSolveMap(entry.solves ?? []),
                    })),
                )
            } catch (error) {
                if (active && currentRequest === requestIdRef.current) {
                    setErrorMessage(formatApiError(error, t).message)
                    setPagination(EMPTY_PAGINATION)
                }
            } finally {
                if (active && currentRequest === requestIdRef.current) {
                    setLoading(false)
                }
            }
        }

        loadLegacyLeaderboard()
        return () => {
            active = false
        }
    }, [api, page, refreshTrigger, t])

    useEffect(() => {
        const onPopState = () => {
            if (typeof window === 'undefined') return
            const params = new URLSearchParams(window.location.search)
            setPage(parsePositiveInt(params.get('board_page'), 1))
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    const gridTemplate = (count: number) => `${fixedCols} repeat(${count}, ${flagSize}px)`

    return (
        <div className='min-w-0 rounded-xl border border-border bg-surface p-4'>
            <div className='flex items-center justify-between'>
                <h3 className='text-lg text-text'>{t('legacyLeaderboard.title')}</h3>
                <span className='text-xs text-text-subtle'>{t('leaderboard.challengesCount', { count: challenges.length })}</span>
            </div>
            {loading ? (
                <div className='mt-4 space-y-3'>
                    <div className='h-4 w-36 rounded bg-surface-muted animate-pulse' />
                    <div className='space-y-2'>
                        {Array.from({ length: LEGACY_LEADERBOARD_SKELETON_ROWS }, (_, idx) => (
                            <div key={`legacy-leaderboard-skeleton-${idx}`} className='grid grid-cols-[48px_80px_minmax(160px,1fr)] items-center gap-3 rounded-md px-3 py-3'>
                                <div className='h-3 w-8 rounded bg-surface-muted animate-pulse' />
                                <div className='h-3 w-14 rounded bg-surface-muted animate-pulse' />
                                <div className='flex items-center gap-3'>
                                    <div className='h-8 w-8 rounded-full bg-surface-muted animate-pulse' />
                                    <div className='h-4 w-28 rounded bg-surface-muted animate-pulse' />
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            ) : errorMessage ? (
                <p className='mt-4 text-sm text-danger'>{errorMessage}</p>
            ) : (
                <div className='mt-4'>
                    <div className='overflow-x-auto pb-2'>
                        <div className='min-w-max'>
                            <div className='grid items-end gap-3 border-b border-border pb-3 text-[11px] uppercase tracking-wide text-text-subtle' style={{ gridTemplateColumns: gridTemplate(challenges.length) }}>
                                <span className='px-1'>#</span>
                                <span className='px-1'>{t('common.points')}</span>
                                <span className='px-1'>{t('leaderboard.userLabel')}</span>
                                {challenges.map((challenge) => (
                                    <span
                                        key={`challenge-${challenge.id}`}
                                        className='relative inline-block h-18 w-5.5 text-[10px]'
                                        title={t('leaderboard.challengeTitle', {
                                            title: challenge.title,
                                            points: challenge.points,
                                        })}
                                    >
                                        <span className='absolute bottom-0 left-0 block max-w-[15ch] overflow-hidden text-ellipsis whitespace-nowrap -rotate-35 origin-bottom-left leading-none'>{challenge.title}</span>
                                    </span>
                                ))}
                            </div>

                            <div className='divide-y divide-border/70'>
                                {scores.map((entry, index) => (
                                    <button
                                        key={`entry-${entry.username}-${index}`}
                                        className='grid w-full items-center gap-3 rounded-md px-3 py-3 text-left transition hover:bg-surface-muted cursor-pointer'
                                        style={{ gridTemplateColumns: gridTemplate(challenges.length) }}
                                        onClick={() => navigate(`/users/${entry.user_id}`)}
                                    >
                                        <span className='text-xs text-text-subtle'>#{(pagination.page - 1) * pagination.page_size + index + 1}</span>
                                        <span className='text-xs font-semibold text-text'>{t('common.pointsShort', { points: entry.score })}</span>
                                        <div className='flex items-center gap-3.75 truncate'>
                                            <UserAvatar username={entry.username} size='sm' />
                                            <span className='truncate text-sm text-text'>{entry.username}</span>
                                        </div>
                                        {challenges.map((challenge) => {
                                            const solve = entry.solveMap.get(challenge.id)
                                            return (
                                                <span
                                                    key={`solve-${entry.username}-${challenge.id}`}
                                                    className={`inline-flex h-4 w-5 items-center justify-center ${solve?.is_first_blood ? 'text-danger' : solve ? 'text-info' : 'text-text-subtle'}`}
                                                    title={`${challenge.title} • ${solve ? (solve.is_first_blood ? t('leaderboard.firstBlood') : t('leaderboard.solved')) : t('leaderboard.unsolved')}`}
                                                >
                                                    <svg viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg'>
                                                        <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity={solve ? '0.7' : '0'} />
                                                        <path
                                                            d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z'
                                                            fill='none'
                                                            stroke='currentColor'
                                                            strokeLinecap='round'
                                                            strokeLinejoin='round'
                                                        />
                                                    </svg>
                                                </span>
                                            )
                                        })}
                                    </button>
                                ))}
                            </div>
                        </div>
                    </div>

                    {scores.length === 0 ? <p className='text-sm text-text-muted'>{t('leaderboard.noScores')}</p> : null}
                    <div className='mt-2 flex items-center justify-between border-t border-border/70 pt-3 text-xs text-text-subtle'>
                        <span>{t('common.totalCount', { count: pagination.total_count })}</span>
                        <div className='flex items-center gap-2'>
                            <button
                                type='button'
                                className='rounded-md border border-border/70 px-2.5 py-1 text-text transition disabled:opacity-50'
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
                                className='rounded-md border border-border/70 px-2.5 py-1 text-text transition disabled:opacity-50'
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
                    </div>
                </div>
            )}
        </div>
    )
}

export default LegacyLeaderboard
