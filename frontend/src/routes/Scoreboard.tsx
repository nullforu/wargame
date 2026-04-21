import ScoreboardTimeline from '../components/ScoreboardTimeline'
import ScoreboardLeaderboard from '../components/ScoreboardLeaderboard'
import LoginRequired from '../components/LoginRequired'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Scoreboard = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const { state: auth } = useAuth()

    if (!auth.user) {
        return <LoginRequired title={t('scoreboard.title')} />
    }

    return (
        <section className='animate space-y-4'>
            <h2 className='text-2xl font-semibold text-text'>{t('scoreboard.title')}</h2>

            <div className='grid min-w-0 grid-cols-1 gap-4'>
                <ScoreboardTimeline />
                <ScoreboardLeaderboard />
            </div>
        </section>
    )
}

export default Scoreboard
