import { navigate } from '../lib/router'
import Markdown from '../components/Markdown'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'
import { SITE_CONFIG } from '../lib/siteConfig'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Home = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const { state: auth } = useAuth()

    return (
        <section className='animate'>
            <div className='relative overflow-hidden p-4 sm:p-8 md:p-10'>
                <div className='relative z-10'>
                    <h1 className='mt-2 text-2xl font-semibold text-text sm:mt-4 md:text-3xl lg:text-4xl'>{SITE_CONFIG.title}</h1>
                    <div className='mt-3 max-w-2xl text-base text-text sm:mt-4 sm:text-base md:text-lg'>
                        <Markdown content={SITE_CONFIG.description} />
                    </div>
                    <div className='mt-6 flex flex-wrap gap-3 sm:mt-8 sm:gap-4'>
                        <a
                            href='/challenges'
                            className='rounded-full bg-accent px-5 py-2.5 text-sm text-contrast-foreground transition hover:bg-accent-strong sm:px-6 sm:py-3 sm:text-base cursor-pointer'
                            onClick={(e) => navigate('/challenges', e)}
                        >
                            {t('home.ctaChallenges')}
                        </a>
                        {!auth.user ? (
                            <a
                                href='/register'
                                className='rounded-full border border-border px-5 py-2.5 text-sm text-text transition hover:border-accent sm:px-6 sm:py-3 sm:text-base cursor-pointer'
                                onClick={(e) => navigate('/register', e)}
                            >
                                {t('home.ctaSignUp')}
                            </a>
                        ) : null}
                    </div>
                </div>
            </div>
        </section>
    )
}

export default Home
