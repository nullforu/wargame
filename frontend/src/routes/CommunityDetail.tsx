import { useEffect, useMemo, useState } from 'react'
import Markdown from '../components/Markdown'
import UserAvatar from '../components/UserAvatar'
import { useApi } from '../lib/useApi'
import { useAuth } from '../lib/auth'
import { getLocaleTag, useLocale, useT } from '../lib/i18n'
import { navigate } from '../lib/router'
import type { CommunityComment, CommunityPost, CommunityPostLike, PaginationMeta } from '../lib/types'
import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
import { categoryTextKey, categoryBadgeClass } from './Community'

interface RouteProps {
    routeParams?: Record<string, string>
}
const EMPTY_LIKE_PAGINATION: PaginationMeta = { page: 1, page_size: 20, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const EMPTY_COMMENT_PAGINATION: PaginationMeta = { page: 1, page_size: 10, total_count: 0, total_pages: 0, has_prev: false, has_next: false }
const COMMENT_PAGE_SIZE = 10

const CommunityDetail = ({ routeParams = {} }: RouteProps) => {
    const t = useT()
    const locale = useLocale()
    const localeTag = useMemo(() => getLocaleTag(locale), [locale])
    const api = useApi()
    const { state: auth } = useAuth()
    const postID = useMemo(() => parseRouteId(routeParams.id), [routeParams.id])

    const [post, setPost] = useState<CommunityPost | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [editing, setEditing] = useState(false)
    const [title, setTitle] = useState('')
    const [content, setContent] = useState('')
    const [category, setCategory] = useState(1)
    const [saving, setSaving] = useState(false)
    const [likes, setLikes] = useState<CommunityPostLike[]>([])
    const [likePage, setLikePage] = useState(1)
    const [likePagination, setLikePagination] = useState<PaginationMeta>(EMPTY_LIKE_PAGINATION)
    const [likeSubmitting, setLikeSubmitting] = useState(false)
    const [comments, setComments] = useState<CommunityComment[]>([])
    const [commentPage, setCommentPage] = useState(1)
    const [commentPagination, setCommentPagination] = useState<PaginationMeta>(EMPTY_COMMENT_PAGINATION)
    const [commentInput, setCommentInput] = useState('')
    const [commentInputExpanded, setCommentInputExpanded] = useState(false)
    const [commentLoading, setCommentLoading] = useState(true)
    const [commentSubmitting, setCommentSubmitting] = useState(false)
    const [editingCommentID, setEditingCommentID] = useState<number | null>(null)
    const [editingCommentContent, setEditingCommentContent] = useState('')

    const listQuery = useMemo(() => {
        if (typeof window === 'undefined') return ''
        return window.location.search
    }, [])

    const load = async () => {
        if (!postID) return
        setLoading(true)
        setError('')
        try {
            const data = await api.communityPost(postID)
            setPost(data.post)
            setTitle(data.post.title)
            setContent(data.post.content)
            setCategory(data.post.category)
        } catch (e) {
            setError(formatApiError(e, t).message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void load()
    }, [postID])

    useEffect(() => {
        if (!postID) return
        const loadLikes = async () => {
            try {
                const data = await api.communityPostLikes(postID, likePage, 20)
                setLikes(data.likes)
                setLikePagination(data.pagination)
            } catch {
                setLikes([])
                setLikePagination(EMPTY_LIKE_PAGINATION)
            }
        }
        void loadLikes()
    }, [api, postID, likePage])

    useEffect(() => {
        if (!postID) return
        const loadComments = async () => {
            setCommentLoading(true)
            try {
                const data = await api.communityComments(postID, commentPage, COMMENT_PAGE_SIZE)
                setComments(data.comments)
                setCommentPagination(data.pagination)
            } catch {
                setComments([])
                setCommentPagination(EMPTY_COMMENT_PAGINATION)
            } finally {
                setCommentLoading(false)
            }
        }
        void loadComments()
    }, [api, postID, commentPage])

    if (!postID) return <p className='text-sm text-danger'>{t('errors.invalid')}</p>
    if (loading) return <p className='text-sm text-text-muted'>{t('common.loading')}</p>
    if (!post || error) return <p className='text-sm text-danger'>{error || t('errors.notFound')}</p>

    const isAdmin = auth.user?.role === 'admin'
    const isOwner = auth.user?.id === post.author.user_id
    const canEdit = isAdmin || (isOwner && post.category !== 0)

    const save = async () => {
        setSaving(true)
        try {
            const updated = await api.updateCommunityPost(post.id, { category, title, content })
            setPost(updated)
            setEditing(false)
        } catch (e) {
            setError(formatApiError(e, t).message)
        } finally {
            setSaving(false)
        }
    }

    const remove = async () => {
        try {
            await api.deleteCommunityPost(post.id)
            navigate(`/community${listQuery}`)
        } catch (e) {
            setError(formatApiError(e, t).message)
        }
    }

    const toggleLike = async () => {
        if (!auth.user || likeSubmitting) return
        setLikeSubmitting(true)
        try {
            const res = await api.toggleCommunityPostLike(post.id)
            setPost((prev) => (prev ? { ...prev, liked_by_me: res.liked, like_count: res.like_count } : prev))
            const data = await api.communityPostLikes(post.id, likePage, 20)
            setLikes(data.likes)
            setLikePagination(data.pagination)
        } catch (e) {
            setError(formatApiError(e, t).message)
        } finally {
            setLikeSubmitting(false)
        }
    }

    const startEdit = () => {
        setTitle(post.title)
        setContent(post.content)
        setCategory(post.category)
        setEditing(true)
    }

    const cancelEdit = () => {
        setTitle(post.title)
        setContent(post.content)
        setCategory(post.category)
        setEditing(false)
    }

    return (
        <section className='animate space-y-4 px-0 sm:px-1 md:px-2 lg:px-0'>
            <button className='inline-flex items-center gap-2 rounded-md px-1 py-1 text-sm text-text-muted hover:text-text' onClick={() => navigate(`/community${listQuery}`)}>
                <svg xmlns='http://www.w3.org/2000/svg' className='h-5 w-5' fill='none' viewBox='0 0 24 24' stroke='currentColor' strokeWidth={2}>
                    <path strokeLinecap='round' strokeLinejoin='round' d='M15 19l-7-7 7-7' />
                </svg>
                {t('community.back')}
            </button>

            <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(320px,0.92fr)]'>
                <article className='min-w-0 space-y-4'>
                    <header className='border-b border-border/70 pb-4'>
                        <div className='flex flex-wrap items-center gap-2'>
                            <span className={`inline-flex rounded-md border px-2 py-0.5 text-[11px] font-medium ${categoryBadgeClass(post.category)}`}>{t(categoryTextKey(post.category))}</span>
                            <span className='text-xs text-text-subtle'>#{post.id}</span>
                            <span className='text-xs text-text-subtle'>·</span>
                            <span className='text-xs text-text-subtle'>{t('community.views', { count: post.view_count })}</span>
                            <span className='text-xs text-text-subtle'>·</span>
                            <span className='text-xs text-text-subtle'>{t('community.likes', { count: post.like_count })}</span>
                            <span className='text-xs text-text-subtle'>·</span>
                            <span className='text-xs text-text-subtle'>{t('community.comments', { count: post.comment_count })}</span>
                        </div>
                        {editing ? (
                            <div className='mt-3 space-y-2'>
                                <div className='flex flex-wrap items-center gap-2'>
                                    {[0, 1, 2, 3]
                                        .filter((c) => (isAdmin ? true : c !== 0))
                                        .map((c) => (
                                            <button
                                                key={c}
                                                type='button'
                                                className={`rounded-md border px-2 py-1 text-xs ${category === c ? categoryBadgeClass(c) : 'border-border/60 bg-surface-muted text-text-muted'}`}
                                                onClick={() => setCategory(c)}
                                            >
                                                {t(categoryTextKey(c))}
                                            </button>
                                        ))}
                                </div>
                                <input value={title} onChange={(e) => setTitle(e.target.value)} className='w-full rounded-md border border-border/70 bg-surface px-3 py-2 text-base font-semibold text-text' />
                            </div>
                        ) : (
                            <h1 className='mt-3 text-2xl font-semibold text-text'>{post.title}</h1>
                        )}
                        <p className='mt-2 text-xs text-text-muted'>{formatDateTime(post.created_at, localeTag)}</p>
                    </header>

                    <section className='min-w-0'>
                        {editing ? (
                            <textarea value={content} onChange={(e) => setContent(e.target.value)} className='min-h-90 w-full rounded-md border border-border/70 bg-surface px-3 py-2 text-sm text-text' />
                        ) : (
                            <Markdown content={post.content} className='text-sm text-text' />
                        )}
                        {post.updated_at !== post.created_at ? (
                            <p className='mt-6 text-xs text-text-muted'>
                                {t('common.updatedAt')}: {formatDateTime(post.updated_at, localeTag)}
                            </p>
                        ) : null}
                    </section>

                    <section className='space-y-3 px-1 lg:hidden'>
                        <h2 className='text-xl font-semibold text-text'>{t('community.authorTitle')}</h2>

                        <div className='rounded-2xl bg-surface/70'>
                            <div className='flex items-start justify-between gap-4 py-2'>
                                <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                    <UserAvatar username={post.author.username} size='md' />
                                    <div className='min-w-0'>
                                        <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${post.author.user_id}`)}>
                                            {post.author.username}
                                        </button>
                                        {post.author.affiliation ? <p className='mt-1 text-sm text-text-subtle'>{post.author.affiliation.trim()}</p> : null}
                                        <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{post.author.bio?.trim() || t('profile.noBio')}</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </section>

                    <section className='flex flex-col items-center gap-2 pt-5'>
                        <button
                            type='button'
                            disabled={!auth.user || likeSubmitting}
                            onClick={() => void toggleLike()}
                            className={`inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-50 ${
                                post.liked_by_me ? 'bg-danger/12 text-danger hover:bg-danger/20' : 'bg-surface-muted text-text hover:bg-surface-subtle'
                            }`}
                        >
                            {post.liked_by_me ? t('community.unlike') : t('community.like')}
                            <span className='text-sm font-semibold text-text'>{post.like_count}</span>
                        </button>
                        {!auth.user ? <p className='text-xs text-text-subtle'>{t('community.likeLoginRequired')}</p> : null}
                    </section>

                    {canEdit ? (
                        <div className='flex justify-end mt-4'>
                            <div className='flex gap-2'>
                                {editing ? (
                                    <>
                                        <button disabled={saving} onClick={() => void save()} className='rounded-md bg-accent px-3 py-1.5 text-xs text-white disabled:opacity-60'>
                                            {saving ? t('common.loading') : t('common.save')}
                                        </button>
                                        <button onClick={cancelEdit} className='rounded-md border border-border bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle'>
                                            {t('common.cancel')}
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <button onClick={startEdit} className='rounded-md border border-border bg-surface-muted px-3 py-1.5 text-xs text-text hover:bg-surface-subtle'>
                                            {t('common.edit')}
                                        </button>
                                        <button onClick={() => void remove()} className='rounded-md border border-danger/30 bg-danger/10 px-3 py-1.5 text-xs text-danger hover:bg-danger/15'>
                                            {t('common.delete')}
                                        </button>
                                    </>
                                )}
                            </div>
                        </div>
                    ) : null}

                    <section className='space-y-3 pt-4 lg:pt-10'>
                        <h2 className='text-xl font-semibold text-text'>
                            {t('community.commentsTitle')} <span className='text-accent'>{post.comment_count}</span>
                        </h2>

                        <div className='rounded-lg border border-border/70 bg-surface'>
                            <textarea
                                className={`w-full resize-none border-0 bg-transparent p-3 text-sm text-text outline-none placeholder:text-text-subtle/80 ${commentInputExpanded ? 'h-24 rounded-t-lg' : 'h-12 rounded-lg'}`}
                                placeholder={t('community.commentPlaceholder')}
                                maxLength={500}
                                value={commentInput}
                                id='comment-input'
                                onChange={(e) => setCommentInput(e.target.value)}
                                onFocus={() => setCommentInputExpanded(true)}
                                onClick={() => setCommentInputExpanded(true)}
                                disabled={!auth.user || commentSubmitting}
                            />
                            {commentInputExpanded ? (
                                <div className='flex items-center justify-between border-t border-border/70 px-3 py-2'>
                                    <span className='text-xs text-text-subtle'>{commentInput.length}/500</span>
                                    <div className='flex items-center gap-2'>
                                        <button
                                            className='rounded border border-border px-2 py-1 text-xs'
                                            onClick={() => {
                                                setCommentInput('')
                                                setCommentInputExpanded(false)
                                            }}
                                            disabled={commentSubmitting}
                                        >
                                            {t('common.cancel')}
                                        </button>
                                        <button
                                            className='rounded border border-border px-2 py-1 text-xs disabled:opacity-40'
                                            disabled={!auth.user || commentSubmitting || commentInput.trim().length === 0}
                                            onClick={async () => {
                                                if (!postID) return
                                                setCommentSubmitting(true)
                                                try {
                                                    await api.createCommunityComment(postID, commentInput)
                                                    setCommentInput('')
                                                    setCommentInputExpanded(false)
                                                    const data = await api.communityComments(postID, 1, COMMENT_PAGE_SIZE)
                                                    setCommentPage(1)
                                                    setComments(data.comments)
                                                    setCommentPagination(data.pagination)
                                                    setPost((prev) => (prev ? { ...prev, comment_count: data.pagination.total_count } : prev))
                                                } catch (e) {
                                                    setError(formatApiError(e, t).message)
                                                } finally {
                                                    setCommentSubmitting(false)
                                                }
                                            }}
                                        >
                                            {t('community.commentSubmit')}
                                        </button>
                                    </div>
                                </div>
                            ) : null}
                        </div>
                        {!auth.user ? <p className='text-xs text-warning'>{t('community.commentLoginRequired')}</p> : null}

                        <div className='space-y-3'>
                            {commentLoading ? (
                                <div className='space-y-3'>
                                    {[0, 1].map((idx) => (
                                        <div key={idx} className='rounded-lg bg-surface/50 animate-pulse'>
                                            <div className='flex items-center justify-between'>
                                                <div className='h-4 w-24 rounded bg-surface-muted' />
                                                <div className='h-3 w-28 rounded bg-surface-muted' />
                                            </div>
                                            <div className='mt-3 h-3 w-full rounded bg-surface-muted' />
                                        </div>
                                    ))}
                                </div>
                            ) : null}
                            {!commentLoading &&
                                comments.map((item) => {
                                    const isMine = auth.user?.id === item.author.user_id
                                    const isEditing = editingCommentID === item.id
                                    return (
                                        <div key={item.id} className='rounded-lg bg-surface/50 px-2 py-3'>
                                            <div className='flex items-start justify-between gap-2'>
                                                <div className='flex min-w-0 items-center gap-2'>
                                                    <UserAvatar username={item.author.username} size='sm' />
                                                    <span className='truncate text-sm font-semibold text-text'>{item.author.username}</span>
                                                </div>
                                                <span className='shrink-0 text-xs text-text-subtle'>{formatDateTime(item.created_at, localeTag)}</span>
                                            </div>
                                            {isEditing ? (
                                                <div className='mt-2 space-y-2'>
                                                    <textarea
                                                        className='w-full rounded border border-border bg-surface px-2 py-1 text-sm'
                                                        value={editingCommentContent}
                                                        onChange={(e) => setEditingCommentContent(e.target.value)}
                                                        maxLength={500}
                                                    />
                                                    <div className='flex gap-2'>
                                                        <button className='rounded border border-border px-2 py-1 text-xs' onClick={() => setEditingCommentID(null)}>
                                                            {t('common.cancel')}
                                                        </button>
                                                        <button
                                                            className='rounded bg-accent px-2 py-1 text-xs text-white'
                                                            onClick={async () => {
                                                                setCommentSubmitting(true)
                                                                try {
                                                                    await api.updateCommunityComment(item.id, { content: editingCommentContent })
                                                                    setEditingCommentID(null)
                                                                    const data = await api.communityComments(post.id, commentPage, COMMENT_PAGE_SIZE)
                                                                    setComments(data.comments)
                                                                } catch (e) {
                                                                    setError(formatApiError(e, t).message)
                                                                } finally {
                                                                    setCommentSubmitting(false)
                                                                }
                                                            }}
                                                        >
                                                            {t('common.save')}
                                                        </button>
                                                    </div>
                                                </div>
                                            ) : (
                                                <p className='mt-2 whitespace-pre-wrap wrap-break-word text-sm text-text'>{item.content}</p>
                                            )}
                                            {isMine && !isEditing ? (
                                                <div className='mt-2 flex gap-3 text-xs'>
                                                    <button
                                                        className='text-accent'
                                                        onClick={() => {
                                                            setEditingCommentID(item.id)
                                                            setEditingCommentContent(item.content)
                                                        }}
                                                    >
                                                        {t('common.edit')}
                                                    </button>
                                                    <button
                                                        className='text-danger'
                                                        onClick={async () => {
                                                            setCommentSubmitting(true)
                                                            try {
                                                                await api.deleteCommunityComment(item.id)
                                                                const targetPage = comments.length === 1 && commentPage > 1 ? commentPage - 1 : commentPage
                                                                const data = await api.communityComments(post.id, targetPage, COMMENT_PAGE_SIZE)
                                                                setCommentPage(targetPage)
                                                                setComments(data.comments)
                                                                setCommentPagination(data.pagination)
                                                                setPost((prev) => (prev ? { ...prev, comment_count: data.pagination.total_count } : prev))
                                                            } catch (e) {
                                                                setError(formatApiError(e, t).message)
                                                            } finally {
                                                                setCommentSubmitting(false)
                                                            }
                                                        }}
                                                    >
                                                        {t('common.delete')}
                                                    </button>
                                                </div>
                                            ) : null}
                                        </div>
                                    )
                                })}
                            {!commentLoading && comments.length === 0 ? <p className='text-sm text-text-muted'>{t('community.commentEmpty')}</p> : null}
                        </div>

                        <div className='flex items-center justify-between pt-1 text-sm text-text-muted'>
                            <span>
                                {commentPagination.page} / {commentPagination.total_pages || 1}
                            </span>
                            <div className='flex gap-2'>
                                <button
                                    className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                    disabled={!commentPagination.has_prev || commentLoading}
                                    onClick={() => setCommentPage((prev) => Math.max(1, prev - 1))}
                                >
                                    {t('common.previous')}
                                </button>
                                <button
                                    className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                                    disabled={!commentPagination.has_next || commentLoading}
                                    onClick={() => setCommentPage((prev) => prev + 1)}
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        </div>
                    </section>

                    {error ? <p className='text-xs text-danger'>{error}</p> : null}
                </article>

                <aside className='space-y-6'>
                    <section className='space-y-3 px-1'>
                        <h2 className='hidden lg:block text-xl font-semibold text-text'>{t('community.authorTitle')}</h2>

                        <div className='hidden lg:block rounded-2xl bg-surface/70'>
                            <div className='flex items-start justify-between gap-4 py-2'>
                                <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                    <UserAvatar username={post.author.username} size='md' />
                                    <div className='min-w-0'>
                                        <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${post.author.user_id}`)}>
                                            {post.author.username}
                                        </button>
                                        {post.author.affiliation ? <p className='mt-1 text-sm text-text-subtle'>{post.author.affiliation.trim()}</p> : null}
                                        <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>{post.author.bio?.trim() || t('profile.noBio')}</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </section>

                    <section className='space-y-3 px-1'>
                        <h2 className='text-xl font-semibold text-text'>{t('community.likeUsers')}</h2>

                        <div className='rounded-2xl bg-surface/70'>
                            <div className='space-y-3'>
                                {likes.length === 0 ? (
                                    <p className='text-sm text-text-muted'>{t('community.likeUsersEmpty')}</p>
                                ) : (
                                    likes.map((like, index) => (
                                        <div key={`${like.user_id}-${index}`} className='flex items-start justify-between gap-4 py-2'>
                                            <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                                <UserAvatar username={like.username} size='md' />
                                                <div className='min-w-0'>
                                                    <button className='block max-w-full truncate text-left text-lg font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${like.user_id}`)}>
                                                        {like.username}
                                                    </button>

                                                    <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>
                                                        {like.affiliation && like.affiliation.trim().length > 0 ? `${like.affiliation} · ` : ''}
                                                        {like.bio && like.bio.trim().length > 0 ? like.bio : t('profile.noBio')}
                                                    </p>
                                                    <p className='mt-1 text-sm text-text-subtle'>{formatDateTime(like.created_at, localeTag)}</p>
                                                </div>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>

                            <div className='flex items-center justify-between px-2 pt-2 text-sm text-text-muted'>
                                <span>
                                    {likePagination.page} / {likePagination.total_pages || 1}
                                </span>

                                <div className='flex gap-2'>
                                    <button className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50' disabled={!likePagination.has_prev} onClick={() => setLikePage((prev) => Math.max(1, prev - 1))}>
                                        {t('common.previous')}
                                    </button>

                                    <button className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50' disabled={!likePagination.has_next} onClick={() => setLikePage((prev) => prev + 1)}>
                                        {t('common.next')}
                                    </button>
                                </div>
                            </div>
                        </div>
                    </section>
                </aside>
            </div>
        </section>
    )
}

export default CommunityDetail
