import { useEffect, useMemo, useState } from 'react'
import { ApiError } from '../lib/api'
import type { Challenge, ChallengeSolver, PaginationMeta, Stack } from '../lib/types'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { getCategoryKey, getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import LoginRequired from '../components/LoginRequired'
import Markdown from '../components/Markdown'
import UserAvatar from '../components/UserAvatar'
import { DifficultyBadge } from './Challenges'

interface RouteProps {
    routeParams?: Record<string, string>
}

interface SubmissionState {
    status: 'idle' | 'loading' | 'success' | 'error'
    message?: string
}

const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: 5, total_count: 0, total_pages: 0, has_prev: false, has_next: false }

const ChallengeDetail = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()
    const challengeId = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])

    const [challenge, setChallenge] = useState<Challenge | null>(null)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const [flagInput, setFlagInput] = useState('')
    const [submission, setSubmission] = useState<SubmissionState>({ status: 'idle' })

    const readSolverPageFromQuery = () => {
        if (typeof window === 'undefined') return 1
        const params = new URLSearchParams(window.location.search)
        const value = Number(params.get('solver_page'))
        return Number.isInteger(value) && value > 0 ? value : 1
    }
    const [solvers, setSolvers] = useState<ChallengeSolver[]>([])
    const [solverPage, setSolverPage] = useState(readSolverPageFromQuery)
    const [solverPagination, setSolverPagination] = useState<PaginationMeta>(EMPTY_PAGINATION)

    const [downloadLoading, setDownloadLoading] = useState(false)
    const [downloadMessage, setDownloadMessage] = useState('')
    const [stackInfo, setStackInfo] = useState<Stack | null>(null)
    const [stackLoading, setStackLoading] = useState(false)
    const [stackMessage, setStackMessage] = useState('')

    const pushSolverPageQuery = (nextPage: number) => {
        if (typeof window === 'undefined') return
        const params = new URLSearchParams(window.location.search)
        if (nextPage > 1) params.set('solver_page', String(nextPage))
        else params.delete('solver_page')
        const query = params.toString()
        const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname
        const currentURL = `${window.location.pathname}${window.location.search}`
        if (nextURL !== currentURL) {
            window.history.pushState({}, '', nextURL)
        }
    }

    const loadChallenge = async () => {
        if (!challengeId) return
        setLoading(true)
        setErrorMessage('')

        try {
            const data = await api.challenge(challengeId)
            setChallenge(data)
        } catch (error) {
            setErrorMessage(formatApiError(error, t).message)
            setChallenge(null)
        } finally {
            setLoading(false)
        }
    }

    const loadSolvers = async (page: number) => {
        if (!challengeId) return

        try {
            const data = await api.challengeSolvers(challengeId, page, 5)
            setSolvers(data.solvers)
            setSolverPagination(data.pagination)
        } catch {
            setSolvers([])
            setSolverPagination(EMPTY_PAGINATION)
        }
    }

    useEffect(() => {
        if (!auth.user || !challengeId) return
        void loadChallenge()
    }, [auth.user?.id, challengeId])

    const loadStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || challenge.is_solved || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
        setStackLoading(true)
        setStackMessage('')
        try {
            const stack = await api.getStack(challengeId)
            setStackInfo(stack)
        } catch (error) {
            if (error instanceof ApiError && error.status === 404) {
                setStackInfo(null)
            } else {
                setStackMessage(formatApiError(error, t).message)
            }
        } finally {
            setStackLoading(false)
        }
    }

    useEffect(() => {
        if (!auth.user || !challengeId) return
        void loadSolvers(solverPage)
    }, [auth.user?.id, challengeId, solverPage])

    useEffect(() => {
        if (!auth.user || !challengeId || !challenge || challenge.is_locked || challenge.is_solved || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
        void loadStack()
    }, [auth.user?.id, challengeId, challenge?.id, challenge?.is_locked])

    useEffect(() => {
        const onPopState = () => {
            setSolverPage(readSolverPageFromQuery())
        }
        window.addEventListener('popstate', onPopState)
        return () => window.removeEventListener('popstate', onPopState)
    }, [])

    const submitFlag = async () => {
        if (!challengeId || !challenge || challenge.is_locked || challenge.is_active === false) return
        if (challenge.is_solved) {
            setSubmission({ status: 'success', message: t('challenge.correct') })
            return
        }
        if (submission.status === 'loading') return

        setSubmission({ status: 'loading' })

        try {
            const result = await api.submitFlag(challengeId, flagInput)
            if (result.correct) {
                setSubmission({ status: 'success', message: t('challenge.correct') })
                setFlagInput('')
                await loadChallenge()
                await loadSolvers(solverPage)
            } else {
                setSubmission({ status: 'error', message: t('challenge.incorrect') })
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                setSubmission({ status: 'success', message: t('challenge.correct') })
                setFlagInput('')
                await loadChallenge()
                await loadSolvers(solverPage)
                return
            }
            setSubmission({ status: 'error', message: formatApiError(error, t).message })
        }
    }

    const downloadFile = async () => {
        if (!challengeId || !challenge || !('has_file' in challenge) || !challenge.has_file || downloadLoading) return
        setDownloadLoading(true)
        setDownloadMessage('')
        try {
            const result = await api.requestChallengeFileDownload(challengeId)
            window.open(result.url, '_blank', 'noopener')
        } catch (error) {
            setDownloadMessage(formatApiError(error, t).message)
        } finally {
            setDownloadLoading(false)
        }
    }

    const createStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true || stackLoading) return
        setStackLoading(true)
        setStackMessage('')
        try {
            const created = await api.createStack(challengeId)
            setStackInfo(created)
        } catch (error) {
            setStackMessage(formatApiError(error, t).message)
        } finally {
            setStackLoading(false)
        }
    }

    const deleteStack = async () => {
        if (!challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true || stackLoading) return
        setStackLoading(true)
        setStackMessage('')
        try {
            await api.deleteStack(challengeId)
            setStackInfo(null)
        } catch (error) {
            setStackMessage(formatApiError(error, t).message)
        } finally {
            setStackLoading(false)
        }
    }

    const formatTimestamp = (value: string) => formatDateTime(value, localeTag)
    const firstBloodSolver = useMemo(() => solvers.find((solver) => solver.is_first_blood) ?? null, [solvers])

    if (!auth.user) {
        return <LoginRequired title={t('challenges.title')} />
    }

    if (!challengeId) {
        return (
            <section className='animate'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{t('errors.invalid')}</div>
            </section>
        )
    }

    if (loading) {
        return (
            <section className='animate'>
                <div className='border border-border bg-surface p-8 text-sm text-text-muted'>{t('common.loading')}</div>
            </section>
        )
    }

    if (errorMessage || !challenge) {
        return (
            <section className='animate space-y-3'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{errorMessage || t('errors.notFound')}</div>
            </section>
        )
    }

    const detail = challenge.is_locked ? null : challenge
    const isChallengeActive = challenge.is_locked ? false : challenge.is_active !== false
    const isSubmissionDisabled = !isChallengeActive || challenge.is_solved || submission.status === 'loading'

    return (
        <section className='animate space-y-4 px-3 sm:px-4 md:px-2 lg:px-0'>
            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <div className='space-y-4'>
                    <div className='rounded-2xl border border-border/20 bg-surface p-5 shadow-sm lg:hidden'>
                        <div className='flex items-center gap-3 min-w-0'>
                            <DifficultyBadge level={challenge.level} />
                            <span className='text-sm font-semibold text-accent'>{t('challenge.levelLabel', { level: challenge.level })}</span>
                        </div>

                        <h1 className='mt-3 wrap-break-word text-xl font-semibold leading-tight text-text sm:text-2xl lg:text-3xl'>{challenge.title}</h1>

                        <div className='mt-4 inline-flex rounded-lg bg-surface-muted px-2.5 py-1 text-xs text-text-muted'>{t(getCategoryKey(challenge.category))}</div>

                        <div className='mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-text-muted'>
                            <span>{t('common.pointsShort', { points: challenge.points })}</span>
                            <span>{t('challenge.solvedCount', { count: challenge.solve_count })}</span>
                        </div>
                    </div>

                    <div className='rounded-2xl p-4 sm:p-5'>
                        <h2 className='text-base font-semibold text-text'>{t('common.description')}</h2>

                        {challenge.is_locked ? (
                            <div className='mt-3 rounded-xl bg-warning/10 p-4 text-sm text-warning'>
                                <p>{t('challenge.lockedNotice')}</p>

                                <button
                                    className='mt-3 rounded-md bg-warning px-3 py-1.5 text-xs text-white hover:bg-warning-strong disabled:opacity-60'
                                    onClick={() => {
                                        if (challenge.previous_challenge_id) {
                                            navigate(`/challenges/${challenge.previous_challenge_id}`)
                                        }
                                    }}
                                    disabled={!challenge.previous_challenge_id}
                                >
                                    {challenge.previous_challenge_category} - {challenge.previous_challenge_title}
                                </button>
                            </div>
                        ) : (
                            <div className='mt-3'>
                                <Markdown className='wrap-break-word text-sm text-text' content={detail?.description ?? ''} />
                            </div>
                        )}

                        <div className='mt-6 space-y-8 lg:hidden'>
                            <section className='space-y-3 px-1'>
                                <h2 className='text-xl font-semibold text-text'>{t('challenges.tableAuthor')}</h2>

                                <div className='rounded-2xl bg-surface/70'>
                                    {challenge.created_by_username && challenge.created_by_username.trim() !== '' ? (
                                        <div className='flex items-center gap-3.75'>
                                            <UserAvatar username={challenge.created_by_username} size='md' />
                                            <div className='text-lg font-semibold text-text'>{challenge.created_by_username}</div>
                                        </div>
                                    ) : (
                                        <div className='text-lg font-semibold text-text'>{t('common.na')}</div>
                                    )}
                                </div>
                            </section>

                            {firstBloodSolver ? (
                                <section className='space-y-3 px-1'>
                                    <h2 className='flex items-center gap-2 text-xl font-semibold text-danger'>
                                        <svg viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' className='h-5 w-5'>
                                            <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity='0.2' />
                                            <path
                                                d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z'
                                                fill='none'
                                                stroke='currentColor'
                                                strokeLinecap='round'
                                                strokeLinejoin='round'
                                            />
                                        </svg>
                                        {t('leaderboard.firstBlood')}
                                    </h2>

                                    <div className='rounded-2xl bg-surface/70'>
                                        <div className='flex items-start justify-between gap-4 py-2'>
                                            <div className='min-w-0 flex-1 flex items-center gap-3.75'>
                                                <UserAvatar username={firstBloodSolver.username} size='md' />
                                                <div className='min-w-0'>
                                                    <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${firstBloodSolver.user_id}`)}>
                                                        {firstBloodSolver.username}
                                                    </button>
                                                    <p className='mt-1 text-sm text-text-subtle'>{formatTimestamp(firstBloodSolver.solved_at)}</p>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </section>
                            ) : null}

                            <section className='space-y-3 px-1'>
                                <h2 className='text-xl font-semibold text-text'>{t('challenge.recentSolversTitle')}</h2>

                                <div className='space-y-3'>
                                    {solvers.length === 0 ? (
                                        <p className='text-sm text-text-muted'>{t('challenge.noSolversYet')}</p>
                                    ) : (
                                        solvers.map((solver, index) => (
                                            <div key={`${solver.user_id}-${index}`} className='flex items-start justify-between gap-4 py-2'>
                                                <div className='min-w-0 flex-1 flex items-center gap-3.75'>
                                                    <UserAvatar username={solver.username} size='md' />
                                                    <div className='min-w-0'>
                                                        <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${solver.user_id}`)}>
                                                            {solver.username}
                                                        </button>

                                                        <p className='mt-1 text-sm text-text-subtle'>{formatTimestamp(solver.solved_at)}</p>
                                                    </div>
                                                </div>

                                                <span className='shrink-0 text-sm text-text-subtle'>{index + 1}</span>
                                            </div>
                                        ))
                                    )}
                                </div>

                                <div className='flex items-center justify-between pt-2 text-sm text-text-muted'>
                                    <span>
                                        {solverPagination.page} / {solverPagination.total_pages || 1}
                                    </span>

                                    <div className='flex gap-2'>
                                        <button
                                            className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                            disabled={!solverPagination.has_prev}
                                            onClick={() => {
                                                const next = Math.max(1, solverPage - 1)
                                                setSolverPage(next)
                                                pushSolverPageQuery(next)
                                            }}
                                        >
                                            {t('common.previous')}
                                        </button>

                                        <button
                                            className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                            disabled={!solverPagination.has_next}
                                            onClick={() => {
                                                const next = solverPage + 1
                                                setSolverPage(next)
                                                pushSolverPageQuery(next)
                                            }}
                                        >
                                            {t('common.next')}
                                        </button>
                                    </div>
                                </div>
                            </section>
                        </div>

                        {!challenge.is_locked && detail?.has_file && (
                            <div className='mt-12'>
                                <button
                                    onClick={downloadFile}
                                    disabled={downloadLoading}
                                    className='w-full h-10 flex items-center justify-center gap-2 rounded-sm border border-border bg-surface-muted px-4 py-4 text-sm font-medium text-accent hover:bg-surface-subtle transition disabled:opacity-60'
                                >
                                    <svg xmlns='http://www.w3.org/2000/svg' className='h-4 w-4' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                                        <path strokeLinecap='round' strokeLinejoin='round' d='M12 3v12m0 0l-4-4m4 4l4-4M4 17h16' />
                                    </svg>

                                    {downloadLoading ? t('challenge.downloadPreparing') : t('challenge.download')}
                                </button>

                                {downloadMessage && <p className='mt-2 text-xs text-danger'>{downloadMessage}</p>}
                            </div>
                        )}

                        {!challenge.is_locked && !challenge.is_solved && 'stack_enabled' in challenge && challenge.stack_enabled ? (
                            <div className='rounded-md border border-border/30 bg-surface p-4 sm:p-5 shadow-sm mt-8'>
                                <div className='flex items-center justify-between gap-2'>
                                    <h2 className='text-base font-semibold text-text'>{t('challenge.stackInstance')}</h2>
                                    <button className='rounded-lg bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle disabled:opacity-60' onClick={() => void loadStack()} disabled={stackLoading}>
                                        {t('common.refresh')}
                                    </button>
                                </div>

                                {stackInfo ? (
                                    <div className='mt-3 space-y-1.5 text-sm text-text-muted'>
                                        <p>
                                            {t('challenge.stackStatus')} <span className='text-text'>{stackInfo.status}</span>
                                        </p>
                                        <p>
                                            {t('challenge.stackEndpoint')}{' '}
                                            <span className='break-all text-text'>
                                                {stackInfo.node_public_ip && stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.protocol} ${stackInfo.node_public_ip}:${port.node_port}`).join(', ') : t('common.pending')}
                                            </span>
                                        </p>
                                        <p>
                                            {t('challenge.stackPorts')}{' '}
                                            <span className='text-text'>{stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.container_port}/${port.protocol}`).join(', ') : t('common.pending')}</span>
                                        </p>
                                        <p>
                                            {t('challenge.stackTtl')} <span className='text-text'>{stackInfo.ttl_expires_at ? formatTimestamp(stackInfo.ttl_expires_at) : t('common.pending')}</span>
                                        </p>
                                    </div>
                                ) : (
                                    <p className='mt-3 text-sm text-text-muted'>{t('challenge.stackNoActive')}</p>
                                )}

                                <div className='mt-4 flex flex-wrap gap-2'>
                                    {stackInfo ? (
                                        <button className='rounded-lg border border-danger/20 px-3 py-2 text-sm text-danger hover:border-danger/40 disabled:opacity-60' onClick={deleteStack} disabled={stackLoading}>
                                            {stackLoading ? t('challenge.stackWorking') : t('challenge.deleteStack')}
                                        </button>
                                    ) : (
                                        <button className='rounded-lg bg-accent px-3 py-2 text-sm text-white hover:bg-accent-strong disabled:opacity-60' onClick={createStack} disabled={stackLoading || challenge.is_solved}>
                                            {stackLoading ? t('challenge.stackWorking') : t('challenge.createStack')}
                                        </button>
                                    )}
                                </div>

                                {stackMessage ? <p className='mt-2 text-xs text-danger'>{stackMessage}</p> : null}
                            </div>
                        ) : null}

                        {!challenge.is_locked && !challenge.is_solved && (
                            <form
                                className='rounded-md bg-surface-muted p-4 sm:p-5 mt-4 shadow-sm border border-border/30'
                                onSubmit={(e) => {
                                    e.preventDefault()
                                    void submitFlag()
                                }}
                            >
                                <label className='text-sm font-semibold text-text'>{t('challenge.enterFlag')}</label>

                                <div className='mt-3 flex flex-col gap-2 sm:flex-row'>
                                    <input
                                        className='min-w-0 flex-1 rounded-md border border-border/40 bg-surface px-3 py-2.5 text-sm text-text focus:border-accent focus:outline-none'
                                        type='text'
                                        value={flagInput}
                                        onChange={(e) => setFlagInput(e.target.value)}
                                        disabled={isSubmissionDisabled}
                                    />

                                    <button className='w-full rounded-md bg-accent px-4 py-2.5 text-sm text-white hover:bg-accent-strong disabled:opacity-60 sm:w-auto sm:min-w-30' disabled={isSubmissionDisabled}>
                                        {submission.status === 'loading' ? t('challenge.submitting') : t('challenge.submit')}
                                    </button>
                                </div>

                                {submission.message && <p className={`mt-2 text-sm ${submission.status === 'success' ? 'text-success' : 'text-danger'}`}>{submission.message}</p>}
                            </form>
                        )}
                    </div>
                </div>

                <aside className='hidden lg:block lg:sticky'>
                    <div className='space-y-8'>
                        <div className='rounded-2xl border border-border/20 bg-surface p-5 shadow-sm'>
                            <div className='flex items-center gap-3 min-w-0'>
                                <DifficultyBadge level={challenge.level} />
                                <span className='text-sm font-semibold text-accent'>{t('challenge.levelLabel', { level: challenge.level })}</span>
                            </div>

                            <h1 className='mt-3 wrap-break-word text-xl font-semibold leading-tight text-text sm:text-2xl lg:text-3xl'>{challenge.title}</h1>

                            <div className='mt-4 inline-flex rounded-lg bg-surface-muted px-2.5 py-1 text-xs text-text-muted'>{t(getCategoryKey(challenge.category))}</div>

                            <div className='mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-text-muted'>
                                <span>{t('common.pointsShort', { points: challenge.points })}</span>
                                <span>{t('challenge.solvedCount', { count: challenge.solve_count })}</span>
                            </div>
                        </div>

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('challenges.tableAuthor')}</h2>

                            <div className='rounded-2xl bg-surface/70'>
                                {challenge.created_by_username && challenge.created_by_username.trim() !== '' ? (
                                    <div className='flex items-center gap-3.75'>
                                        <UserAvatar username={challenge.created_by_username} size='md' />
                                        <div className='text-lg font-semibold text-text'>{challenge.created_by_username}</div>
                                    </div>
                                ) : (
                                    <div className='text-lg font-semibold text-text'>{t('common.na')}</div>
                                )}
                            </div>
                        </section>

                        {firstBloodSolver ? (
                            <section className='space-y-3 px-1'>
                                <h2 className='flex items-center gap-2 text-xl font-semibold text-danger'>
                                    <svg viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg' className='h-5 w-5'>
                                        <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity='0.2' />
                                        <path
                                            d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z'
                                            fill='none'
                                            stroke='currentColor'
                                            strokeLinecap='round'
                                            strokeLinejoin='round'
                                        />
                                    </svg>
                                    {t('leaderboard.firstBlood')}
                                </h2>

                                <div className='rounded-2xl bg-surface/70'>
                                    <div className='flex items-start justify-between gap-4 py-2'>
                                        <div className='min-w-0 flex-1 flex items-center gap-3.75'>
                                            <UserAvatar username={firstBloodSolver.username} size='md' />
                                            <div className='min-w-0'>
                                                <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${firstBloodSolver.user_id}`)}>
                                                    {firstBloodSolver.username}
                                                </button>
                                                <p className='mt-1 text-sm text-text-subtle'>{formatTimestamp(firstBloodSolver.solved_at)}</p>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </section>
                        ) : null}

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('challenge.recentSolversTitle')}</h2>

                            <div className='space-y-3'>
                                {solvers.length === 0 ? (
                                    <p className='text-sm text-text-muted'>{t('challenge.noSolversYet')}</p>
                                ) : (
                                    solvers.map((solver, index) => (
                                        <div key={`${solver.user_id}-${index}`} className='flex items-start justify-between gap-4 py-2'>
                                            <div className='min-w-0 flex-1 flex items-center gap-3.75'>
                                                <UserAvatar username={solver.username} size='md' />
                                                <div className='min-w-0'>
                                                    <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${solver.user_id}`)}>
                                                        {solver.username}
                                                    </button>

                                                    <p className='mt-1 text-sm text-text-subtle'>{formatTimestamp(solver.solved_at)}</p>
                                                </div>
                                            </div>

                                            <span className='shrink-0 text-sm text-text-subtle'>{index + 1}</span>
                                        </div>
                                    ))
                                )}
                            </div>

                            <div className='flex items-center justify-between pt-2 text-sm text-text-muted'>
                                <span>
                                    {solverPagination.page} / {solverPagination.total_pages || 1}
                                </span>

                                <div className='flex gap-2'>
                                    <button
                                        className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                        disabled={!solverPagination.has_prev}
                                        onClick={() => {
                                            const next = Math.max(1, solverPage - 1)
                                            setSolverPage(next)
                                            pushSolverPageQuery(next)
                                        }}
                                    >
                                        {t('common.previous')}
                                    </button>

                                    <button
                                        className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                        disabled={!solverPagination.has_next}
                                        onClick={() => {
                                            const next = solverPage + 1
                                            setSolverPage(next)
                                            pushSolverPageQuery(next)
                                        }}
                                    >
                                        {t('common.next')}
                                    </button>
                                </div>
                            </div>
                        </section>
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default ChallengeDetail
