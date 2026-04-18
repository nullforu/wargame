import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'

interface RouteProps {
    routeParams?: Record<string, string>
}

const NotFound = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()

    return (
        <section className='fade-in'>
            <div className='rounded-3xl border border-border bg-surface p-10 text-center'>
                <h2 className='text-3xl text-text'>{t('notFound.title')}</h2>
                <p className='mt-2 text-sm text-text-muted'>{t('notFound.message')}</p>
                <a
                    href='/'
                    className='mt-6 inline-flex rounded-full bg-accent px-6 py-2 text-sm text-contrast-foreground transition hover:bg-accent-strong cursor-pointer'
                    onClick={(event) => {
                        event.preventDefault()
                        navigate('/')
                    }}
                >
                    {t('notFound.home')}
                </a>
            </div>
        </section>
    )
}

export default NotFound
