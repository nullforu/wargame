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
            <div className='grid gap-4 md:grid-cols-[1.2fr_0.9fr]'>
                <div className='border border-border bg-surface p-6 md:p-8 rounded-xl'>
                    <h2 className='text-2xl font-semibold text-text'>{t('auth.register')}</h2>

                    <form
                        className='mt-5 space-y-4'
                        onSubmit={(event) => {
                            event.preventDefault()
                            void submit()
                        }}
                    >
                        <div>
                            <label className='text-xs font-medium text-text-muted' htmlFor='register-email'>
                                {t('auth.emailLabel')}
                            </label>
                            <input
                                id='register-email'
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
                            <label className='text-xs font-medium text-text-muted' htmlFor='register-username'>
                                {t('auth.usernameLabel')}
                            </label>
                            <input
                                id='register-username'
                                className='mt-1 w-full border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                type='text'
                                value={username}
                                onChange={(event) => setUsername(event.target.value)}
                                placeholder={t('auth.usernamePlaceholder')}
                                autoComplete='username'
                            />
                            {fieldErrors.username ? <p className='mt-1 text-xs text-danger'>{fieldErrors.username}</p> : null}
                        </div>

                        <div>
                            <label className='text-xs font-medium text-text-muted' htmlFor='register-password'>
                                {t('auth.passwordLabel')}
                            </label>
                            <input
                                id='register-password'
                                className='mt-1 w-full border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                type='password'
                                value={password}
                                onChange={(event) => setPassword(event.target.value)}
                                placeholder={t('auth.passwordPlaceholder')}
                                autoComplete='new-password'
                            />
                            {fieldErrors.password ? <p className='mt-1 text-xs text-danger'>{fieldErrors.password}</p> : null}
                        </div>

                        {errorMessage ? <FormMessage variant='error' message={errorMessage} /> : null}
                        {success ? (
                            <FormMessage variant='success'>
                                {t('auth.accountCreatedPrefix')}{' '}
                                <a className='underline' href='/login' onClick={(e) => navigate('/login', e)}>
                                    {t('auth.loginLink')}
                                </a>{' '}
                                {t('auth.accountCreatedSuffix')}
                            </FormMessage>
                        ) : null}

                        <button className='w-full border border-accent bg-accent py-2.5 text-sm text-white hover:bg-accent-strong disabled:opacity-60' type='submit' disabled={loading}>
                            {loading ? t('auth.creating') : t('auth.createAccount')}
                        </button>
                    </form>
                </div>

                <div className='border border-border bg-surface p-6 md:p-8 rounded-xl'>
                    <h3 className='text-lg font-semibold text-text'>{t('register.noticeTitle')}</h3>
                    <ul className='mt-3 space-y-2 text-sm text-text-muted'>
                        <li>{t('register.noticeRule1')}</li>
                        <li>{t('register.noticeRule2')}</li>
                    </ul>
                </div>
            </div>
        </section>
    )
}

export default Register
