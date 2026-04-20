import { useT } from '../lib/i18n'
import { navigate } from '../lib/router'

interface LoginRequiredProps {
    title: string
}

const LoginRequired = ({ title }: LoginRequiredProps) => {
    const t = useT()

    return (
        <section className='animate'>
            <h2 className='text-2xl font-semibold text-text'>{title}</h2>
            <div className='mt-4 rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning'>
                {t('profile.loginToViewPrefix')}{' '}
                <a className='underline' href='/login' onClick={(e) => navigate('/login', e)}>
                    {t('auth.loginLink')}
                </a>{' '}
                {t('profile.loginToViewSuffix')}
            </div>
        </section>
    )
}

export default LoginRequired
