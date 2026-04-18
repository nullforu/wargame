import { useT } from '../lib/i18n'
import { navigate } from '../lib/router'

interface LoginRequiredProps {
    title: string
}

const LoginRequired = ({ title }: LoginRequiredProps) => {
    const t = useT()

    return (
        <section className='fade-in'>
            <h2 className='text-3xl text-text'>{title}</h2>
            <div className='mt-6 rounded-2xl border border-warning/40 bg-warning/10 p-6 text-sm text-warning-strong'>
                {t('profile.loginToViewPrefix')}{' '}
                <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                    {t('auth.loginLink')}
                </a>{' '}
                {t('profile.loginToViewSuffix')}
            </div>
        </section>
    )
}

export default LoginRequired
