import { useMemo, useState } from 'react'
import MonacoEditor from '../components/MonacoEditor'
import UserAvatar from '../components/UserAvatar'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'
import { formatApiError } from '../lib/utils'

const categoryTextKey = (category: number) => {
    switch (category) {
        case 0:
            return 'community.category.notice'
        case 1:
            return 'community.category.free'
        case 2:
            return 'community.category.qna'
        case 3:
            return 'community.category.humor'
        default:
            return 'common.na'
    }
}

interface RouteProps {
    routeParams?: Record<string, string>
}

const CommunityEditor = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const { state: auth } = useAuth()

    const listQuery = useMemo(() => {
        if (typeof window === 'undefined') return ''
        return window.location.search
    }, [])

    const [category, setCategory] = useState(1)
    const [title, setTitle] = useState('')
    const [content, setContent] = useState('')
    const [creating, setCreating] = useState(false)
    const [message, setMessage] = useState('')

    const submit = async () => {
        if (!auth.user || creating) return
        setCreating(true)
        setMessage('')
        try {
            const created = await api.createCommunityPost({ category, title, content })
            navigate(`/community/${created.id}${listQuery}`)
        } catch (e) {
            setMessage(formatApiError(e, t).message)
        } finally {
            setCreating(false)
        }
    }

    if (!auth.user) {
        return (
            <section className='animate'>
                <div className='rounded-xl bg-warning/10 p-4 text-sm text-warning'>{t('writeup.loginRequired')}</div>
            </section>
        )
    }

    const authorName = auth.user.username?.trim() || t('common.na')
    const authorAffiliation = auth.user.affiliation?.trim() ?? ''
    const authorBio = auth.user.bio?.trim() ?? ''
    const isAdmin = auth.user.role === 'admin'

    return (
        <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <div className='min-w-0 space-y-4'>
                    <header className='min-w-0 rounded-2xl py-1 sm:py-3'>
                        <button className='inline-flex items-center gap-2 rounded-md px-1 py-1 text-sm text-text-muted hover:text-text' onClick={() => navigate(`/community${listQuery}`)}>
                            <svg xmlns='http://www.w3.org/2000/svg' className='h-5 w-5' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                                <path strokeLinecap='round' strokeLinejoin='round' d='M15 19l-7-7 7-7' />
                            </svg>
                            {t('community.back')}
                        </button>
                    </header>

                    <div className='space-y-3 border-t border-border/70 pt-6'>
                        <div className='flex flex-wrap gap-2'>
                            {[0, 1, 2, 3]
                                .filter((c) => (isAdmin ? true : c !== 0))
                                .map((c) => (
                                    <button
                                        key={c}
                                        type='button'
                                        className={`rounded-md border px-3 py-1 text-xs ${category === c ? 'border-accent/60 bg-accent/12 text-accent' : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                        onClick={() => setCategory(c)}
                                    >
                                        {t(categoryTextKey(c))}
                                    </button>
                                ))}
                        </div>

                        <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder={t('community.titleInput')} className='w-full rounded-md border border-border/70 bg-surface px-3 py-2 text-sm text-text' />

                        <div className='rounded-2xl bg-surface/70 p-4 sm:p-5'>
                            <MonacoEditor value={content} onChange={setContent} language='markdown' height='420px' />
                        </div>

                        <div className='flex flex-wrap gap-2'>
                            <button className='rounded-md bg-accent px-4 py-2 text-sm text-white hover:bg-accent-strong disabled:opacity-60' onClick={() => void submit()} disabled={creating}>
                                {creating ? t('community.creating') : t('community.create')}
                            </button>
                            <button className='rounded-md border border-border bg-surface-muted px-4 py-2 text-sm text-text hover:bg-surface-subtle' onClick={() => navigate(`/community${listQuery}`)}>
                                {t('common.cancel')}
                            </button>
                        </div>
                        {message ? <p className='text-xs text-danger'>{message}</p> : null}
                    </div>
                </div>

                <aside className='space-y-3'>
                    <section className='space-y-3 px-1'>
                        <h2 className='text-xl font-semibold text-text'>{t('community.authorTitle')}</h2>

                        <div className='rounded-2xl bg-surface/70'>
                            <div className='flex items-start justify-between gap-4 py-2'>
                                <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                    <UserAvatar username={authorName} size='md' />
                                    <div className='min-w-0'>
                                        <button className='block max-w-full truncate text-left text-base font-semibold text-text hover:text-accent' onClick={() => navigate('/profile')}>
                                            {authorName}
                                        </button>
                                        <p className='mt-1 text-sm text-text-subtle'>{authorAffiliation || t('common.na')}</p>
                                        <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{authorBio || t('profile.noBio')}</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </section>
                </aside>
            </div>
        </section>
    )
}

export default CommunityEditor
