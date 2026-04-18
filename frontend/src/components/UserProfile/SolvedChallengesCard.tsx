import type { SolvedChallenge } from '../../lib/types'
import { useT } from '../../lib/i18n'

interface SolvedChallengesCardProps {
    solved: SolvedChallenge[]
    formatDateTime: (value: string) => string
}

const SolvedChallengesCard = ({ solved, formatDateTime }: SolvedChallengesCardProps) => {
    const t = useT()

    return (
        <div className='mt-8 rounded-2xl border border-border bg-surface p-6'>
            <div className='flex items-center justify-between'>
                <h3 className='text-lg text-text'>{t('profile.solvedChallenges')}</h3>
                <span className='text-sm text-text-muted'>{solved.length === 1 ? t('profile.problemSingular', { count: solved.length }) : t('profile.problemPlural', { count: solved.length })}</span>
            </div>

            <div className='mt-6 space-y-3'>
                {solved.map((item) => (
                    <div key={item.challenge_id} className='rounded-xl border border-border bg-surface-muted p-5'>
                        <h4 className='text-base font-medium text-text'>
                            {item.title}
                            <span className='ml-2 text-xs text-accent'>{t('common.pointsShort', { points: item.points })}</span>
                        </h4>
                        <p className='mt-2 text-sm text-text-muted'>{t('profile.solvedAt', { time: formatDateTime(item.solved_at) })}</p>
                    </div>
                ))}

                {solved.length === 0 ? (
                    <div className='rounded-xl border border-border bg-surface-muted p-8 text-center'>
                        <p className='text-sm text-text-muted'>{t('profile.noSolved')}</p>
                    </div>
                ) : null}
            </div>
        </div>
    )
}

export default SolvedChallengesCard
