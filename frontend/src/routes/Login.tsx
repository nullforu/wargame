import { useState } from 'react'
import { formatApiError, type FieldErrors } from '../lib/utils'
import { navigate } from '../lib/router'
import FormMessage from '../components/FormMessage'
import { useT } from '../lib/i18n'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Login = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const { state: auth } = useAuth()
    const api = useApi()
    const [email, setEmail] = useState('')
    const [password, setPassword] = useState('')
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})

    const submit = async () => {
        setLoading(true)
        setErrorMessage('')
        setFieldErrors({})

        try {
            await api.login({ email, password })
            navigate('/challenges')
        } catch (error) {
            const formatted = formatApiError(error, t)
            setErrorMessage(formatted.message)
            setFieldErrors(formatted.fieldErrors)
        } finally {
            setLoading(false)
        }
    }

    return (
        <section className='animate'>
            <div className='grid gap-4 md:grid-cols-[1.2fr_0.9fr]'>
                <div className='border border-border bg-surface p-6 md:p-8 rounded-xl'>
                    <h2 className='text-2xl font-semibold text-text'>{t('auth.login')}</h2>

                    {auth.user ? (
                        <div className='mt-4 border border-info/40 bg-info/10 p-3 text-sm text-info'>
                            {t('auth.alreadyLoggedIn', { username: auth.user.username })}{' '}
                            <a className='underline' href='/challenges' onClick={(e) => navigate('/challenges', e)}>
                                {t('auth.goToChallenges')}
                            </a>
                        </div>
                    ) : null}

                    <form
                        className='mt-5 space-y-4'
                        onSubmit={(event) => {
                            event.preventDefault()
                            void submit()
                        }}
                    >
                        <div>
                            <label className='text-xs font-medium text-text-muted' htmlFor='login-email'>
                                {t('auth.emailLabel')}
                            </label>
                            <input
                                id='login-email'
                                className='mt-1 w-full border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                type='email'
                                value={email}
                                onChange={(event) => setEmail(event.target.value)}
                                placeholder={t('auth.emailPlaceholder')}
                                autoComplete='email'
                            />
                            {fieldErrors.email ? <p className='mt-1 text-xs text-danger'>{fieldErrors.email}</p> : null}
                        </div>
                        <div>
                            <label className='text-xs font-medium text-text-muted' htmlFor='login-password'>
                                {t('auth.passwordLabel')}
                            </label>
                            <input
                                id='login-password'
                                className='mt-1 w-full border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                type='password'
                                value={password}
                                onChange={(event) => setPassword(event.target.value)}
                                placeholder={t('auth.passwordPlaceholder')}
                                autoComplete='current-password'
                            />
                            {fieldErrors.password ? <p className='mt-1 text-xs text-danger'>{fieldErrors.password}</p> : null}
                        </div>

                        {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}

                        <button className='w-full border border-accent bg-accent py-2.5 text-sm text-white hover:bg-accent-strong disabled:opacity-60' type='submit' disabled={loading}>
                            {loading ? t('auth.loggingIn') : t('auth.login')}
                        </button>
                    </form>
                </div>

                <div className='border border-border bg-surface p-6 md:p-8 rounded-xl'>
                    <h3 className='text-lg font-semibold text-text'>{t('auth.needHelp')}</h3>
                    <ul className='mt-3 space-y-2 text-sm text-text-muted'>
                        <li>
                            {t('auth.noAccount')}{' '}
                            <a className='text-accent underline' href='/register' onClick={(e) => navigate('/register', e)}>
                                {t('auth.signUpLink')}
                            </a>
                        </li>
                    </ul>
                </div>
            </div>
        </section>
    )
}

export default Login
