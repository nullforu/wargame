import { useEffect, useMemo, useState, type ReactElement } from 'react'
import Header from './components/Header'
import Home from './routes/Home'
import Login from './routes/Login'
import Register from './routes/Register'
import Challenges from './routes/Challenges'
import Scoreboard from './routes/Scoreboard'
import Users from './routes/Users'
import UserProfile from './routes/UserProfile'
import Admin from './routes/Admin'
import NotFound from './routes/NotFound'
import { useAuth } from './lib/auth'
import { useApi } from './lib/useApi'
import { useLocale, useT } from './lib/i18n'
import { useTheme } from './lib/theme'
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
    const { theme } = useTheme()
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
        loadSession()
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

    useEffect(() => {
        if (typeof document !== 'undefined') {
            if (theme === 'dark') {
                document.documentElement.classList.add('dark')
            } else {
                document.documentElement.classList.remove('dark')
            }
        }
    }, [theme])

    const content = useMemo(() => {
        if (booting) {
            return <div className='rounded-2xl border border-border bg-surface p-8 text-center text-text-muted'>{t('app.checkingSession')}</div>
        }
        return <RouteComponent routeParams={routeParams} />
    }, [RouteComponent, booting, routeParams, t])

    const isAdminPage = RouteComponent === Admin

    return (
        <div className='min-h-screen'>
            <Header user={auth.user} />
            <main className={`mx-auto w-full ${isAdminPage ? 'max-w-400' : 'max-w-6xl'} px-6 py-10`}>{content}</main>
            <footer className='border-t border-border py-6 text-center text-xs text-text-subtle'>
                <p>{t('footer.copyright')}</p>
            </footer>
        </div>
    )
}

export default App
