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

interface RouteProps {
    routeParams?: Record<string, string>
}

interface SubmissionState {
    status: 'idle' | 'loading' | 'success' | 'error'
    message?: string
}

const EMPTY_PAGINATION: PaginationMeta = { page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false }

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

    const challengesBackURL = useMemo(() => {
        if (typeof window === 'undefined') return '/challenges'
        const params = new URLSearchParams(window.location.search)
        params.delete('solver_page')
        const query = params.toString()
        return query ? `/challenges?${query}` : '/challenges'
    }, [routeParams.id, solverPage])

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
            const data = await api.challengeSolvers(challengeId, page, 20)
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
        if (!challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
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
        if (!auth.user || !challengeId || !challenge || challenge.is_locked || !('stack_enabled' in challenge) || challenge.stack_enabled !== true) return
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
                <button className='border border-border bg-surface px-3 py-1 text-xs text-text-muted hover:bg-surface-muted' onClick={() => navigate(challengesBackURL)}>
                    ← {t('profile.backToUsers')}
                </button>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{errorMessage || t('errors.notFound')}</div>
            </section>
        )
    }

    const detail = challenge.is_locked ? null : challenge
    const isChallengeActive = challenge.is_locked ? false : challenge.is_active !== false
    const isSubmissionDisabled = !isChallengeActive || challenge.is_solved || submission.status === 'loading'

    return (
        <section className='animate space-y-4 px-0 md:px-2 lg:px-0'>
            <button className='text-xs text-text-muted hover:text-text' onClick={() => navigate(challengesBackURL)}>
                ← {t('challenge.backToChallenges')}
            </button>

            <div className='grid gap-4 grid-cols-1 lg:grid-cols-[1.9fr_0.9fr]'>
                <div className='space-y-3'>
                    <div className='md:p-4'>
                        <h1 className='text-xl sm:text-2xl font-semibold text-text wrap-break-word'>{challenge.title}</h1>
                    </div>

                    <hr className='border-border' />

                    {challenge.is_locked ? (
                        <div className='rounded-none md:rounded-xl bg-warning/10 p-4 text-sm text-warning'>
                            <p>{t('challenge.lockedNotice')}</p>

                            <button
                                className='mt-3 rounded-md bg-warning px-3 py-1 text-xs text-white hover:bg-warning-strong'
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
                        <div className='md:p-4'>
                            <Markdown className='wrap-break-word text-sm text-text' content={detail?.description ?? ''} />
                        </div>
                    )}

                    {!challenge.is_locked && detail?.has_file && (
                        <div className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm'>
                            <div className='flex items-center justify-between gap-3'>
                                <div className='min-w-0'>
                                    <p className='text-sm font-medium text-text'>{t('challenge.fileTitle')}</p>
                                    <p className='text-xs text-text-subtle truncate'>{detail.file_name ?? 'challenge.zip'}</p>
                                </div>

                                <button className='rounded-md bg-accent px-3 py-1 text-xs text-white hover:bg-accent-strong disabled:opacity-60' onClick={downloadFile} disabled={downloadLoading}>
                                    {downloadLoading ? t('challenge.downloadPreparing') : t('challenge.download')}
                                </button>
                            </div>

                            {downloadMessage && <p className='mt-2 text-xs text-danger'>{downloadMessage}</p>}
                        </div>
                    )}

                    {!challenge.is_locked && 'stack_enabled' in challenge && challenge.stack_enabled ? (
                        <div className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm'>
                            <div className='flex items-center justify-between gap-2'>
                                <h2 className='text-sm font-semibold text-text'>{t('challenge.stackInstance')}</h2>
                                <button className='rounded-md bg-surface-muted px-3 py-1 text-xs text-text hover:bg-surface-subtle disabled:opacity-60' onClick={() => void loadStack()} disabled={stackLoading}>
                                    {t('common.refresh')}
                                </button>
                            </div>
                            {stackInfo ? (
                                <div className='mt-3 space-y-1 text-xs text-text-muted'>
                                    <p>
                                        {t('challenge.stackStatus')} <span className='text-text'>{stackInfo.status}</span>
                                    </p>
                                    <p>
                                        {t('challenge.stackEndpoint')}{' '}
                                        <span className='text-text'>
                                            {stackInfo.node_public_ip && stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.protocol} ${stackInfo.node_public_ip}:${port.node_port}`).join(', ') : t('common.pending')}
                                        </span>
                                    </p>
                                    <p>
                                        {t('challenge.stackPorts')} <span className='text-text'>{stackInfo.ports.length > 0 ? stackInfo.ports.map((port) => `${port.container_port}/${port.protocol}`).join(', ') : t('common.pending')}</span>
                                    </p>
                                    <p>
                                        {t('challenge.stackTtl')} <span className='text-text'>{stackInfo.ttl_expires_at ? formatTimestamp(stackInfo.ttl_expires_at) : t('common.pending')}</span>
                                    </p>
                                </div>
                            ) : (
                                <p className='mt-3 text-sm text-text-muted'>{t('challenge.stackNoActive')}</p>
                            )}
                            <div className='mt-3 flex gap-2'>
                                {stackInfo ? (
                                    <button className='rounded-md border border-danger/30 px-3 py-1 text-xs text-danger hover:border-danger/50 disabled:opacity-60' onClick={deleteStack} disabled={stackLoading}>
                                        {stackLoading ? t('challenge.stackWorking') : t('challenge.deleteStack')}
                                    </button>
                                ) : (
                                    <button className='rounded-md bg-accent px-3 py-1 text-xs text-white hover:bg-accent-strong disabled:opacity-60' onClick={createStack} disabled={stackLoading || challenge.is_solved}>
                                        {stackLoading ? t('challenge.stackWorking') : t('challenge.createStack')}
                                    </button>
                                )}
                            </div>
                            {stackMessage ? <p className='mt-2 text-xs text-danger'>{stackMessage}</p> : null}
                        </div>
                    ) : null}

                    {!challenge.is_locked && (
                        <form
                            className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm'
                            onSubmit={(e) => {
                                e.preventDefault()
                                void submitFlag()
                            }}
                        >
                            <label className='text-xs text-text-muted'>{t('challenge.enterFlag')}</label>

                            <div className='mt-2 flex gap-2'>
                                <input
                                    className='flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                    type='text'
                                    value={flagInput}
                                    onChange={(e) => setFlagInput(e.target.value)}
                                    disabled={isSubmissionDisabled}
                                />

                                <button className='rounded-md bg-accent px-4 py-2 text-sm text-white hover:bg-accent-strong disabled:opacity-60' disabled={isSubmissionDisabled}>
                                    {submission.status === 'loading' ? t('challenge.submitting') : t('challenge.submit')}
                                </button>
                            </div>

                            {submission.message && <p className={`mt-2 text-sm ${submission.status === 'success' ? 'text-success' : 'text-danger'}`}>{submission.message}</p>}
                        </form>
                    )}
                </div>

                <aside className='space-y-3'>
                    <div className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm border border-border/60'>
                        <div className='flex items-center gap-2 text-accent text-sm font-semibold'>
                            <span>{t('challenge.levelLabel', { level: challenge.level })}</span>
                        </div>
                        <p className='mt-2 text-2xl font-semibold text-text wrap-break-word'>{challenge.title}</p>
                        <div className='mt-2 inline-flex rounded bg-surface-muted px-2 py-1 text-xs text-text-muted'>{t(getCategoryKey(challenge.category))}</div>
                        <div className='mt-3 space-y-1 text-xs text-text-muted'>
                            <p>{t('common.pointsShort', { points: challenge.points })}</p>
                            <p>{t('challenge.solvedCount', { count: challenge.solve_count })}</p>
                        </div>
                    </div>

                    <div className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm border border-border/60'>
                        <h2 className='text-sm font-semibold text-text'>{t('challenges.tableAuthor')}</h2>
                        <p className='mt-2 text-sm text-text'>{challenge.created_by_username && challenge.created_by_username.trim() !== '' ? challenge.created_by_username : t('common.na')}</p>
                    </div>

                    <div className='rounded-none md:rounded-xl bg-transparent md:bg-surface md:p-4 md:shadow-sm'>
                        <h2 className='text-sm font-semibold text-text'>{t('challenge.recentSolversTitle')}</h2>

                        <div className='mt-3 space-y-2'>
                            {solvers.length === 0 ? (
                                <p className='text-sm text-text-muted'>{t('challenge.noSolversYet')}</p>
                            ) : (
                                solvers.map((solver, index) => (
                                    <div key={`${solver.user_id}-${index}`} className='flex items-center justify-between rounded px-2 py-2 text-sm hover:bg-surface-muted'>
                                        <button className='text-accent hover:underline truncate' onClick={() => navigate(`/users/${solver.user_id}`)}>
                                            {solver.username}
                                        </button>

                                        <span className='text-xs text-text-subtle'>{formatTimestamp(solver.solved_at)}</span>
                                    </div>
                                ))
                            )}
                        </div>
                    </div>

                    <div className='flex items-center justify-between rounded-none md:rounded-xl bg-transparent md:bg-surface md:px-3 md:py-2 text-xs text-text-muted'>
                        <span>
                            {solverPagination.page} / {solverPagination.total_pages || 1}
                        </span>

                        <div className='flex gap-2'>
                            <button
                                className='rounded bg-surface-muted px-2 py-1 disabled:opacity-50'
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
                                className='rounded bg-surface-muted px-2 py-1 disabled:opacity-50'
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
                </aside>
            </div>
        </section>
    )
}

export default ChallengeDetail
