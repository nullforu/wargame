import UserAvatar from '../../components/UserAvatar'
import type { ChallengeCommentItem, PaginationMeta } from '../../lib/types'

interface ChallengeCommentsPanelProps {
    comments: ChallengeCommentItem[]
    commentPagination: PaginationMeta
    commentPage: number
    commentLoading: boolean
    commentError: string
    commentInput: string
    commentSubmitting: boolean
    editingCommentID: number | null
    editingCommentContent: string
    isAuthenticated: boolean
    currentUserID?: number
    formatTimestamp: (value: string) => string
    onCommentInputChange: (value: string) => void
    onCreateComment: () => void
    onEditStart: (id: number, content: string) => void
    onEditCancel: () => void
    onEditContentChange: (value: string) => void
    onUpdateComment: (id: number) => void
    onDeleteComment: (id: number) => void
    onSetCommentPage: (page: number) => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const ChallengeCommentsPanel = ({
    comments,
    commentPagination,
    commentPage,
    commentLoading,
    commentError,
    commentInput,
    commentSubmitting,
    editingCommentID,
    editingCommentContent,
    isAuthenticated,
    currentUserID,
    formatTimestamp,
    onCommentInputChange,
    onCreateComment,
    onEditStart,
    onEditCancel,
    onEditContentChange,
    onUpdateComment,
    onDeleteComment,
    onSetCommentPage,
    t,
}: ChallengeCommentsPanelProps) => {
    return (
        <section className='space-y-3 px-1'>
            <h2 className='text-xl font-semibold text-text'>
                {t('challengeComment.sectionTitle')} <span className='text-accent'>{commentPagination.total_count}</span>
            </h2>

            <div className='rounded-lg border border-border/70 bg-surface'>
                <textarea
                    className='h-24 w-full resize-none rounded-t-lg border-0 bg-transparent p-3 text-sm text-text outline-none placeholder:text-text-subtle/80'
                    placeholder={t('challengeComment.placeholder')}
                    maxLength={500}
                    value={commentInput}
                    onChange={(e) => onCommentInputChange(e.target.value)}
                    disabled={!isAuthenticated || commentSubmitting}
                />
                <div className='flex items-center justify-between border-t border-border/70 px-3 py-2'>
                    <span className='text-xs text-text-subtle'>{commentInput.length}/500</span>
                    <button className='rounded border border-border px-2 py-1 text-xs disabled:opacity-40' disabled={!isAuthenticated || commentSubmitting || commentInput.trim().length === 0} onClick={onCreateComment}>
                        {t('challengeComment.submit')}
                    </button>
                </div>
            </div>

            {!isAuthenticated ? <p className='text-xs text-warning'>{t('challengeComment.loginRequired')}</p> : null}
            {commentError ? <p className='text-xs text-danger'>{commentError}</p> : null}

            <div className='space-y-3'>
                {commentLoading ? (
                    <div className='space-y-3'>
                        {[0, 1].map((idx) => (
                            <div key={idx} className='rounded-lg bg-surface/50 p-3 animate-pulse'>
                                <div className='flex items-center justify-between'>
                                    <div className='h-4 w-24 rounded bg-surface-muted' />
                                    <div className='h-3 w-28 rounded bg-surface-muted' />
                                </div>
                                <div className='mt-3 h-3 w-full rounded bg-surface-muted' />
                                <div className='mt-2 h-3 w-4/5 rounded bg-surface-muted' />
                            </div>
                        ))}
                    </div>
                ) : null}
                {!commentLoading &&
                    comments.map((item) => {
                        const isMine = currentUserID === item.author.user_id
                        const isEditing = editingCommentID === item.id
                        return (
                            <div key={item.id} className='rounded-lg bg-surface/50 p-3'>
                                <div className='flex items-start justify-between gap-2'>
                                    <div className='flex min-w-0 items-center gap-2'>
                                        <UserAvatar username={item.author.username} profileImage={item.author.profile_image ?? null} size='sm' />
                                        <span className='truncate text-sm font-semibold text-text'>{item.author.username}</span>
                                    </div>
                                    <span className='shrink-0 text-xs text-text-subtle'>{formatTimestamp(item.created_at)}</span>
                                </div>
                                {isEditing ? (
                                    <div className='mt-2 space-y-2'>
                                        <textarea className='w-full rounded border border-border bg-surface px-2 py-1 text-sm' value={editingCommentContent} onChange={(e) => onEditContentChange(e.target.value)} maxLength={500} />
                                        <div className='flex gap-2'>
                                            <button className='rounded border border-border px-2 py-1 text-xs' onClick={onEditCancel}>
                                                {t('common.cancel')}
                                            </button>
                                            <button className='rounded bg-accent px-2 py-1 text-xs text-white' onClick={() => onUpdateComment(item.id)}>
                                                {t('common.save')}
                                            </button>
                                        </div>
                                    </div>
                                ) : (
                                    <p className='mt-2 whitespace-pre-wrap wrap-break-word text-sm text-text'>{item.content}</p>
                                )}
                                {isMine && !isEditing ? (
                                    <div className='mt-2 flex gap-3 text-xs'>
                                        <button className='text-accent' onClick={() => onEditStart(item.id, item.content)}>
                                            {t('common.edit')}
                                        </button>
                                        <button className='text-danger' onClick={() => onDeleteComment(item.id)}>
                                            {t('common.delete')}
                                        </button>
                                    </div>
                                ) : null}
                            </div>
                        )
                    })}
                {!commentLoading && comments.length === 0 ? <p className='text-sm text-text-muted'>{t('challengeComment.empty')}</p> : null}
            </div>

            <div className='flex items-center justify-between pt-1 text-sm text-text-muted'>
                <span>
                    {commentPagination.page} / {commentPagination.total_pages || 1}
                </span>
                <div className='flex gap-2'>
                    <button
                        className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50'
                        disabled={!commentPagination.has_prev || commentLoading}
                        onClick={() => onSetCommentPage(Math.max(1, commentPage - 1))}
                    >
                        {t('common.previous')}
                    </button>
                    <button className='rounded-lg bg-surface-muted px-3 py-1.5 hover:bg-surface-subtle disabled:opacity-50' disabled={!commentPagination.has_next || commentLoading} onClick={() => onSetCommentPage(commentPage + 1)}>
                        {t('common.next')}
                    </button>
                </div>
            </div>
        </section>
    )
}

export default ChallengeCommentsPanel
