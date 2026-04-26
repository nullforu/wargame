import { getCategoryKey, useT } from '../lib/i18n'
import type { Challenge } from '../lib/types'

interface ChallengeRowProps {
    challenge: Challenge
    isSolved: boolean
    onClick: () => void
}

const ChallengeRow = ({ challenge, isSolved, onClick }: ChallengeRowProps) => {
    const t = useT()
    const isLocked = challenge.is_locked === true
    const isActive = 'is_active' in challenge ? challenge.is_active !== false : true
    const hasCategory = 'category' in challenge && challenge.category

    return (
        <article
            className={`cursor-pointer border-b border-border px-4 py-4 transition hover:bg-surface-muted ${!isActive ? 'opacity-60' : ''}`}
            onClick={onClick}
            role='button'
            tabIndex={0}
            onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    onClick()
                }
            }}
        >
            <div className='flex items-center justify-between gap-4'>
                <div className='min-w-0 flex-1'>
                    <h3 className='truncate text-base font-medium text-text'>{challenge.title}</h3>
                    <div className='mt-1 flex flex-wrap items-center gap-3 text-xs text-text-muted'>
                        {hasCategory ? <span>{t(getCategoryKey(challenge.category))}</span> : null}
                        <span>{t('common.pointsShort', { points: challenge.points })}</span>
                    </div>
                </div>
                {isLocked ? (
                    <span className='rounded-full bg-warning/20 px-2.5 py-1 text-xs text-warning-strong'>{t('challenge.lockedLabel')}</span>
                ) : isSolved ? (
                    <span className='rounded-full bg-success/20 px-2.5 py-1 text-xs text-success'>{t('challenge.solvedLabel')}</span>
                ) : !isActive ? (
                    <span className='rounded-full bg-surface/10 px-2.5 py-1 text-xs text-text-muted'>{t('challenge.inactiveLabel')}</span>
                ) : null}
            </div>
        </article>
    )
}

const CHALLENGES_VIEW_SKELETON_ROWS = 5

interface ChallengesViewProps {
    title: string
    summaryText?: string
    stackSummaryText?: string
    showSummary: boolean
    groupByCategory: boolean
    toggleLabel: string
    onGroupByCategoryChange: (checked: boolean) => void
    loading: boolean
    loadingText: string
    errorMessage: string
    notStarted: boolean
    notStartedText: string
    startAtLabel: string
    startAtValue: string
    endAtLabel: string
    endAtValue: string
    ended: boolean
    endedText: string
    challenges: Challenge[]
    groupedCategories: Array<{ id: string; label: string; items: Challenge[] }>
    solvedIds: Set<number>
    onSelectChallenge: (challenge: Challenge) => void
}

const ChallengesView = ({
    title,
    summaryText,
    stackSummaryText,
    showSummary,
    groupByCategory,
    toggleLabel,
    onGroupByCategoryChange,
    loading,
    loadingText,
    errorMessage,
    notStarted,
    notStartedText,
    startAtLabel,
    startAtValue,
    endAtLabel,
    endAtValue,
    ended,
    endedText,
    challenges,
    groupedCategories,
    solvedIds,
    onSelectChallenge,
}: ChallengesViewProps) => {
    const renderChallengeList = (items: Challenge[]) => (
        <div className='mt-6 overflow-hidden rounded-2xl border border-border bg-surface'>
            {items.map((challenge) => (
                <ChallengeRow key={challenge.id} challenge={challenge} isSolved={solvedIds.has(challenge.id)} onClick={() => onSelectChallenge(challenge)} />
            ))}
        </div>
    )

    const renderGroupedChallenges = () => (
        <div className='mt-6 space-y-8'>
            {groupedCategories.map((category) => {
                if (category.items.length === 0) return null

                return (
                    <div key={category.id}>
                        <h3 className='text-lg font-semibold text-text'>{category.label}</h3>
                        <div className='mt-4 overflow-hidden rounded-2xl border border-border bg-surface'>
                            {category.items.map((challenge) => (
                                <ChallengeRow key={challenge.id} challenge={challenge} isSolved={solvedIds.has(challenge.id)} onClick={() => onSelectChallenge(challenge)} />
                            ))}
                        </div>
                    </div>
                )
            })}
        </div>
    )

    const renderChallenges = () => (groupByCategory ? renderGroupedChallenges() : renderChallengeList(challenges))

    const renderBody = () => {
        if (loading) {
            return (
                <div className='mt-6 overflow-hidden rounded-2xl border border-border bg-surface'>
                    <p className='sr-only'>{loadingText}</p>
                    {Array.from({ length: CHALLENGES_VIEW_SKELETON_ROWS }, (_, idx) => (
                        <div key={`challenges-view-skeleton-${idx}`} className='border-b border-border px-4 py-4 last:border-b-0'>
                            <div className='flex items-center justify-between gap-4 animate-pulse'>
                                <div className='min-w-0 flex-1 space-y-2'>
                                    <div className='h-4 w-2/3 rounded bg-surface-muted' />
                                    <div className='h-3 w-1/3 rounded bg-surface-muted' />
                                </div>
                                <div className='h-6 w-16 rounded-full bg-surface-muted' />
                            </div>
                        </div>
                    ))}
                </div>
            )
        }

        if (errorMessage) {
            return <div className='mt-6 rounded-2xl border border-danger/40 bg-danger/10 p-6 text-sm text-danger'>{errorMessage}</div>
        }

        if (notStarted) {
            return (
                <div className='mt-6 space-y-3 rounded-2xl border border-warning/40 bg-warning/10 p-6 text-sm text-warning-strong'>
                    <p>{notStartedText}</p>
                    <div className='text-xs text-text-muted'>
                        <p>
                            {startAtLabel}: {startAtValue}
                        </p>
                        <p>
                            {endAtLabel}: {endAtValue}
                        </p>
                    </div>
                </div>
            )
        }

        return (
            <>
                {ended ? <div className='mt-6 rounded-2xl border border-warning/40 bg-warning/10 p-6 text-sm text-warning-strong'>{endedText}</div> : null}
                {renderChallenges()}
            </>
        )
    }

    return (
        <section className='fade-in'>
            <div className='flex flex-wrap items-end justify-between gap-4'>
                <div>
                    <h2 className='text-3xl text-text'>{title}</h2>
                </div>
                {showSummary && summaryText ? (
                    <div className='rounded-lg border border-border bg-surface px-4 py-2 text-xs text-text'>
                        <p>{summaryText}</p>
                        {stackSummaryText ? <p className='mt-1 text-text-muted'>{stackSummaryText}</p> : null}
                    </div>
                ) : null}
                {!showSummary && stackSummaryText ? (
                    <div className='rounded-lg border border-border bg-surface px-4 py-2 text-xs text-text'>
                        <p>{stackSummaryText}</p>
                    </div>
                ) : null}
            </div>
            <div className='mt-4 flex items-center justify-end'>
                <label className='flex items-center gap-2 text-xs text-text-muted'>
                    <input className='h-4 w-4 accent-accent' type='checkbox' checked={groupByCategory} onChange={(event) => onGroupByCategoryChange(event.target.checked)} />
                    <span>{toggleLabel}</span>
                </label>
            </div>

            {renderBody()}
        </section>
    )
}

export default ChallengesView
