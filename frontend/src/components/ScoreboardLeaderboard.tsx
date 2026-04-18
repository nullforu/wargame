import { useEffect, useRef, useState } from 'react'
import { formatApiError } from '../lib/utils'
import type { LeaderboardChallenge, LeaderboardSolve, ScoreEntry } from '../lib/types'
import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'

interface ScoreboardLeaderboardProps {
    refreshTrigger?: number
}

type UserEntryView = ScoreEntry & { solveMap: Map<number, LeaderboardSolve> }

const ScoreboardLeaderboard = ({ refreshTrigger = 0 }: ScoreboardLeaderboardProps) => {
    const t = useT()
    const api = useApi()
    const [challenges, setChallenges] = useState<LeaderboardChallenge[]>([])
    const [scores, setScores] = useState<UserEntryView[]>([])
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

        const loadScoreboard = async () => {
            try {
                const payload = await api.leaderboard()
                if (!active || currentRequest !== requestIdRef.current) return

                setChallenges(payload.challenges)
                setScores(
                    payload.entries.map((entry) => ({
                        ...entry,
                        solveMap: buildSolveMap(entry.solves ?? []),
                    })),
                )
            } catch (error) {
                if (active && currentRequest === requestIdRef.current) {
                    setErrorMessage(formatApiError(error, t).message)
                }
            } finally {
                if (active && currentRequest === requestIdRef.current) {
                    setLoading(false)
                }
            }
        }

        loadScoreboard()
        return () => {
            active = false
        }
    }, [api, refreshTrigger, t])

    const gridTemplate = (count: number) => `${fixedCols} repeat(${count}, ${flagSize}px)`

    return (
        <div className='min-w-0 rounded-2xl border border-border bg-surface p-4 sm:p-6'>
            <div className='flex items-center justify-between'>
                <h3 className='text-lg text-text'>{t('leaderboard.title')}</h3>
                <span className='text-xs text-text-subtle'>{t('leaderboard.challengesCount', { count: challenges.length })}</span>
            </div>
            {loading ? (
                <p className='mt-4 text-sm text-text-muted'>{t('common.loading')}</p>
            ) : errorMessage ? (
                <p className='mt-4 text-sm text-danger'>{errorMessage}</p>
            ) : (
                <div className='mt-4 overflow-x-auto'>
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
                                    className='grid w-full items-center gap-3 px-3 py-3 text-left transition hover:bg-surface-muted cursor-pointer'
                                    style={{ gridTemplateColumns: gridTemplate(challenges.length) }}
                                    onClick={() => navigate(`/users/${entry.user_id}`)}
                                >
                                    <span className='text-xs text-text-subtle'>#{index + 1}</span>
                                    <span className='text-xs font-semibold text-text'>{t('common.pointsShort', { points: entry.score })}</span>
                                    <span className='truncate text-sm text-text'>{entry.username}</span>
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

                    {scores.length === 0 ? <p className='text-sm text-text-muted'>{t('leaderboard.noScores')}</p> : null}
                </div>
            )}
        </div>
    )
}

export default ScoreboardLeaderboard
