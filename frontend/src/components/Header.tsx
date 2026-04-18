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
    const isBlocked = user?.role === 'blocked'

    const toggleMobileMenu = () => setMobileMenuOpen((prev) => !prev)
    const closeMobileMenu = () => setMobileMenuOpen(false)

    const navigateAndClose = (path: string, event: React.MouseEvent) => {
        navigate(path, event.nativeEvent)
        closeMobileMenu()
    }

    const handleLocaleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        setLocale(event.target.value as Locale)
    }

    const logout = async (after?: () => void) => {
        try {
            await api.logout()
        } catch {
            clearAuth()
        }
        navigate('/login')
        after?.()
    }

    return (
        <>
            {/* <div className='border-b border-danger/30 bg-danger/10'>
                <div className='mx-auto max-w-6xl px-6 py-3 text-sm text-danger'>
                    <p className='font-medium'>테스트 서버 운영 중이 아닙니다.</p>
                    <p className='text-xs text-danger/80'>
                        프론트엔드 UI를 둘러볼 수는 있지만 로그인이나 문제 풀이 등은 불가능합니다.
                    </p>
                </div>
            </div> */}
            {isBlocked ? (
                <div className='border-b border-danger/30 bg-danger/10'>
                    <div className='mx-auto max-w-6xl px-6 py-3 text-sm text-danger'>
                        <p className='font-medium'>{t('blocked.bannerTitle')}</p>
                        <p className='text-xs text-danger/80'>{t('blocked.bannerBody')}</p>
                        {user?.blocked_reason ? (
                            <p className='mt-1 text-xs text-danger/80'>
                                {t('blocked.reasonLabel')}: {user.blocked_reason}
                            </p>
                        ) : null}
                    </div>
                </div>
            ) : null}
            <header className='border-b border-border bg-surface/70 backdrop-blur'>
                <div className='mx-auto flex max-w-6xl items-center justify-between px-6 py-4'>
                    <button className='flex items-center justify-center p-2 text-text lg:hidden cursor-pointer' onClick={toggleMobileMenu} aria-label={t('header.toggleMobileMenu')}>
                        <svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                            {mobileMenuOpen ? (
                                <>
                                    <line x1='18' y1='6' x2='6' y2='18' />
                                    <line x1='6' y1='6' x2='18' y2='18' />
                                </>
                            ) : (
                                <>
                                    <line x1='3' y1='12' x2='21' y2='12' />
                                    <line x1='3' y1='6' x2='21' y2='6' />
                                    <line x1='3' y1='18' x2='21' y2='18' />
                                </>
                            )}
                        </svg>
                    </button>

                    <a href='/' className='hidden items-center gap-4 lg:flex cursor-pointer' onClick={(event) => navigate('/', event)}>
                        <img src={`/logo_${toggleThemeValue(theme)}_cropped.svg`} alt={t('header.logoAlt')} className='h-6 w-auto' />
                        <div>
                            <p className='font-display text-xl text-text'>{SITE_CONFIG.headerTitle}</p>
                            <p className='text-xs text-text-muted'>{SITE_CONFIG.headerDescription}</p>
                        </div>
                    </a>

                    <nav className='hidden items-center gap-6 text-sm text-text lg:flex'>
                        <a className='hover:text-accent cursor-pointer' href='/challenges' onClick={(e) => navigate('/challenges', e)}>
                            {t('nav.challenges')}
                        </a>
                        <a className='hover:text-accent cursor-pointer' href='/scoreboard' onClick={(e) => navigate('/scoreboard', e)}>
                            {t('nav.scoreboard')}
                        </a>
                        <a className='hover:text-accent cursor-pointer' href='/users' onClick={(e) => navigate('/users', e)}>
                            {t('nav.users')}
                        </a>
                        <a className='hover:text-accent cursor-pointer' href='/profile' onClick={(e) => navigate('/profile', e)}>
                            {t('nav.profile')}
                        </a>
                        {user?.role === 'admin' ? (
                            <a className='hover:text-accent cursor-pointer' href='/admin' onClick={(e) => navigate('/admin', e)}>
                                {t('nav.admin')}
                            </a>
                        ) : null}
                    </nav>

                    <div className='hidden items-center gap-3 lg:flex'>
                        {user ? (
                            <>
                                <button className='hidden text-right text-xs text-text-muted sm:block cursor-pointer' onClick={() => navigate('/profile')}>
                                    <p className='text-text'>{user.username}</p>
                                    <p>{user.email}</p>
                                </button>
                                <button className='rounded-full border border-border px-4 py-2 text-xs text-text transition hover:border-accent hover:text-accent cursor-pointer' onClick={() => logout()}>
                                    {t('auth.logout')}
                                </button>
                            </>
                        ) : (
                            <>
                                <a href='/login' className='rounded-full border border-border px-4 py-2 text-xs text-text transition hover:border-accent hover:text-accent cursor-pointer' onClick={(e) => navigate('/login', e)}>
                                    {t('auth.login')}
                                </a>
                                <a href='/register' className='rounded-full bg-accent/20 px-4 py-2 text-xs text-accent-strong transition hover:bg-accent/30 cursor-pointer' onClick={(e) => navigate('/register', e)}>
                                    {t('auth.register')}
                                </a>
                            </>
                        )}
                        <button
                            className='rounded-full border border-border p-2 text-text transition hover:border-accent hover:text-accent cursor-pointer'
                            onClick={toggleTheme}
                            aria-label={t('header.toggleTheme')}
                            title={theme === 'light' ? t('header.switchToDark') : t('header.switchToLight')}
                        >
                            {theme === 'light' ? (
                                <svg xmlns='http://www.w3.org/2000/svg' width='18' height='18' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                                    <path d='M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z' />
                                </svg>
                            ) : (
                                <svg xmlns='http://www.w3.org/2000/svg' width='18' height='18' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                                    <circle cx='12' cy='12' r='4' />
                                    <path d='M12 2v2' />
                                    <path d='M12 20v2' />
                                    <path d='m4.93 4.93 1.41 1.41' />
                                    <path d='m17.66 17.66 1.41 1.41' />
                                    <path d='M2 12h2' />
                                    <path d='M20 12h2' />
                                    <path d='m6.34 17.66-1.41 1.41' />
                                    <path d='m19.07 4.93-1.41 1.41' />
                                </svg>
                            )}
                        </button>
                        <div className='flex items-center gap-2 rounded-full border border-border px-3 py-1.5 text-xs text-text'>
                            <select className='bg-transparent text-xs text-text focus:outline-none cursor-pointer' value={locale} onChange={handleLocaleChange} aria-label={t('header.language')}>
                                <option value='en'>{t('header.languageEnglish')}</option>
                                <option value='ko'>{t('header.languageKorean')}</option>
                                <option value='ja'>{t('header.languageJapanese')}</option>
                            </select>
                        </div>
                    </div>
                </div>
            </header>

            {mobileMenuOpen ? <button className='fixed inset-0 z-40 bg-overlay/50 backdrop-blur-sm lg:hidden cursor-pointer' onClick={closeMobileMenu} aria-label={t('header.closeMenu')}></button> : null}

            <aside className={`fixed left-0 top-0 z-50 h-full w-72 transform border-r border-border bg-surface shadow-xl transition-transform duration-300 lg:hidden ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                <div className='flex h-full flex-col'>
                    <div className='flex items-center justify-between border-b border-border p-6'>
                        <div className='flex items-center gap-3'>
                            <img src={`/logo_${toggleThemeValue(theme)}_cropped.svg`} alt={t('header.logoAlt')} className='h-4 w-auto' />
                            <div>
                                <p className='font-display text-xl text-text'>{SITE_CONFIG.headerTitle}</p>
                                <p className='text-xs text-text-muted'>{SITE_CONFIG.headerDescription}</p>
                            </div>
                        </div>
                        <button className='p-1 text-text cursor-pointer' onClick={closeMobileMenu} aria-label={t('header.closeMenu')}>
                            <svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                                <line x1='18' y1='6' x2='6' y2='18' />
                                <line x1='6' y1='6' x2='18' y2='18' />
                            </svg>
                        </button>
                    </div>

                    <div className='flex flex-1 flex-col overflow-y-auto p-6'>
                        {user ? (
                            <div className='mb-6 rounded-lg border border-border bg-surface-muted p-4'>
                                <p className='text-sm font-medium text-text'>{user.username}</p>
                                <p className='text-xs text-text-muted'>{user.email}</p>
                                {user.role === 'admin' ? <span className='mt-2 inline-block rounded-full bg-accent/20 px-2 py-0.5 text-xs text-accent-strong'>{t('common.admin')}</span> : null}
                            </div>
                        ) : null}

                        <nav className='flex flex-col gap-2'>
                            <a href='/challenges' className='rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent cursor-pointer' onClick={(e) => navigateAndClose('/challenges', e)}>
                                {t('nav.challenges')}
                            </a>
                            <a href='/scoreboard' className='rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent cursor-pointer' onClick={(e) => navigateAndClose('/scoreboard', e)}>
                                {t('nav.scoreboard')}
                            </a>
                            <a href='/users' className='rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent cursor-pointer' onClick={(e) => navigateAndClose('/users', e)}>
                                {t('nav.users')}
                            </a>
                            <a href='/profile' className='rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent cursor-pointer' onClick={(e) => navigateAndClose('/profile', e)}>
                                {t('nav.profile')}
                            </a>
                            {user?.role === 'admin' ? (
                                <a href='/admin' className='rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent cursor-pointer' onClick={(e) => navigateAndClose('/admin', e)}>
                                    {t('nav.admin')}
                                </a>
                            ) : null}
                        </nav>

                        <div className='my-6 border-t border-border'></div>

                        <div className='flex flex-col gap-3'>
                            <div className='rounded-lg border border-border px-4 py-3 text-sm text-text'>
                                <div className='flex items-center justify-between gap-3'>
                                    <span className='text-text-muted'>{t('header.language')}</span>
                                    <select className='bg-transparent text-sm text-text focus:outline-none' value={locale} onChange={handleLocaleChange} aria-label={t('header.language')}>
                                        <option value='en'>{t('header.languageEnglish')}</option>
                                        <option value='ko'>{t('header.languageKorean')}</option>
                                        <option value='ja'>{t('header.languageJapanese')}</option>
                                    </select>
                                </div>
                            </div>
                            <button className='flex items-center justify-between rounded-lg border border-border px-4 py-3 text-sm text-text transition hover:border-accent hover:text-accent cursor-pointer' onClick={toggleTheme}>
                                <span>{theme === 'light' ? t('header.switchToDark') : t('header.switchToLight')}</span>
                                {theme === 'light' ? (
                                    <svg xmlns='http://www.w3.org/2000/svg' width='18' height='18' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                                        <path d='M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z' />
                                    </svg>
                                ) : (
                                    <svg xmlns='http://www.w3.org/2000/svg' width='18' height='18' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                                        <circle cx='12' cy='12' r='4' />
                                        <path d='M12 2v2' />
                                        <path d='M12 20v2' />
                                        <path d='m4.93 4.93 1.41 1.41' />
                                        <path d='m17.66 17.66 1.41 1.41' />
                                        <path d='M2 12h2' />
                                        <path d='M20 12h2' />
                                        <path d='m6.34 17.66-1.41 1.41' />
                                        <path d='m19.07 4.93-1.41 1.41' />
                                    </svg>
                                )}
                            </button>

                            {user ? (
                                <button className='rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-danger transition hover:border-danger/50 hover:bg-danger/20 cursor-pointer' onClick={() => logout(closeMobileMenu)}>
                                    {t('auth.logout')}
                                </button>
                            ) : (
                                <>
                                    <a
                                        href='/login'
                                        className='rounded-lg border border-border px-4 py-3 text-center text-sm text-text transition hover:border-accent hover:text-accent cursor-pointer'
                                        onClick={(e) => navigateAndClose('/login', e)}
                                    >
                                        {t('auth.login')}
                                    </a>
                                    <a href='/register' className='rounded-lg bg-accent/20 px-4 py-3 text-center text-sm text-accent-strong transition hover:bg-accent/30 cursor-pointer' onClick={(e) => navigateAndClose('/register', e)}>
                                        {t('auth.register')}
                                    </a>
                                </>
                            )}
                        </div>
                    </div>
                </div>
            </aside>
        </>
    )
}

export default Header
