import { useEffect, useMemo, useState } from 'react'
import Markdown from '../components/Markdown'
import UserAvatar from '../components/UserAvatar'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import type { Challenge, Writeup } from '../lib/types'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { normalizeLevel } from '../lib/level'
import ChallengeSummaryCard from './challenge-detail/ChallengeSummaryCard'

interface RouteProps {
    routeParams?: Record<string, string>
}

const WriteupDetail = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()
    const writeupID = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])

    const [writeup, setWriteup] = useState<Writeup | null>(null)
    const [summaryChallenge, setSummaryChallenge] = useState<Challenge | null>(null)
    const [canViewContent, setCanViewContent] = useState(false)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')

    const [deleting, setDeleting] = useState(false)
    const [message, setMessage] = useState('')

    const loadWriteup = async () => {
        if (!writeupID) return
        setLoading(true)
        setErrorMessage('')

        try {
            const data = await api.writeup(writeupID)
            const challengeData = await api.challenge(data.writeup.challenge.id)
            setWriteup(data.writeup)
            setSummaryChallenge(challengeData)
            setCanViewContent(data.can_view_content)
            if (!data.can_view_content) {
                navigate(`/challenges/${data.writeup.challenge.id}`)
                return
            }
        } catch (error) {
            setWriteup(null)
            setSummaryChallenge(null)
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadWriteup()
    }, [writeupID])

    const deleteWriteup = async () => {
        if (!writeup || deleting) return
        setDeleting(true)
        setMessage('')
        try {
            await api.deleteWriteup(writeup.id)
            navigate(`/challenges/${writeup.challenge.id}`)
        } catch (error) {
            setMessage(formatApiError(error, t).message)
        } finally {
            setDeleting(false)
        }
    }

    if (!writeupID) {
        return (
            <section className='animate'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{t('errors.invalid')}</div>
            </section>
        )
    }

    if (loading) {
        return (
            <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
                <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                    <div className='min-w-0 space-y-2'>
                        <div className='rounded-2xl sm:p-5'>
                            <div className='animate-pulse space-y-4'>
                                <div className='h-5 w-40 rounded bg-surface-muted' />
                                <div className='space-y-2'>
                                    <div className='h-4 w-full rounded bg-surface-muted' />
                                    <div className='h-4 w-11/12 rounded bg-surface-muted' />
                                    <div className='h-4 w-4/5 rounded bg-surface-muted' />
                                </div>
                                <div className='mt-8 h-10 w-full rounded bg-surface-muted' />
                            </div>
                        </div>
                    </div>

                    <div className='hidden lg:block'>
                        <div className='space-y-8'>
                            <div className='rounded-2xl border border-border/20 bg-surface p-5 shadow-sm animate-pulse space-y-3'>
                                <div className='h-6 w-16 rounded bg-surface-muted' />
                                <div className='h-9 w-full rounded bg-surface-muted' />
                                <div className='h-5 w-1/3 rounded bg-surface-muted' />
                                <div className='h-5 w-4/5 rounded bg-surface-muted' />
                            </div>

                            <section className='space-y-3 px-1'>
                                <div className='h-7 w-32 rounded bg-surface-muted animate-pulse' />
                                <div className='rounded-2xl bg-surface/70 p-2 animate-pulse'>
                                    <div className='flex items-start gap-3.75'>
                                        <div className='h-10 w-10 rounded-full bg-surface-muted' />
                                        <div className='min-w-0 flex-1 space-y-2'>
                                            <div className='h-5 w-1/2 rounded bg-surface-muted' />
                                            <div className='h-4 w-1/3 rounded bg-surface-muted' />
                                            <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                        </div>
                                    </div>
                                </div>
                            </section>
                        </div>
                    </div>
                </div>
            </section>
        )
    }

    if (!writeup || errorMessage) {
        return (
            <section className='animate'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{errorMessage || t('errors.notFound')}</div>
            </section>
        )
    }

    const isMine = Boolean(auth.user && auth.user.id === writeup.author.user_id)
    const createdAtLabel = formatDateTime(writeup.created_at, localeTag)
    const authorName = writeup.author.username.trim()
    const authorAffiliation = writeup.author.affiliation?.trim() ?? ''
    const authorBio = writeup.author.bio?.trim() ?? ''
    const formatCompactDateTime = (value: string) => {
        const date = new Date(value)
        if (Number.isNaN(date.getTime())) return t('common.na')
        const yyyy = date.getFullYear()
        const mm = String(date.getMonth() + 1).padStart(2, '0')
        const dd = String(date.getDate()).padStart(2, '0')
        const hh = String(date.getHours()).padStart(2, '0')
        const min = String(date.getMinutes()).padStart(2, '0')
        return `${yyyy}-${mm}-${dd} ${hh}:${min}`
    }
    const currentLevel = normalizeLevel(summaryChallenge?.level)
    const levelLabel = currentLevel > 0 ? String(currentLevel) : t('level.unknown')
    const createdSummary = summaryChallenge?.created_at ? formatCompactDateTime(summaryChallenge.created_at) : t('common.na')

    return (
        <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <div className='min-w-0 space-y-4'>
                    <header className='min-w-0 rounded-2xl py-1 sm:py-3'>
                        <button className='inline-flex items-center gap-2 rounded-md px-1 py-1 text-sm text-text-muted hover:text-text' onClick={() => navigate(`/challenges/${writeup.challenge.id}`)}>
                            <svg xmlns='http://www.w3.org/2000/svg' className='h-5 w-5' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                                <path strokeLinecap='round' strokeLinejoin='round' d='M15 19l-7-7 7-7' />
                            </svg>
                            {t('writeup.backToChallenge')}
                        </button>
                    </header>

                    <div className='space-y-8 lg:hidden'>
                        {summaryChallenge ? <ChallengeSummaryCard challenge={summaryChallenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} /> : null}

                        {!canViewContent ? <div className='rounded-xl bg-warning/10 p-4 text-sm text-warning'>{t('writeup.hiddenUntilSolved')}</div> : null}

                        {canViewContent ? (
                            <article className='min-w-0'>
                                <Markdown content={writeup.content ?? ''} className='text-sm leading-7 text-text' />
                                <p className='mt-8 text-xs text-text-muted/85 sm:text-sm'>{createdAtLabel}</p>
                            </article>
                        ) : null}

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('writeup.authorTitle')}</h2>

                            <div className='rounded-2xl bg-surface/70'>
                                <div className='flex items-start justify-between gap-4 py-2'>
                                    <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                        <UserAvatar username={authorName} size='md' />
                                        <div className='min-w-0'>
                                            <button className='block max-w-full truncate text-left text-base font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${writeup.author.user_id}`)}>
                                                {authorName}
                                            </button>
                                            {authorAffiliation ? <p className='mt-1 text-sm text-text-subtle'>{authorAffiliation}</p> : null}
                                            <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{authorBio || t('profile.noBio')}</p>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {isMine ? (
                                <div className='flex flex-col items-end gap-2'>
                                    <div className='flex flex-wrap gap-2'>
                                        <button className='rounded-md border border-border bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle' onClick={() => navigate(`/challenges/${writeup.challenge.id}/writeup`)}>
                                            {t('common.edit')}
                                        </button>
                                        <button className='rounded-md border border-danger/30 bg-danger/10 px-3 py-1.5 text-xs text-danger hover:bg-danger/15 disabled:opacity-60' onClick={() => void deleteWriteup()} disabled={deleting}>
                                            {deleting ? t('writeup.deleting') : t('common.delete')}
                                        </button>
                                    </div>
                                    {message ? <p className='mt-2 text-xs text-text-muted'>{message}</p> : null}
                                </div>
                            ) : null}
                        </section>
                    </div>

                    {!canViewContent ? <div className='rounded-xl bg-warning/10 p-4 text-sm text-warning'>{t('writeup.hiddenUntilSolved')}</div> : null}

                    {canViewContent ? (
                        <article className='min-w-0 border-t border-border/70 pt-8 sm:pt-10'>
                            <Markdown content={writeup.content ?? ''} className='text-sm leading-7 text-text' />
                            <p className='mt-8 text-xs text-text-muted/85 sm:text-sm'>{createdAtLabel}</p>
                        </article>
                    ) : null}
                </div>

                <aside className='hidden lg:block lg:sticky'>
                    <div className='space-y-8'>
                        {summaryChallenge ? <ChallengeSummaryCard challenge={summaryChallenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} /> : null}

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('writeup.authorTitle')}</h2>

                            <div className='rounded-2xl bg-surface/70'>
                                <div className='flex items-start justify-between gap-4 py-2'>
                                    <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                        <UserAvatar username={authorName} size='md' />
                                        <div className='min-w-0'>
                                            <button className='block max-w-full truncate text-left text-base font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${writeup.author.user_id}`)}>
                                                {authorName}
                                            </button>
                                            <p className='mt-1 text-sm text-text-subtle'>{authorAffiliation || t('common.na')}</p>
                                            <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{authorBio || t('profile.noBio')}</p>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {isMine ? (
                                <div className='flex flex-col items-end gap-2'>
                                    <div className='flex flex-wrap gap-2'>
                                        <button className='rounded-md border border-border bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle' onClick={() => navigate(`/challenges/${writeup.challenge.id}/writeup`)}>
                                            {t('common.edit')}
                                        </button>
                                        <button className='rounded-md border border-danger/30 bg-danger/10 px-3 py-1.5 text-xs text-danger hover:bg-danger/15 disabled:opacity-60' onClick={() => void deleteWriteup()} disabled={deleting}>
                                            {deleting ? t('writeup.deleting') : t('common.delete')}
                                        </button>
                                    </div>
                                    {message ? <p className='mt-2 text-xs text-text-muted'>{message}</p> : null}
                                </div>
                            ) : null}
                        </section>
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default WriteupDetail
