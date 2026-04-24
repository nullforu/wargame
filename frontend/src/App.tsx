import { useEffect, useMemo, useState, type ReactElement } from 'react'
import Header from './components/Header'
import Home from './routes/Home'
import Login from './routes/Login'
import Register from './routes/Register'
import Challenges from './routes/Challenges'
import ChallengeDetail from './routes/ChallengeDetail'
import Scoreboard from './routes/Scoreboard'
import Ranking from './routes/Ranking'
import Users from './routes/Users'
import UserProfile from './routes/UserProfile'
import Admin from './routes/Admin'
import NotFound from './routes/NotFound'
import { useAuth } from './lib/auth'
import { useApi } from './lib/useApi'
import { useLocale, useT } from './lib/i18n'
import { SITE_CONFIG } from './lib/siteConfig'
import './index.css'

interface RouteProps {
    routeParams?: Record<string, string>
}

type RouteComponent = (props: RouteProps) => ReactElement

const routes: Record<string, RouteComponent> = {
    '/': Home,
    '/login': Login,
    '/register': Register,
    '/challenges': Challenges,
    '/ranking': Ranking,
    '/scoreboard': Scoreboard,
    '/profile': UserProfile,
    '/users': Users,
    '/admin': Admin,
}

const dynamicRoutes: Array<{
    pattern: RegExp
    component: RouteComponent
    extractParams: (path: string) => Record<string, string>
}> = [
    {
        pattern: /^\/challenges\/(\d+)$/,
        component: ChallengeDetail,
        extractParams: (path) => {
            const match = path.match(/^\/challenges\/(\d+)$/)
            return match ? { id: match[1] } : { id: '' }
        },
    },
    {
        pattern: /^\/users\/(\d+)$/,
        component: UserProfile,
        extractParams: (path) => {
            const match = path.match(/^\/users\/(\d+)$/)
            return match ? { id: match[1] } : { id: '' }
        },
    },
]

const normalizePath = (path: string) => {
    return path.length > 1 && path.endsWith('/') ? path.replace(/\/+$/, '') : path
}

const App = () => {
    const t = useT()
    const { state: auth, setAuthUser, clearAuth } = useAuth()
    const locale = useLocale()
    const api = useApi()

    const [RouteComponent, setRouteComponent] = useState<RouteComponent>(() => Home)
    const [routeParams, setRouteParams] = useState<Record<string, string>>({})
    const [booting, setBooting] = useState(true)

    const updateRoute = () => {
        const nextPath = normalizePath(window.location.pathname || '/')

        if (routes[nextPath]) {
            setRouteComponent(() => routes[nextPath])
            setRouteParams({})
            return
        }

        for (const route of dynamicRoutes) {
            if (route.pattern.test(nextPath)) {
                setRouteComponent(() => route.component)
                setRouteParams(route.extractParams(nextPath))
                return
            }
        }

        setRouteComponent(() => NotFound)
        setRouteParams({})
    }

    const loadSession = async () => {
        if (!auth.accessToken) {
            setBooting(false)
            return
        }
        try {
            const user = await api.me()
            setAuthUser(user)
        } catch {
            clearAuth()
        } finally {
            setBooting(false)
        }
    }

    useEffect(() => {
        updateRoute()
        window.addEventListener('popstate', updateRoute)
        void loadSession()
        return () => window.removeEventListener('popstate', updateRoute)
    }, [])

    useEffect(() => {
        if (typeof document !== 'undefined') {
            document.documentElement.lang = locale
        }
    }, [locale])

    useEffect(() => {
        if (typeof document !== 'undefined') {
            document.title = SITE_CONFIG.title || t('app.title')
        }
    }, [t])

    const content = useMemo(() => {
        if (booting) {
            return <div className='rounded-xl border border-border bg-surface px-6 py-10 text-center text-sm text-text-muted'>{t('app.checkingSession')}</div>
        }
        return <RouteComponent routeParams={routeParams} />
    }, [RouteComponent, booting, routeParams, t])

    return (
        <div className='min-h-screen bg-background flex flex-col overflow-x-hidden'>
            <Header user={auth.user} />
            <main className='mx-auto w-full max-w-7xl flex-1 overflow-x-hidden px-4 py-5 md:px-6 md:py-6'>{content}</main>
            <footer className='border-t border-border bg-surface-muted py-5 text-center text-xs text-text-subtle dark:border-border dark:bg-surface dark:text-text-muted'>
                <p className='mx-auto max-w-7xl px-4 md:px-6'>{t('footer.copyright')}</p>
            </footer>
        </div>
    )
}

export default App
