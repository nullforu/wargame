import { useState } from 'react'
import { navigate } from '../lib/router'
import type { AuthUser } from '../lib/types'
import { useT, type Locale, useLocale, useSetLocale } from '../lib/i18n'
import { useTheme, toggleThemeValue } from '../lib/theme'
import { useAuth } from '../lib/auth'
import { useApi } from '../lib/useApi'
import { SITE_CONFIG } from '../lib/siteConfig'

interface HeaderProps {
    user: AuthUser | null
}

const Header = ({ user }: HeaderProps) => {
    const t = useT()
    const { theme, toggleTheme } = useTheme()
    const locale = useLocale()
    const setLocale = useSetLocale()
    const { clearAuth } = useAuth()
    const api = useApi()
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
    const pathname = window.location.pathname || '/'

    const navItems = [
        { path: '/challenges', label: t('nav.challenges') },
        { path: '/scoreboard', label: t('nav.scoreboard') },
        { path: '/users', label: t('nav.users') },
        { path: '/profile', label: t('nav.profile') },
    ]

    const logout = async (after?: () => void) => {
        try {
            await api.logout()
        } catch {
            clearAuth()
        }
        navigate('/login')
        after?.()
    }

    const navClass = (path: string) => `inline-flex h-10 items-center rounded-md px-3 text-sm transition ${pathname.startsWith(path) ? 'bg-accent/10 text-accent font-semibold' : 'text-text-subtle hover:bg-surface-muted hover:text-text'}`

    const handleLocaleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        setLocale(event.target.value as Locale)
    }

    const themeButtonLabel = theme === 'dark' ? t('header.switchToLight') : t('header.switchToDark')

    return (
        <>
            <header className='bg-surface dark:bg-surface'>
                <div className='mx-auto flex h-14 w-full max-w-7xl items-center justify-between gap-4 px-4 md:px-6'>
                    <div className='flex items-center gap-4 lg:gap-8'>
                        <button className='px-2.5 text-xs text-text-subtle lg:hidden dark:text-text-muted' onClick={() => setMobileMenuOpen((prev) => !prev)}>
                            {mobileMenuOpen ? (
                                <svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <line x1='18' y1='6' x2='6' y2='18'></line>
                                    <line x1='6' y1='6' x2='18' y2='18'></line>
                                </svg>
                            ) : (
                                <svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <line x1='3' y1='12' x2='21' y2='12'></line>
                                    <line x1='3' y1='6' x2='21' y2='6'></line>
                                    <line x1='3' y1='18' x2='21' y2='18'></line>
                                </svg>
                            )}
                        </button>

                        <a href='/' className='flex items-center gap-2' onClick={(event) => navigate('/', event)}>
                            <img src={`/logo_${toggleThemeValue(theme)}_cropped.svg`} alt={t('header.logoAlt')} className='h-6 w-auto' />
                        </a>

                        <nav className='hidden items-center gap-2 lg:flex'>
                            {navItems.map((item) => (
                                <a key={item.path} href={item.path} className={navClass(item.path)} onClick={(event) => navigate(item.path, event)}>
                                    {item.label}
                                </a>
                            ))}
                            {user?.role === 'admin' ? (
                                <a href='/admin' className={navClass('/admin')} onClick={(event) => navigate('/admin', event)}>
                                    {t('nav.admin')}
                                </a>
                            ) : null}
                        </nav>
                    </div>

                    <div className='hidden items-center gap-2 lg:flex'>
                        <button
                            className='bg-surface-muted px-2 py-1 text-xs text-text-muted transition hover:bg-surface-subtle dark:text-text-muted dark:hover:bg-surface-muted'
                            onClick={toggleTheme}
                            title={themeButtonLabel}
                            aria-label={themeButtonLabel}
                        >
                            {theme === 'dark' ? (
                                <svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <circle cx='12' cy='12' r='4'></circle>
                                    <path d='M12 2v2'></path>
                                    <path d='M12 20v2'></path>
                                    <path d='m4.93 4.93 1.41 1.41'></path>
                                    <path d='m17.66 17.66 1.41 1.41'></path>
                                    <path d='M2 12h2'></path>
                                    <path d='M20 12h2'></path>
                                    <path d='m6.34 17.66-1.41 1.41'></path>
                                    <path d='m19.07 4.93-1.41 1.41'></path>
                                </svg>
                            ) : (
                                <svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <path d='M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z'></path>
                                </svg>
                            )}
                        </button>
                        <select className='bg-surface-muted px-2 py-1 text-xs text-text-muted dark:bg-surface-muted dark:text-text' value={locale} onChange={handleLocaleChange} aria-label={t('header.language')}>
                            <option value='en'>{t('header.languageEnglish')}</option>
                            <option value='ko'>{t('header.languageKorean')}</option>
                            <option value='ja'>{t('header.languageJapanese')}</option>
                        </select>

                        {user ? (
                            <>
                                <button className='bg-surface-muted px-3 py-1 text-xs text-text-muted transition hover:bg-surface-subtle dark:text-text dark:hover:bg-surface-muted rounded-md' onClick={() => navigate('/profile')}>
                                    {user.username}
                                </button>
                                <button className='bg-accent px-3 py-1 text-xs text-white transition hover:bg-accent-strong rounded-md' onClick={() => logout()}>
                                    {t('auth.logout')}
                                </button>
                            </>
                        ) : (
                            <>
                                <a
                                    href='/login'
                                    className='bg-surface-muted px-3 py-1 text-xs text-text-muted transition hover:bg-surface-subtle dark:text-text dark:hover:bg-surface-muted rounded-md'
                                    onClick={(event) => navigate('/login', event)}
                                >
                                    {t('auth.login')}
                                </a>
                                <a href='/register' className='bg-accent px-3 py-1 text-xs text-white transition hover:bg-accent-strong rounded-md' onClick={(event) => navigate('/register', event)}>
                                    {t('auth.register')}
                                </a>
                            </>
                        )}
                    </div>
                </div>
            </header>

            {mobileMenuOpen ? <button className='fixed inset-0 z-40 bg-black/25 lg:hidden' onClick={() => setMobileMenuOpen(false)} aria-label={t('header.closeMenu')}></button> : null}

            <aside className={`fixed left-0 top-0 z-50 h-full w-[82vw] max-w-xs transform bg-surface shadow-xl transition-transform duration-200 lg:hidden dark:bg-surface ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                <div className='px-4 py-3 text-base font-semibold text-text dark:text-text'>{SITE_CONFIG.headerTitle}</div>
                <nav className='p-2'>
                    {navItems.map((item) => (
                        <a
                            key={item.path}
                            href={item.path}
                            className={`block rounded-md px-3 py-2 text-sm transition ${pathname.startsWith(item.path) ? 'bg-accent/10 text-accent' : 'text-text-muted hover:bg-surface-muted dark:text-text'}`}
                            onClick={(event) => {
                                navigate(item.path, event)
                                setMobileMenuOpen(false)
                            }}
                        >
                            {item.label}
                        </a>
                    ))}
                    {user?.role === 'admin' ? (
                        <a
                            href='/admin'
                            className={`block rounded-md px-3 py-2 text-sm transition ${pathname.startsWith('/admin') ? 'bg-accent/10 text-accent' : 'text-text-muted hover:bg-surface-muted dark:text-text'}`}
                            onClick={(event) => {
                                navigate('/admin', event)
                                setMobileMenuOpen(false)
                            }}
                        >
                            {t('nav.admin')}
                        </a>
                    ) : null}
                </nav>

                <div className='mt-3 p-3'>
                    <div className='flex items-center gap-2'>
                        <select className='flex-1 bg-surface-muted px-2 py-1 text-xs text-text-muted dark:bg-surface-muted dark:text-text' value={locale} onChange={handleLocaleChange} aria-label={t('header.language')}>
                            <option value='en'>{t('header.languageEnglish')}</option>
                            <option value='ko'>{t('header.languageKorean')}</option>
                            <option value='ja'>{t('header.languageJapanese')}</option>
                        </select>
                        <button className='bg-surface-muted px-2 py-1 text-xs text-text-muted dark:bg-surface-muted dark:text-text-muted' onClick={toggleTheme} title={themeButtonLabel} aria-label={themeButtonLabel}>
                            {theme === 'dark' ? (
                                <svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <circle cx='12' cy='12' r='4'></circle>
                                    <path d='M12 2v2'></path>
                                    <path d='M12 20v2'></path>
                                    <path d='m4.93 4.93 1.41 1.41'></path>
                                    <path d='m17.66 17.66 1.41 1.41'></path>
                                    <path d='M2 12h2'></path>
                                    <path d='M20 12h2'></path>
                                    <path d='m6.34 17.66-1.41 1.41'></path>
                                    <path d='m19.07 4.93-1.41 1.41'></path>
                                </svg>
                            ) : (
                                <svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
                                    <path d='M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z'></path>
                                </svg>
                            )}
                        </button>
                    </div>
                    <div className='mt-2 space-y-2'>
                        {user ? (
                            <button
                                className='w-full bg-accent px-3 py-1.5 text-xs text-white'
                                onClick={() => {
                                    void logout(() => setMobileMenuOpen(false))
                                }}
                            >
                                {t('auth.logout')}
                            </button>
                        ) : (
                            <a
                                href='/login'
                                className='block w-full bg-surface-muted px-3 py-1.5 text-center text-xs text-text-muted'
                                onClick={(event) => {
                                    navigate('/login', event)
                                    setMobileMenuOpen(false)
                                }}
                            >
                                {t('auth.login')}
                            </a>
                        )}
                    </div>
                </div>
            </aside>
        </>
    )
}

export default Header
