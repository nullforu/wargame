import { useState } from 'react'
import { formatApiError, type FieldErrors } from '../lib/utils'
import { navigate } from '../lib/router'
import FormMessage from '../components/FormMessage'
import { useT } from '../lib/i18n'
import { useApi } from '../lib/useApi'

interface RouteProps {
    routeParams?: Record<string, string>
}

const Register = ({ routeParams = {} }: RouteProps) => {
    void routeParams
    const t = useT()
    const api = useApi()
    const [email, setEmail] = useState('')
    const [username, setUsername] = useState('')
    const [password, setPassword] = useState('')
    const [loading, setLoading] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
    const [success, setSuccess] = useState(false)

    const submit = async () => {
        setLoading(true)
        setSuccess(false)
        setErrorMessage('')
        setFieldErrors({})

        try {
            await api.register({ email, username, password })

            setSuccess(true)
            setEmail('')
            setUsername('')
            setPassword('')
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
            <div className='grid gap-8 md:grid-cols-[1.1fr_1fr]'>
                <div className='rounded-3xl border border-border bg-surface p-10'>
                    <h2 className='text-3xl text-text'>{t('auth.register')}</h2>

                    <form
                        className='mt-6 space-y-5'
                        onSubmit={(event) => {
                            event.preventDefault()
                            submit()
                        }}
                    >
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='register-email'>
                                {t('auth.emailLabel')}
                            </label>
                            <input
                                id='register-email'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='email'
                                value={email}
                                onChange={(event) => setEmail(event.target.value)}
                                placeholder={t('auth.emailPlaceholder')}
                                autoComplete='email'
                            />
                            {fieldErrors.email ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('auth.emailLabel')}: {fieldErrors.email}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='register-username'>
                                {t('auth.usernameLabel')}
                            </label>
                            <input
                                id='register-username'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='text'
                                value={username}
                                onChange={(event) => setUsername(event.target.value)}
                                placeholder={t('auth.usernamePlaceholder')}
                                autoComplete='username'
                            />
                            {fieldErrors.username ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('auth.usernameLabel')}: {fieldErrors.username}
                                </p>
                            ) : null}
                        </div>
                        <div>
                            <label className='text-xs uppercase tracking-wide text-text-muted' htmlFor='register-password'>
                                {t('auth.passwordLabel')}
                            </label>
                            <input
                                id='register-password'
                                className='mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none'
                                type='password'
                                value={password}
                                onChange={(event) => setPassword(event.target.value)}
                                placeholder={t('auth.passwordPlaceholder')}
                                autoComplete='new-password'
                            />
                            {fieldErrors.password ? (
                                <p className='mt-2 text-xs text-danger'>
                                    {t('auth.passwordLabel')}: {fieldErrors.password}
                                </p>
                            ) : null}
                        </div>
                        {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
                        {success ? (
                            <FormMessage variant='success'>
                                {t('auth.accountCreatedPrefix')}{' '}
                                <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                                    {t('auth.loginLink')}
                                </a>{' '}
                                {t('auth.accountCreatedSuffix')}
                            </FormMessage>
                        ) : null}

                        <button className='w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60 cursor-pointer' type='submit' disabled={loading}>
                            {loading ? t('auth.creating') : t('auth.createAccount')}
                        </button>
                    </form>
                </div>

                <div className='rounded-3xl border border-border bg-surface p-10'>
                    <h3 className='text-lg text-text'>{t('register.noticeTitle')}</h3>
                    <ul className='mt-4 space-y-3 text-sm text-text'>
                        <li>{t('register.noticeRule1')}</li>
                        <li>{t('register.noticeRule2')}</li>
                    </ul>
                </div>
            </div>
        </section>
    )
}

export default Register
