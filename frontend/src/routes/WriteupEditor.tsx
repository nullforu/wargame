import { useEffect, useMemo, useState } from 'react'
import MonacoEditor from '../components/MonacoEditor'
import UserAvatar from '../components/UserAvatar'
import ChallengeSummaryCard from './challenge-detail/ChallengeSummaryCard'
import { useApi } from '../lib/useApi'
import { useAuth } from '../lib/auth'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { navigate } from '../lib/router'
import type { Challenge, Writeup } from '../lib/types'
import { normalizeLevel } from '../lib/level'

interface RouteProps {
    routeParams?: Record<string, string>
}

const WriteupEditor = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()
    const challengeId = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])

    const [writeup, setWriteup] = useState<Writeup | null>(null)
    const [summaryChallenge, setSummaryChallenge] = useState<Challenge | null>(null)
    const [_canViewContent, setCanViewContent] = useState(false)
    const [loading, setLoading] = useState(true)
    const [errorMessage, setErrorMessage] = useState('')
    const [saving, setSaving] = useState(false)
    const [message, setMessage] = useState('')
    const [content, setContent] = useState('')

    const loadWriteup = async () => {
        if (!challengeId) return
        setLoading(true)
        setErrorMessage('')
        setMessage('')

        try {
            const [challengeData, myWriteupResult] = await Promise.all([api.challenge(challengeId), auth.user ? api.challengeMyWriteup(challengeId).catch(() => null) : Promise.resolve(null)])
            setSummaryChallenge(challengeData)
            setCanViewContent(Boolean(challengeData.is_solved))
            const myWriteup = myWriteupResult?.writeup ?? null
            setWriteup(myWriteup)
            setContent(myWriteup?.content ?? '')
        } catch (error) {
            setWriteup(null)
            setSummaryChallenge(null)
            setCanViewContent(false)
            setErrorMessage(formatApiError(error, t).message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadWriteup()
    }, [challengeId])

    const saveWriteup = async () => {
        if (!challengeId || !auth.user || saving) return
        setSaving(true)
        setMessage('')
        try {
            if (writeup) {
                const updated = await api.updateWriteup(writeup.id, { content })
                setWriteup(updated)
                setContent(updated.content ?? '')
            } else {
                const created = await api.createWriteup(challengeId, content)
                setWriteup(created)
                setContent(created.content ?? '')
            }
            setMessage(t('writeup.saved'))
        } catch (error) {
            setMessage(formatApiError(error, t).message)
        } finally {
            setSaving(false)
        }
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
            <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
                <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                    <div className='min-w-0 space-y-4'>
                        <div className='rounded-2xl sm:p-5'>
                            <div className='animate-pulse space-y-4'>
                                <div className='h-5 w-40 rounded bg-surface-muted' />
                                <div className='space-y-2'>
                                    <div className='h-4 w-full rounded bg-surface-muted' />
                                    <div className='h-4 w-11/12 rounded bg-surface-muted' />
                                    <div className='h-4 w-4/5 rounded bg-surface-muted' />
                                </div>
                                <div className='mt-8 h-10 w-full rounded bg-surface-muted' />
                                <div className='h-105 w-full rounded bg-surface-muted' />
                                <div className='h-10 w-40 rounded bg-surface-muted' />
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
                        </div>
                    </div>
                </div>
            </section>
        )
    }

    if (errorMessage || !summaryChallenge) {
        return (
            <section className='animate'>
                <div className='border border-danger/40 bg-danger/10 p-4 text-sm text-danger'>{errorMessage || t('errors.notFound')}</div>
            </section>
        )
    }

    const level = normalizeLevel(summaryChallenge.level)
    const levelLabel = level > 0 ? String(level) : t('level.unknown')
    const createdSummary = summaryChallenge.created_at ? formatDateTime(summaryChallenge.created_at, localeTag) : t('common.na')
    const authorName = auth.user?.username?.trim() || t('common.na')
    const authorAffiliation = auth.user?.affiliation?.trim() ?? ''
    const authorBio = auth.user?.bio?.trim() ?? ''

    return (
        <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <div className='min-w-0 space-y-4'>
                    <header className='min-w-0 rounded-2xl py-1 sm:py-3'>
                        <button className='inline-flex items-center gap-2 rounded-md px-1 py-1 text-sm text-text-muted hover:text-text' onClick={() => navigate(`/challenges/${challengeId}`)}>
                            <svg xmlns='http://www.w3.org/2000/svg' className='h-5 w-5' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                                <path strokeLinecap='round' strokeLinejoin='round' d='M15 19l-7-7 7-7' />
                            </svg>
                            {t('writeup.backToChallenge')}
                        </button>
                    </header>

                    <div className='space-y-8 lg:hidden'>
                        <ChallengeSummaryCard challenge={summaryChallenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} />

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('writeup.authorTitle')}</h2>

                            <div className='rounded-2xl bg-surface/70'>
                                <div className='flex items-start justify-between gap-4 py-2'>
                                    <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                        <UserAvatar username={authorName} size='md' />
                                        <div className='min-w-0'>
                                            <button className='block max-w-full truncate text-left text-base font-semibold text-text hover:text-accent' onClick={() => navigate(`/profile`)}>
                                                {authorName}
                                            </button>
                                            <p className='mt-1 text-sm text-text-subtle'>{authorAffiliation || t('common.na')}</p>
                                            <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{authorBio || t('profile.noBio')}</p>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </section>
                    </div>

                    {!auth.user ? <div className='rounded-xl bg-warning/10 p-4 text-sm text-warning'>{t('writeup.loginRequired')}</div> : null}
                    {auth.user && !summaryChallenge.is_solved ? <div className='rounded-xl bg-warning/10 p-4 text-sm text-warning'>{t('writeup.hiddenUntilSolved')}</div> : null}

                    {auth.user && summaryChallenge.is_solved ? (
                        <div className='min-w-0 border-t border-border/70 pt-8 sm:pt-10'>
                            <div className='rounded-2xl bg-surface/70 p-4 sm:p-5'>
                                <MonacoEditor value={content} onChange={setContent} language='markdown' height='420px' />
                            </div>

                            <div className='mt-4 flex flex-wrap gap-2'>
                                <button className='rounded-md bg-accent px-4 py-2 text-sm text-white hover:bg-accent-strong disabled:opacity-60' onClick={() => void saveWriteup()} disabled={saving}>
                                    {saving ? t('writeup.saving') : t('common.save')}
                                </button>
                                {writeup ? (
                                    <button className='rounded-md border border-border bg-surface-muted px-4 py-2 text-sm text-text hover:bg-surface-subtle' onClick={() => navigate(`/writeups/${writeup.id}`)}>
                                        {t('common.view')}
                                    </button>
                                ) : null}
                            </div>

                            {message ? <p className='mt-2 text-xs text-text-muted'>{message}</p> : null}
                        </div>
                    ) : null}
                </div>

                <aside className='hidden lg:block lg:sticky'>
                    <div className='space-y-8'>
                        <ChallengeSummaryCard challenge={summaryChallenge} levelLabel={levelLabel} createdSummary={createdSummary} t={t} />

                        <section className='space-y-3 px-1'>
                            <h2 className='text-xl font-semibold text-text'>{t('writeup.authorTitle')}</h2>

                            <div className='rounded-2xl bg-surface/70'>
                                <div className='flex items-start justify-between gap-4 py-2'>
                                    <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                        <UserAvatar username={authorName} size='md' />
                                        <div className='min-w-0'>
                                            <button className='block max-w-full truncate text-left text-base font-semibold text-text hover:text-accent' onClick={() => navigate(`/profile/${authorName}`)}>
                                                {authorName}
                                            </button>
                                            <p className='mt-1 text-sm text-text-subtle'>{authorAffiliation || t('common.na')}</p>
                                            <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{authorBio || t('profile.noBio')}</p>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </section>
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default WriteupEditor
