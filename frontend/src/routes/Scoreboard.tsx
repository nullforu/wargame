import { useEffect, useRef, useState } from 'react'
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
    const [refreshTrigger, setRefreshTrigger] = useState(0)
    const [liveUpdatesEnabled, setLiveUpdatesEnabled] = useState(true)
    const reconnectTimeoutRef = useRef<number | null>(null)
    const eventSourceRef = useRef<EventSource | null>(null)

    useEffect(() => {
        if (!auth.user) {
            return () => {}
        }
        if (!liveUpdatesEnabled) {
            if (reconnectTimeoutRef.current !== null) {
                window.clearTimeout(reconnectTimeoutRef.current)
                reconnectTimeoutRef.current = null
            }
            if (eventSourceRef.current) {
                eventSourceRef.current.close()
                eventSourceRef.current = null
            }
            return
        }
        if (typeof EventSource === 'undefined') return
        const apiBase = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'
        const url = `${apiBase}/api/scoreboard/stream`
        let active = true

        const cleanupEventSource = () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close()
                eventSourceRef.current = null
            }
        }

        const scheduleReconnect = () => {
            if (!active) return
            if (reconnectTimeoutRef.current !== null) return
            reconnectTimeoutRef.current = window.setTimeout(() => {
                reconnectTimeoutRef.current = null
                connect()
            }, 1000)
        }

        const handleScoreboard = (_event: MessageEvent) => {
            try {
                setRefreshTrigger((value) => value + 1)
            } catch {
                setRefreshTrigger((value) => value + 1)
            }
        }

        const connect = () => {
            cleanupEventSource()
            const eventSource = new EventSource(url)
            eventSourceRef.current = eventSource
            eventSource.addEventListener('scoreboard', handleScoreboard)
            eventSource.onerror = () => {
                cleanupEventSource()
                scheduleReconnect()
            }
        }

        setRefreshTrigger((value) => value + 1)
        connect()

        return () => {
            active = false
            if (reconnectTimeoutRef.current !== null) {
                window.clearTimeout(reconnectTimeoutRef.current)
                reconnectTimeoutRef.current = null
            }
            cleanupEventSource()
        }
    }, [auth.user, liveUpdatesEnabled])

    if (!auth.user) {
        return <LoginRequired title={t('scoreboard.title')} />
    }

    return (
        <section className='animate'>
            <div>
                <h2 className='text-3xl text-text'>{t('scoreboard.title')}</h2>
            </div>

            <div className='mt-6 space-y-3'>
                <div className='flex items-center justify-end gap-3'>
                    <select value={liveUpdatesEnabled ? 'on' : 'off'} onChange={(e) => setLiveUpdatesEnabled(e.target.value === 'on')} className='p-1 text-xs text-text outline-none focus:border-accent'>
                        <option value='on'>{t('scoreboard.liveOn')}</option>
                        <option value='off'>{t('scoreboard.liveOff')}</option>
                    </select>
                </div>

                <div className='grid min-w-0 grid-cols-1 gap-6'>
                    <ScoreboardTimeline refreshTrigger={refreshTrigger} />
                    <ScoreboardLeaderboard refreshTrigger={refreshTrigger} />
                </div>
            </div>
        </section>
    )
}

export default Scoreboard
