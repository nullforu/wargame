import { useEffect, useMemo, useState } from 'react'
import { ApiError } from '../lib/api'
import { formatApiError, formatDateTime } from '../lib/utils'
import type { Challenge, Stack } from '../lib/types'
import { getCategoryKey, getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import Markdown from './Markdown'

interface SubmissionState {
    status: 'idle' | 'loading' | 'success' | 'error'
    message?: string
}

interface ChallengeModalProps {
    challenge: Challenge
    isSolved: boolean
    onClose: () => void
    onSolved: () => void
}

const STACK_POLL_FAST_MS = 10000
const STACK_POLL_SLOW_MS = 60000

const ChallengeModal = ({ challenge, isSolved, onClose, onSolved }: ChallengeModalProps) => {
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const [flagInput, setFlagInput] = useState('')
    const [submission, setSubmission] = useState<SubmissionState>({ status: 'idle' })
    const [downloadLoading, setDownloadLoading] = useState(false)
    const [downloadMessage, setDownloadMessage] = useState('')
    const [stackInfo, setStackInfo] = useState<Stack | null>(null)
    const [stackLoading, setStackLoading] = useState(false)
    const [stackMessage, setStackMessage] = useState('')
    const [stackPolling, setStackPolling] = useState(false)
    const [stackNextInterval, setStackNextInterval] = useState(STACK_POLL_FAST_MS)

    const isSuccessful = useMemo(() => submission.status === 'success', [submission.status])
    const isLocked = challenge.is_locked === true
    const detail = !isLocked && 'description' in challenge ? challenge : null
    const isActive = 'is_active' in challenge ? challenge.is_active !== false : true
    const categoryValue = 'category' in challenge ? challenge.category : ''
    const hasCategory = Boolean(categoryValue)
    const hasDescription = !!detail?.description
    const solveCount = 'solve_count' in challenge ? challenge.solve_count : null
    const hasFile = !!detail?.has_file
    const stackEnabled = !!detail?.stack_enabled
    const previousChallengeId = isLocked ? (challenge.previous_challenge_id ?? null) : (detail?.previous_challenge_id ?? null)
    const previousChallengeTitle = isLocked ? (challenge.previous_challenge_title ?? null) : null
    const previousChallengeCategory = isLocked ? (challenge.previous_challenge_category ?? null) : null

    const submitFlag = async () => {
        if (isLocked) {
            return
        }

        if (isSolved) {
            setSubmission({ status: 'success', message: t('challenge.correct') })
            return
        }

        if (submission.status === 'loading') return

        setSubmission({ status: 'loading' })

        try {
            const result = await api.submitFlag(challenge.id, flagInput)

            if (result.correct) {
                setSubmission({ status: 'success', message: t('challenge.correct') })
                setFlagInput('')
                onSolved()
            } else {
                setSubmission({ status: 'error', message: t('challenge.incorrect') })
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                setSubmission({ status: 'success', message: t('challenge.correct') })
                setFlagInput('')
                onSolved()
                return
            }

            const formatted = formatApiError(error, t)
            setSubmission({ status: 'error', message: formatted.message })
        }
    }

    const downloadFile = async () => {
        if (!hasFile || downloadLoading) return

        setDownloadLoading(true)
        setDownloadMessage('')

        try {
            const result = await api.requestChallengeFileDownload(challenge.id)
            window.open(result.url, '_blank', 'noopener')
        } catch (error) {
            const formatted = formatApiError(error, t)
            setDownloadMessage(formatted.message)
        } finally {
            setDownloadLoading(false)
        }
    }

    const formatTimestamp = (value?: string | null) => {
        if (!value) return t('common.na')
        return formatDateTime(value, localeTag)
    }

    const loadStack = async () => {
        if (!auth.user || !stackEnabled) return

        try {
            const result = await api.getStack(challenge.id)
            setStackInfo(result)
            setStackNextInterval(result?.status === 'running' ? STACK_POLL_SLOW_MS : STACK_POLL_FAST_MS)
            setStackMessage('')
        } catch (error) {
            if (error instanceof ApiError && error.status === 404) {
                setStackInfo(null)
                setStackNextInterval(STACK_POLL_FAST_MS)
                setStackMessage('')
                return
            }
            const formatted = formatApiError(error, t)
            setStackMessage(formatted.message)
        }
    }

    const createStack = async () => {
        if (isSolved) {
            setStackMessage(t('challenge.solvedCannotCreate'))
            return
        }
        if (stackLoading || !auth.user) return
        setStackLoading(true)
        setStackMessage('')

        try {
            const created = await api.createStack(challenge.id)
            setStackInfo(created)
        } catch (error) {
            const formatted = formatApiError(error, t)
            setStackMessage(formatted.message)
        } finally {
            setStackLoading(false)
        }
    }

    const deleteStack = async () => {
        if (stackLoading || !auth.user) return
        setStackLoading(true)
        setStackMessage('')

        try {
            await api.deleteStack(challenge.id)
            setStackInfo(null)
        } catch (error) {
            const formatted = formatApiError(error, t)
            setStackMessage(formatted.message)
        } finally {
            setStackLoading(false)
        }
    }

    useEffect(() => {
        if (!auth.user || !stackEnabled) {
            setStackInfo(null)
            setStackMessage('')
            setStackPolling(false)
            setStackNextInterval(STACK_POLL_FAST_MS)
            return
        }

        if (isSolved) {
            setStackPolling(false)
            return
        }

        loadStack()
    }, [auth.user, stackEnabled, challenge.id, isSolved])

    useEffect(() => {
        if (!auth.user || !stackEnabled || !stackInfo) {
            setStackPolling(false)
            return
        }

        setStackPolling(true)
        let timeoutId: ReturnType<typeof setTimeout>

        const poll = async () => {
            await loadStack()
            timeoutId = setTimeout(poll, stackNextInterval)
        }

        timeoutId = setTimeout(poll, stackNextInterval)
        return () => {
            clearTimeout(timeoutId)
            setStackPolling(false)
        }
    }, [auth.user, stackEnabled, stackInfo, stackNextInterval])

    return (
        <div
            className='fixed inset-0 z-50 flex items-center justify-center bg-overlay/50 p-4'
            onClick={(event) => {
                if (event.target === event.currentTarget) {
                    onClose()
                }
            }}
        >
            <div className='relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl border border-border bg-surface p-8'>
                <button className='absolute right-2 top-2 text-text-subtle hover:text-text cursor-pointer' onClick={onClose} aria-label={t('challenge.closeModal')}>
                    <svg className='h-6 w-6' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
                        <path strokeLinecap='round' strokeLinejoin='round' strokeWidth='2' d='M6 18L18 6M6 6l12 12' />
                    </svg>
                </button>

                <div className='flex items-start justify-between gap-4'>
                    <div>
                        <h2 className='text-2xl text-text'>{challenge.title}</h2>
                        <div className='mt-2 flex flex-wrap items-center gap-2 text-sm'>
                            {hasCategory ? <span className='rounded-full bg-surface-subtle px-3 py-1 text-xs font-medium text-text'>{t(getCategoryKey(categoryValue))}</span> : null}
                            <span className='text-text-muted'>{t('common.pointsShort', { points: challenge.points })}</span>
                            {solveCount !== null ? <span className='text-text-muted'>{t('challenge.solvedCount', { count: solveCount })}</span> : null}
                        </div>
                    </div>
                    {isLocked ? (
                        <span className='rounded-full bg-warning/20 px-4 py-1.5 text-sm text-warning-strong'>{t('challenge.lockedLabel')}</span>
                    ) : isSolved ? (
                        <span className='rounded-full bg-success/20 px-4 py-1.5 text-sm text-success'>{t('challenge.solvedLabel')}</span>
                    ) : !isActive ? (
                        <span className='rounded-full bg-surface/10 px-4 py-1.5 text-sm text-text-muted'>{t('challenge.inactiveLabel')}</span>
                    ) : null}
                </div>

                {isLocked ? (
                    <div className='mt-6 rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning-strong'>
                        <p>{t('challenge.lockedNotice')}</p>
                        {previousChallengeId ? (
                            <p className='mt-2 text-xs text-warning-strong'>
                                {t('challenge.lockedRequirement', {
                                    id: previousChallengeId,
                                    title: previousChallengeTitle ?? t('common.na'),
                                    category: previousChallengeCategory ?? t('common.na'),
                                })}
                            </p>
                        ) : null}
                    </div>
                ) : (
                    <div className='mt-6 text-text'>
                        <Markdown className='break-keep' content={hasDescription ? (detail?.description ?? '') : ''} />
                    </div>
                )}

                {hasFile ? (
                    <div className='mt-6'>
                        <div className='rounded-xl border border-border bg-surface-muted p-4 text-sm text-text'>
                            <div className='flex flex-wrap items-center justify-between gap-3'>
                                <div>
                                    <p className='font-medium'>{t('challenge.fileTitle')}</p>
                                    <p className='text-xs text-text-subtle'>{detail?.file_name ?? 'challenge.zip'}</p>
                                </div>
                                {auth.user ? (
                                    <button
                                        className='rounded-lg bg-contrast px-4 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60 cursor-pointer'
                                        type='button'
                                        onClick={downloadFile}
                                        disabled={downloadLoading}
                                    >
                                        {downloadLoading ? t('challenge.downloadPreparing') : t('challenge.download')}
                                    </button>
                                ) : null}
                            </div>
                            {!auth.user ? <p className='mt-2 text-xs text-warning'>{t('challenge.fileLoginRequired')}</p> : null}
                            {!auth.user ? (
                                <a className='mt-2 inline-block text-xs text-warning underline' href='/login' onClick={(e) => navigate('/login', e)}>
                                    {t('auth.loginLink')}
                                </a>
                            ) : null}
                            {downloadMessage ? <p className='mt-2 text-xs text-danger'>{downloadMessage}</p> : null}
                        </div>
                    </div>
                ) : null}

                <div className='mt-6 space-y-6'>
                    {stackEnabled ? (
                        <div className='rounded-xl border border-border bg-surface-muted p-4 text-sm text-text'>
                            <div className='flex flex-wrap items-center justify-between gap-3'>
                                <div>
                                    <p className='font-medium'>{t('challenge.stackInstance')}</p>
                                    <p className='text-xs text-text-subtle'>{stackPolling ? (stackNextInterval === 60000 ? t('challenge.stackRefreshing60') : t('challenge.stackRefreshing10')) : t('challenge.stackRefreshPaused')}</p>
                                </div>
                                {auth.user ? (
                                    <div className='flex flex-wrap items-center gap-2'>
                                        {stackInfo ? (
                                            <>
                                                <button
                                                    className='rounded-lg border border-border px-3 py-2 text-xs font-medium text-text transition hover:border-border hover:text-text disabled:opacity-60 cursor-pointer'
                                                    type='button'
                                                    onClick={loadStack}
                                                    disabled={stackLoading}
                                                >
                                                    {stackLoading ? t('common.loading') : t('common.refresh')}
                                                </button>
                                                <button
                                                    className='rounded-lg border border-danger/30 px-3 py-2 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60 cursor-pointer'
                                                    type='button'
                                                    onClick={deleteStack}
                                                    disabled={stackLoading}
                                                >
                                                    {stackLoading ? t('challenge.stackWorking') : t('challenge.deleteStack')}
                                                </button>
                                            </>
                                        ) : (
                                            <button
                                                className='rounded-lg bg-contrast px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60 cursor-pointer'
                                                type='button'
                                                onClick={createStack}
                                                disabled={stackLoading || isSolved}
                                            >
                                                {stackLoading ? t('auth.creating') : t('challenge.createStack')}
                                            </button>
                                        )}
                                    </div>
                                ) : null}
                            </div>

                            {!auth.user ? (
                                <div className='mt-2 text-xs text-warning'>
                                    <p>{t('challenge.stackLoginRequired')}</p>
                                    <a className='mt-1 inline-block underline' href='/login' onClick={(e) => navigate('/login', e)}>
                                        {t('auth.loginLink')}
                                    </a>
                                </div>
                            ) : isSolved ? (
                                <p className='mt-2 text-xs text-text-subtle'>{t('challenge.stackSolvedNoNew')}</p>
                            ) : stackInfo ? (
                                <div className='mt-3 grid gap-2 text-xs text-text-muted'>
                                    {(() => {
                                        const endpoints =
                                            stackInfo.node_public_ip && stackInfo.ports.length > 0
                                                ? stackInfo.ports.map((port, index) => {
                                                      const endpoint = `${port.protocol} ${port.container_port} / ${stackInfo.node_public_ip}:${port.node_port}`
                                                      const nc = `nc${port.protocol === 'UDP' ? ' -u' : ''} ${stackInfo.node_public_ip} ${port.node_port}`
                                                      return (
                                                          <div key={`${port.container_port}-${port.protocol}-${index}`} className='space-y-1'>
                                                              <div>
                                                                  <span className='font-medium text-text'>{t('challenge.stackEndpoint')}</span>
                                                                  <span className='ml-2'>
                                                                      {t('challenge.stackPortIndex', {
                                                                          index: index + 1,
                                                                      })}
                                                                      : {endpoint}
                                                                  </span>
                                                              </div>
                                                              <div>
                                                                  <span className='font-medium text-text'>{t('challenge.stackNCEndpoint')}</span>
                                                                  <span className='ml-2 font-mono'>{nc}</span>
                                                              </div>
                                                          </div>
                                                      )
                                                  })
                                                : t('challenge.stackPending')

                                        return (
                                            <>
                                                <div className='flex flex-wrap items-center gap-2'>
                                                    <span className='font-medium text-text'>{t('challenge.stackStatus')}</span>
                                                    <span className='rounded-full bg-surface-subtle px-2 py-0.5 text-[11px]'>{stackInfo.status}</span>
                                                </div>
                                                <div>
                                                    <span className='font-medium text-text'>{t('challenge.stackCreatedBy')}</span>
                                                    <span className='ml-2'>{stackInfo.created_by_username}</span>
                                                </div>
                                                <div>
                                                    <span className='font-medium text-text'>{t('challenge.stackPorts')}</span>
                                                    {typeof endpoints === 'string' ? <span className='ml-2'>{endpoints}</span> : <div className='mt-2 grid gap-2'>{endpoints}</div>}
                                                </div>
                                                <div>
                                                    <span className='font-medium text-text'>{t('challenge.stackTtl')}</span>
                                                    <span className='ml-2'>{formatTimestamp(stackInfo.ttl_expires_at)}</span>
                                                </div>
                                            </>
                                        )
                                    })()}
                                </div>
                            ) : (
                                <p className='mt-2 text-xs text-text-subtle'>{t('challenge.stackNoActive')}</p>
                            )}

                            {stackMessage ? <p className='mt-2 text-xs text-danger'>{stackMessage}</p> : null}
                        </div>
                    ) : null}
                    {isLocked ? null : !auth.user ? (
                        <div className='rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning-strong'>
                            {t('challenge.loginToSubmitPrefix')}{' '}
                            <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                                {t('auth.loginLink')}
                            </a>{' '}
                            {t('challenge.loginToSubmitSuffix')}
                        </div>
                    ) : isSolved ? (
                        <div className='rounded-xl border border-success/40 bg-success/10 p-4 text-sm text-success'>{t('challenge.correct')}</div>
                    ) : !isActive ? (
                        <div className='rounded-xl border border-border/40 bg-surface/10 p-4 text-sm text-text-muted'>{t('challenge.inactiveMessage')}</div>
                    ) : (
                        <form
                            className='space-y-4'
                            onSubmit={(event) => {
                                event.preventDefault()
                                submitFlag()
                            }}
                        >
                            <div className='flex flex-col gap-3 md:flex-row md:items-end'>
                                <label className='flex-1 text-sm font-medium text-text'>
                                    <span className='block mb-2'>{t('challenge.enterFlag')}</span>
                                    <input
                                        className='w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                        type='text'
                                        value={flagInput}
                                        onChange={(event) => setFlagInput(event.target.value)}
                                        placeholder={t('challenge.flagPlaceholder')}
                                        autoComplete='off'
                                    />
                                </label>
                                <button
                                    className='w-full md:w-auto rounded-xl bg-accent px-6 py-3 text-sm font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer'
                                    type='submit'
                                    disabled={submission.status === 'loading'}
                                >
                                    {submission.status === 'loading' ? t('challenge.submitting') : t('challenge.submit')}
                                </button>
                            </div>
                            {submission.message ? (
                                <div className={`rounded-xl border px-4 py-3 text-sm ${isSuccessful ? 'border-success/40 bg-success/10 text-success' : 'border-danger/40 bg-danger/10 text-danger'}`}>{submission.message}</div>
                            ) : null}
                        </form>
                    )}
                </div>
            </div>
        </div>
    )
}

export default ChallengeModal
