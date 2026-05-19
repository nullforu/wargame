import { useEffect, useState } from 'react'
import { useApi } from '../lib/useApi'
import { getCategoryKey, useT } from '../lib/i18n'
import { formatApiError } from '../lib/utils'
import type { Challenge, ChallengeSeries as ChallengeSeriesItem } from '../lib/types'
import { navigate } from '../lib/router'
import { LevelBadge } from './Challenges'
import FlagIcon from '../components/FlagIcon'

interface RouteProps {
    routeParams?: Record<string, string>
}

const ChallengeSeriesDetail = ({ routeParams = {} }: RouteProps) => {
    const api = useApi()
    const t = useT()
    const seriesID = Number(routeParams.id)

    const [series, setSeries] = useState<ChallengeSeriesItem | null>(null)
    const [challenges, setChallenges] = useState<Challenge[]>([])
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    useEffect(() => {
        const load = async () => {
            if (!Number.isInteger(seriesID) || seriesID <= 0) {
                setErrorMessage(t('errors.requestFailed'))
                setLoading(false)
                return
            }

            setLoading(true)
            setErrorMessage('')
            try {
                const data = await api.challengeSeriesDetail(seriesID)
                setSeries(data.series)
                setChallenges(data.challenges)
            } catch (error) {
                setErrorMessage(formatApiError(error, t).message)
            } finally {
                setLoading(false)
            }
        }

        void load()
    }, [seriesID])

    return (
        <section className='animate space-y-4'>
            <div className='flex flex-col items-start gap-2'>
                <button type='button' className='text-sm text-accent' onClick={() => navigate('/series')}>
                    {t('challengeSeries.backToList')}
                </button>

                <button type='button' className='text-sm text-accent' onClick={() => navigate('/challenges')}>
                    {t('challengeSeries.backToChallenges')}
                </button>
            </div>

            {loading ? (
                <div className='space-y-4'>
                    <div className='rounded-xl border border-border/70 bg-surface p-4'>
                        <div className='animate-pulse space-y-3'>
                            <div className='h-6 w-1/3 rounded bg-surface-muted' />
                            <div className='h-4 w-3/4 rounded bg-surface-muted' />
                        </div>
                    </div>
                    <div className='-mx-4 md:mx-0 overflow-visible md:overflow-hidden rounded-none md:rounded-xl bg-transparent'>
                        <div className='space-y-2 px-4 md:hidden'>
                            {Array.from({ length: 5 }, (_, idx) => (
                                <div key={`series-detail-mobile-skeleton-${idx}`} className='w-full rounded-xl border border-border/60 bg-surface p-3'>
                                    <div className='flex items-start gap-3 animate-pulse'>
                                        <div className='h-8 w-8 shrink-0 rounded-full bg-surface-muted' />
                                        <div className='min-w-0 flex-1 space-y-2'>
                                            <div className='h-4 w-3/5 rounded bg-surface-muted' />
                                            <div className='h-3 w-1/3 rounded bg-surface-muted' />
                                            <div className='mt-1 flex flex-wrap gap-x-4 gap-y-1'>
                                                <div className='h-3 w-20 rounded bg-surface-muted' />
                                                <div className='h-3 w-28 rounded bg-surface-muted' />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div className='hidden overflow-x-auto md:block'>
                            <div className='min-w-150'>
                                <div className='grid grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] bg-surface px-4 py-2 text-[12px] text-text-muted'>
                                    <span>{t('challenges.tableProblem')}</span>
                                    <span>{t('common.category')}</span>
                                    <span>{t('challenges.tableSolveCount')}</span>
                                    <span>{t('challenges.tableAuthor')}</span>
                                </div>
                                <div>
                                    {Array.from({ length: 5 }, (_, idx) => (
                                        <div
                                            key={`series-detail-desktop-skeleton-${idx}`}
                                            className='grid w-full grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] items-center px-4 py-3'
                                        >
                                            <div className='min-w-0 flex items-center gap-3 animate-pulse'>
                                                <div className='h-8 w-8 shrink-0 rounded-full bg-surface-muted' />
                                                <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                            </div>
                                            <div className='h-3 w-2/5 rounded bg-surface-muted animate-pulse' />
                                            <div className='h-3 w-10 rounded bg-surface-muted animate-pulse' />
                                            <div className='h-3 w-4/5 rounded bg-surface-muted animate-pulse' />
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            ) : errorMessage ? (
                <div className='rounded-xl border border-danger/40 bg-danger/10 p-6 text-sm text-danger'>{errorMessage}</div>
            ) : !series ? (
                <div className='rounded-xl border border-border/60 bg-surface p-6 text-sm text-text-muted'>{t('challengeSeries.notFound')}</div>
            ) : (
                <>
                    <div className='my-6'>
                        <h2 className='text-3xl text-text'>{series.title}</h2>
                        <p className='mt-2 text-sm text-text-muted'>{series.description}</p>
                    </div>

                    <div className='-mx-4 md:mx-0 overflow-visible md:overflow-hidden rounded-none md:rounded-xl bg-transparent'>
                        {challenges.length === 0 ? (
                            <div className='px-4 py-8 text-sm text-text-muted'>{t('challengeSeries.emptyChallenges')}</div>
                        ) : (
                            <>
                                <div className='space-y-2 px-4 md:hidden'>
                                    {challenges.map((challenge) => {
                                        const category = 'category' in challenge ? challenge.category : t('common.na')
                                        const solveCount = 'solve_count' in challenge ? challenge.solve_count : 0
                                        const inactive = challenge.is_active === false
                                        const authorName = challenge.created_by?.username?.trim()
                                        const author = authorName && authorName !== '' ? authorName : t('common.na')

                                        return (
                                            <button
                                                key={challenge.id}
                                                type='button'
                                                className='w-full rounded-xl border border-border/60 bg-surface p-3 text-left transition hover:bg-surface-muted disabled:cursor-not-allowed disabled:opacity-70'
                                                disabled={inactive}
                                                onClick={() => {
                                                    if (!inactive) navigate(`/challenges/${challenge.id}`)
                                                }}
                                            >
                                                <div className='flex items-start gap-3'>
                                                    <LevelBadge level={challenge.level} />
                                                    <div className='min-w-0 flex-1'>
                                                        <div className='flex items-center gap-1.5'>
                                                            <p className='truncate text-sm font-semibold text-text'>{challenge.title}</p>
                                                            {challenge.is_locked ? (
                                                                <span className='shrink-0 h-4 w-4 text-warning' title={t('challenge.lockedLabel')}>
                                                                    <svg viewBox='0 0 24 24' className='h-full w-full' fill='none' stroke='currentColor' strokeWidth='2'>
                                                                        <rect x='5' y='11' width='14' height='9' rx='2' />
                                                                        <path d='M8 11V8a4 4 0 1 1 8 0v3' />
                                                                    </svg>
                                                                </span>
                                                            ) : challenge.is_solved ? (
                                                                <FlagIcon className='shrink-0 h-4 w-4 text-accent' />
                                                            ) : null}
                                                        </div>
                                                        <p className='mt-1 text-xs text-text-muted'>{t(getCategoryKey(category))}</p>
                                                        <div className='mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-subtle'>
                                                            <span>
                                                                {t('challenges.tableSolveCount')}: {solveCount}
                                                            </span>
                                                            <span className='truncate'>
                                                                {t('challenges.tableAuthor')}: {author}
                                                            </span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </button>
                                        )
                                    })}
                                </div>

                                <div className='hidden overflow-x-auto md:block'>
                                    <div className='min-w-150'>
                                        <div className='grid grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] bg-surface px-4 py-2 text-[12px] text-text-muted'>
                                            <span>{t('challenges.tableProblem')}</span>
                                            <span>{t('common.category')}</span>
                                            <span>{t('challenges.tableSolveCount')}</span>
                                            <span>{t('challenges.tableAuthor')}</span>
                                        </div>
                                        <div>
                                            {challenges.map((challenge) => {
                                                const category = 'category' in challenge ? challenge.category : t('common.na')
                                                const solveCount = 'solve_count' in challenge ? challenge.solve_count : 0
                                                const inactive = challenge.is_active === false
                                                const authorName = challenge.created_by?.username?.trim()
                                                const author = authorName && authorName !== '' ? authorName : t('common.na')

                                                return (
                                                    <button
                                                        key={challenge.id}
                                                        type='button'
                                                        className='grid w-full grid-cols-[minmax(160px,2fr)_1fr_70px_100px] sm:grid-cols-[minmax(200px,2fr)_1fr_80px_110px] lg:grid-cols-[minmax(220px,2fr)_1fr_90px_120px] items-center px-4 py-3 text-left transition hover:bg-surface-muted disabled:cursor-not-allowed disabled:opacity-70'
                                                        disabled={inactive}
                                                        onClick={() => {
                                                            if (!inactive) navigate(`/challenges/${challenge.id}`)
                                                        }}
                                                    >
                                                        <div className='min-w-0 flex items-center gap-3'>
                                                            <LevelBadge level={challenge.level} />
                                                            <div className='min-w-0 flex flex-1 items-center'>
                                                                <span className='truncate pr-4 text-[14px] font-semibold sm:text-[16px]'>{challenge.title}</span>
                                                                {challenge.is_locked ? (
                                                                    <span className='-ml-1.5 h-4 w-4 shrink-0 text-warning' title={t('challenge.lockedLabel')}>
                                                                        <svg viewBox='0 0 24 24' className='h-full w-full' fill='none' stroke='currentColor' strokeWidth='2'>
                                                                            <rect x='5' y='11' width='14' height='9' rx='2' />
                                                                            <path d='M8 11V8a4 4 0 1 1 8 0v3' />
                                                                        </svg>
                                                                    </span>
                                                                ) : challenge.is_solved ? (
                                                                    <FlagIcon className='-ml-1.5 h-4 w-4 shrink-0 text-accent' />
                                                                ) : null}
                                                            </div>
                                                        </div>

                                                        <span className='wrap-break-words text-xs text-text-muted'>{t(getCategoryKey(category))}</span>
                                                        <span className='text-sm text-text-muted'>{solveCount}</span>
                                                        <span className='truncate text-xs text-text-muted'>{author}</span>
                                                    </button>
                                                )
                                            })}
                                        </div>
                                    </div>
                                </div>
                            </>
                        )}
                    </div>
                </>
            )}
        </section>
    )
}

export default ChallengeSeriesDetail
