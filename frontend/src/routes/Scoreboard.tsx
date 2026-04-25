import ScoreboardTimeline from '../components/ScoreboardTimeline'
import LegacyLeaderboard from '../components/LegacyLeaderboard'
import { useT } from '../lib/i18n'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Scoreboard = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()

    return (
        <section className='animate space-y-4'>
            <h2 className='text-2xl font-semibold text-text'>{t('scoreboard.title')}</h2>

            <div className='grid min-w-0 grid-cols-1 gap-4'>
                <ScoreboardTimeline />
                <LegacyLeaderboard />
            </div>
        </section>
    )
}

export default Scoreboard
