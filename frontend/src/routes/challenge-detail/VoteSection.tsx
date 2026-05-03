import type { Challenge, ChallengeVote, PaginationMeta } from '../../lib/types'
import { LEVEL_VOTE_OPTIONS, levelBarClass } from '../../lib/level'
import UserAvatar from '../../components/UserAvatar'
import { navigate } from '../../lib/router'
import DismissibleNotice from '../../components/DismissibleNotice'

interface VoteSectionProps {
    challenge: Challenge
    isAuthenticated: boolean
    votePagination: PaginationMeta
    voteCountsByLevel: Map<number, number>
    maxVoteCount: number
    selectedLevel: number | null
    votes: ChallengeVote[]
    voteMessage: string
    voteSubmittedMessage: string
    formatTimestamp: (value: string) => string
    onOpenRevote: () => void
    onPrevVotePage: () => void
    onNextVotePage: () => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const VOTE_NOTICE_DISMISSED_KEY = 'challenge_vote_notice_dismissed'

const VoteSection = ({
    challenge,
    isAuthenticated,
    votePagination,
    voteCountsByLevel,
    maxVoteCount,
    selectedLevel,
    votes,
    voteMessage,
    voteSubmittedMessage,
    formatTimestamp,
    onOpenRevote,
    onPrevVotePage,
    onNextVotePage,
    t,
}: VoteSectionProps) => {
    if (challenge.is_locked) return null

    return (
        <section className='mt-7'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
                <h3 className='text-lg font-semibold text-text'>
                    {t('challenge.voteTitle')} <span className='text-accent'>{votePagination.total_count}</span>
                </h3>
                {isAuthenticated && challenge.is_solved ? (
                    <button type='button' className='rounded-md border border-accent/40 px-3 py-1.5 text-xs font-medium text-accent hover:bg-accent/10' onClick={onOpenRevote}>
                        {t('challenge.voteAgain')}
                    </button>
                ) : null}
            </div>
            <DismissibleNotice className='mt-3 rounded-lg' closeAriaLabel={t('common.close')} storageKey={VOTE_NOTICE_DISMISSED_KEY} size='small'>
                {challenge.is_solved ? t('challenge.voteEnabledHint') : t('challenge.voteDisabledHint')}
            </DismissibleNotice>

            <div className='mt-4 grid items-stretch gap-5 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]'>
                <div className='flex min-h-105 flex-col'>
                    <p className='text-sm font-semibold text-text'>{t('challenge.voteResults')}</p>
                    <div className='mt-3 flex-1 rounded-xl bg-surface-muted/60 px-3 py-4 dark:bg-surface-muted/80'>
                        <div className='grid h-full min-h-75 grid-cols-10 items-end gap-1 sm:gap-2'>
                            {LEVEL_VOTE_OPTIONS.map((level) => {
                                const count = voteCountsByLevel.get(level) ?? 0
                                const height = maxVoteCount > 0 ? Math.max(8, Math.round((count / maxVoteCount) * 180)) : 8
                                const isSelected = selectedLevel === level
                                return (
                                    <div key={level} className='flex min-w-0 flex-col items-center justify-end gap-2'>
                                        <div className='flex w-full items-end justify-center'>
                                            <div className={`w-3 rounded-full transition-all ${levelBarClass(level)} ${count > 0 ? 'opacity-100' : 'opacity-35'}`} style={{ height: `${height}px` }} />
                                        </div>
                                        <span
                                            className={`inline-flex h-7 w-7 items-center justify-center rounded-full border text-[11px] font-semibold transition ${
                                                isSelected ? 'border-accent bg-accent text-white shadow-sm' : 'border-border/70 bg-surface text-text dark:border-border dark:bg-surface-subtle dark:text-text'
                                            }`}
                                        >
                                            {level}
                                        </span>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                    {!isAuthenticated ? (
                        <p className='mt-2 text-xs text-warning'>
                            {t('challenge.voteLoginRequired')}{' '}
                            <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                                {t('auth.loginLink')}
                            </a>
                        </p>
                    ) : null}
                    {voteMessage ? <p className={`mt-2 text-xs ${voteMessage === voteSubmittedMessage ? 'text-success' : 'text-danger'}`}>{voteMessage}</p> : null}
                </div>

                <div className='flex min-h-105 flex-col'>
                    <div className='flex flex-wrap items-center justify-between gap-2'>
                        <p className='text-sm font-semibold text-text'>{t('challenge.voteLogTitle')}</p>
                        <div className='flex flex-wrap items-center gap-2 text-xs text-text-muted'>
                            <button type='button' className='rounded-md border border-border/70 px-2 py-1 disabled:opacity-40' disabled={!votePagination.has_prev} onClick={onPrevVotePage}>
                                {t('common.previous')}
                            </button>
                            <span>
                                {votePagination.page} / {votePagination.total_pages || 1}
                            </span>
                            <button type='button' className='rounded-md border border-border/70 px-2 py-1 disabled:opacity-40' disabled={!votePagination.has_next} onClick={onNextVotePage}>
                                {t('common.next')}
                            </button>
                        </div>
                    </div>
                    <div className='mt-3 flex-1 space-y-3'>
                        {votes.length === 0 ? (
                            <p className='flex h-full min-h-75 items-center text-sm text-text-muted'>{t('challenge.voteLogEmpty')}</p>
                        ) : (
                            votes.map((vote) => (
                                <div key={`${vote.user_id}-${vote.updated_at}`} className='flex flex-wrap items-start gap-3 rounded-xl p-2.5 sm:flex-nowrap'>
                                    <UserAvatar username={vote.username} profileImage={vote.profile_image ?? null} size='sm' />
                                    <div className='min-w-0 flex-1'>
                                        <button className='block max-w-full truncate text-left text-sm font-semibold text-text hover:text-accent' onClick={() => navigate(`/users/${vote.user_id}`)}>
                                            {vote.username}
                                        </button>
                                        <p className='mt-1 text-sm text-text-muted'>
                                            {t('challenge.voteLogLine', {
                                                level: vote.level,
                                            })}
                                        </p>
                                    </div>
                                    <span className='w-full text-right text-xs text-text-subtle sm:w-auto'>{formatTimestamp(vote.updated_at)}</span>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>
        </section>
    )
}

export default VoteSection
