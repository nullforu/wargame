import { navigate } from '../lib/router'
import { useT } from '../lib/i18n'

interface RouteProps {
    routeParams?: Record<string, string>
}

const NotFound = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()

    return (
        <section className='animate'>
            <div className='border border-border bg-surface p-10 text-center'>
                <h2 className='text-3xl font-semibold text-text'>{t('notFound.title')}</h2>
                <p className='mt-2 text-sm text-text-muted'>{t('notFound.message')}</p>
                <a
                    href='/'
                    className='mt-5 inline-flex border border-accent bg-accent px-5 py-2 text-sm text-white hover:bg-accent-strong'
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
