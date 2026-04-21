import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Home = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const { state: auth } = useAuth()

    return (
        <section className='animate space-y-4'>
            <div className='grid gap-3 lg:grid-cols-[2fr_1fr]'>
                <div className='overflow-hidden rounded-xl border border-border bg-surface dark:bg-surface dark:border-border'>
                    <div className='border-b border-border bg-surface-muted px-4 py-2 text-xs text-text-muted dark:border-border dark:bg-surface-muted dark:text-text-muted'>Community Portal</div>
                    <div className='px-5 py-5'>
                        <h1 className='text-[28px] leading-tight text-text dark:text-text'>{t('home.heroTitle')}</h1>
                        <p className='mt-3 max-w-3xl text-sm leading-6 text-text-muted dark:text-text-muted'>{t('home.heroBody')}</p>
                        <div className='mt-5 flex flex-wrap gap-2'>
                            <a href='/challenges' className='rounded-md border border-accent bg-accent px-4 py-2 text-sm text-white transition hover:bg-accent-strong' onClick={(e) => navigate('/challenges', e)}>
                                {t('home.ctaChallenges')}
                            </a>
                            {!auth.user ? (
                                <a
                                    href='/register'
                                    className='rounded-md border border-border bg-surface px-4 py-2 text-sm text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                                    onClick={(e) => navigate('/register', e)}
                                >
                                    {t('home.ctaSignUp')}
                                </a>
                            ) : (
                                <a
                                    href='/profile'
                                    className='rounded-md border border-border bg-surface px-4 py-2 text-sm text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface dark:text-text dark:hover:bg-surface-muted'
                                    onClick={(e) => navigate('/profile', e)}
                                >
                                    {t('nav.profile')}
                                </a>
                            )}
                        </div>
                    </div>
                </div>

                <aside className='rounded-xl border border-border bg-surface p-4 dark:bg-surface dark:border-border'>
                    <h2 className='text-base font-semibold text-text dark:text-text'>{t('home.quickTitle')}</h2>
                    <p className='mt-1 text-sm text-text-muted dark:text-text-muted'>{t('home.quickBody')}</p>
                    <div className='mt-4 space-y-2'>
                        <a
                            href='/scoreboard'
                            className='block rounded-md border border-border bg-surface-muted px-3 py-2 text-sm text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface-muted dark:text-text dark:hover:bg-surface-subtle'
                            onClick={(e) => navigate('/scoreboard', e)}
                        >
                            {t('nav.scoreboard')}
                        </a>
                        <a
                            href='/users'
                            className='block rounded-md border border-border bg-surface-muted px-3 py-2 text-sm text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface-muted dark:text-text dark:hover:bg-surface-subtle'
                            onClick={(e) => navigate('/users', e)}
                        >
                            {t('nav.users')}
                        </a>
                        {auth.user?.role === 'admin' ? (
                            <a
                                href='/admin'
                                className='block rounded-md border border-border bg-surface-muted px-3 py-2 text-sm text-text-muted transition hover:bg-surface-muted dark:border-border dark:bg-surface-muted dark:text-text dark:hover:bg-surface-subtle'
                                onClick={(e) => navigate('/admin', e)}
                            >
                                {t('nav.admin')}
                            </a>
                        ) : null}
                    </div>
                </aside>
            </div>
        </section>
    )
}

export default Home
