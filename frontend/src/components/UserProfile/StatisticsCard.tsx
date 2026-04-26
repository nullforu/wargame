import { useT } from '../../lib/i18n'

interface StatisticsCardProps {
    totalPoints: number
    solvedCount: number
}

const StatisticsCard = ({ totalPoints, solvedCount }: StatisticsCardProps) => {
    const t = useT()

    return (
        <div className='mt-8 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-lg md:border md:border-border md:bg-surface md:p-6'>
            <h3 className='text-lg text-text'>{t('profile.statistics')}</h3>
            <div className='mt-4 grid gap-4 sm:grid-cols-2'>
                <div className='rounded-none border-0 bg-surface-muted p-4 md:rounded-xl md:border md:border-border'>
                    <p className='text-xs text-text-muted'>{t('profile.totalPoints')}</p>
                    <p className='mt-1 text-2xl font-semibold text-text'>{totalPoints}</p>
                </div>
                <div className='rounded-none border-0 bg-surface-muted p-4 md:rounded-xl md:border md:border-border'>
                    <p className='text-xs text-text-muted'>{t('profile.problemsSolved')}</p>
                    <p className='mt-1 text-2xl font-semibold text-text'>{solvedCount}</p>
                </div>
            </div>
        </div>
    )
}

export default StatisticsCard
