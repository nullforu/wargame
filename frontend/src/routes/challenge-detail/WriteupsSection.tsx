import DismissibleNotice from '../../components/DismissibleNotice'
import UserAvatar from '../../components/UserAvatar'
import { navigate } from '../../lib/router'
import type { PaginationMeta, Writeup } from '../../lib/types'

interface WriteupsSectionProps {
    challengeId: number
    challengeSolved: boolean
    hasMyWriteup: boolean
    writeups: Writeup[]
    writeupLoading: boolean
    writeupError: string
    canViewWriteupContent: boolean
    writeupPage: number
    writeupPagination: PaginationMeta
    formatWriteupDate: (value: string) => string
    onSetWriteupPage: (page: number) => void
    onPushWriteupPageQuery: (page: number) => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const WRITEUP_NOTICE_DISMISSED_KEY = 'writeup_notice_dismissed'

const WriteupsSection = ({
    challengeId,
    challengeSolved,
    hasMyWriteup,
    writeups,
    writeupLoading,
    writeupError,
    canViewWriteupContent,
    writeupPage,
    writeupPagination,
    formatWriteupDate,
    onSetWriteupPage,
    onPushWriteupPageQuery,
    t,
}: WriteupsSectionProps) => {
    return (
        <section className='mt-8'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
                <h3 className='text-lg font-semibold text-text'>
                    {t('writeup.sectionTitle')} <span className='text-accent'>{writeupPagination.total_count}</span>
                </h3>
                {challengeSolved && !hasMyWriteup ? (
                    <button
                        className='rounded-md border border-accent/40 px-3 py-1.5 text-xs font-medium text-accent hover:bg-accent/10'
                        onClick={() => {
                            navigate(`/challenges/${challengeId}/writeup`)
                        }}
                    >
                        {t('writeup.createTitle')}
                    </button>
                ) : null}
            </div>

            <DismissibleNotice className='mt-3 rounded-lg' closeAriaLabel={t('common.close')} storageKey={WRITEUP_NOTICE_DISMISSED_KEY} size='small'>
                {t('writeup.hint')}
            </DismissibleNotice>

            <div className='mt-4 space-y-2'>
                {writeupLoading ? <p className='py-4 text-sm text-text-muted'>{t('common.loading')}</p> : null}
                {!writeupLoading && writeupError ? <p className='py-4 text-sm text-danger'>{writeupError}</p> : null}
                {!writeupLoading &&
                    !writeupError &&
                    writeups.map((item) => (
                        <button
                            key={item.id}
                            className='w-full rounded-none border-b border-border/60 text-left disabled:cursor-not-allowed disabled:opacity-60 pb-4'
                            onClick={() => navigate(`/writeups/${item.id}`)}
                            disabled={!canViewWriteupContent}
                        >
                            <div className='flex items-start justify-between gap-4 py-1'>
                                <div className='min-w-0 flex flex-1 items-center gap-3.75'>
                                    <UserAvatar username={item.author.username} profileImage={item.author.profile_image ?? null} size='md' />
                                    <div className='min-w-0'>
                                        <div className='flex items-center gap-2'>
                                            <span className='block max-w-full truncate text-left text-base font-semibold text-text'>{item.author.username}</span>
                                        </div>
                                        <p className='mt-1 max-w-full truncate text-sm text-text-subtle'>
                                            {item.author.affiliation && item.author.affiliation.trim().length > 0 ? `${item.author.affiliation} · ` : ''}
                                            {item.author.bio && item.author.bio.trim().length > 0 ? item.author.bio : t('profile.noBio')}
                                        </p>
                                    </div>
                                </div>

                                <span className='shrink-0 text-sm text-text-subtle'>{formatWriteupDate(item.created_at)}</span>
                            </div>
                        </button>
                    ))}
                {!writeupLoading && !writeupError && writeups.length === 0 ? <p className='py-4 text-sm text-text-muted'>{t('writeup.empty')}</p> : null}
            </div>

            {writeupPagination.total_pages > 0 ? (
                <div className='mt-6 flex items-center justify-center gap-4 text-sm text-text-muted'>
                    <button
                        disabled={!writeupPagination.has_prev || writeupLoading}
                        className='rounded border border-border px-2 py-1 disabled:opacity-50'
                        onClick={() => {
                            const nextPage = Math.max(1, writeupPage - 1)
                            onSetWriteupPage(nextPage)
                            onPushWriteupPageQuery(nextPage)
                        }}
                    >
                        {t('common.previous')}
                    </button>
                    <span>
                        {writeupPagination.page} / {writeupPagination.total_pages || 1}
                    </span>
                    <button
                        disabled={!writeupPagination.has_next || writeupLoading}
                        className='rounded border border-border px-2 py-1 disabled:opacity-50'
                        onClick={() => {
                            const nextPage = writeupPage + 1
                            onSetWriteupPage(nextPage)
                            onPushWriteupPageQuery(nextPage)
                        }}
                    >
                        {t('common.next')}
                    </button>
                </div>
            ) : null}
        </section>
    )
}

export default WriteupsSection
