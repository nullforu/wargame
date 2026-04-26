import type { Challenge } from '../../lib/types'
import { getCategoryKey } from '../../lib/i18n'
import FlagIcon from '../../components/FlagIcon'
import { LevelBadge } from '../Challenges'

interface ChallengeSummaryCardProps {
    challenge: Challenge
    levelLabel: string
    createdSummary: string
    t: (key: string, vars?: Record<string, string | number>) => string
}

const ChallengeSummaryCard = ({ challenge, levelLabel, createdSummary, t }: ChallengeSummaryCardProps) => {
    return (
        <div className='rounded-2xl border border-border/20 bg-surface p-5 shadow-sm'>
            <div className='flex min-w-0 items-center gap-3'>
                <LevelBadge level={challenge.level} />
                <span className='text-sm font-semibold text-accent'>{t('challenge.levelLabel', { level: levelLabel })}</span>
            </div>

            <div className='mt-3 flex items-center wrap-break-word text-xl font-semibold leading-tight text-text sm:text-2xl lg:text-3xl'>
                <h1>{challenge.title}</h1>
                {challenge.is_solved ? <FlagIcon className='ml-2 inline-block h-4 w-4 shrink-0 text-accent' /> : null}
            </div>

            <div className='mt-4 inline-flex rounded-lg bg-surface-muted px-2.5 py-1 text-xs text-text-muted'>{t(getCategoryKey(challenge.category))}</div>

            <div className='mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-text-muted'>
                <span>{t('common.pointsShort', { points: challenge.points })}</span>
                <span>{t('challenge.solvedCount', { count: challenge.solve_count })}</span>
                <span>
                    {t('common.createdAt')}: {createdSummary}
                </span>
            </div>
        </div>
    )
}

export default ChallengeSummaryCard
