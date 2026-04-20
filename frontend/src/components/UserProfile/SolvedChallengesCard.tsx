import type { SolvedChallenge } from '../../lib/types'
import { useT } from '../../lib/i18n'

interface SolvedChallengesCardProps {
    solved: SolvedChallenge[]
    formatDateTime: (value: string) => string
}

const SolvedChallengesCard = ({ solved, formatDateTime }: SolvedChallengesCardProps) => {
    const t = useT()

    return (
        <div className='mt-8 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-2xl md:border md:border-border md:bg-surface md:p-6'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
                <h3 className='text-lg text-text'>{t('profile.solvedChallenges')}</h3>
                <span className='text-sm text-text-muted'>{solved.length === 1 ? t('profile.problemSingular', { count: solved.length }) : t('profile.problemPlural', { count: solved.length })}</span>
            </div>

            <div className='mt-6 space-y-3'>
                {solved.map((item) => (
                    <div key={item.challenge_id} className='rounded-none border-0 bg-transparent p-3 md:rounded-xl md:border md:border-border md:bg-surface-muted md:p-5'>
                        <h4 className='text-base font-medium text-text'>
                            {item.title}
                            <span className='ml-2 text-xs text-accent'>{t('common.pointsShort', { points: item.points })}</span>
                        </h4>
                        <p className='mt-2 text-sm text-text-muted'>{t('profile.solvedAt', { time: formatDateTime(item.solved_at) })}</p>
                    </div>
                ))}

                {solved.length === 0 ? (
                    <div className='rounded-none border-0 bg-surface-muted p-5 text-center md:rounded-xl md:border md:border-border md:p-8'>
                        <p className='text-sm text-text-muted'>{t('profile.noSolved')}</p>
                    </div>
                ) : null}
            </div>
        </div>
    )
}

export default SolvedChallengesCard
