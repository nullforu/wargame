import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import en from '../locales/en.json'
import ko from '../locales/ko.json'
import ja from '../locales/ja.json'

export type Locale = 'en' | 'ko' | 'ja'

interface LocaleContextValue {
    locale: Locale
    setLocale: (locale: Locale) => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const STORAGE_KEY = 'wargame.locale'

const dictionaries: Record<Locale, Record<string, string>> = {
    en,
    ko,
    ja,
}

const normalizeLocale = (value?: string | null): Locale => {
    switch (value) {
        case 'ko':
            return 'ko'
        case 'ja':
            return 'ja'
        case 'en':
        default:
            return 'en'
    }
}

const loadLocale = (): Locale => {
    if (typeof localStorage === 'undefined') return 'en'
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) return normalizeLocale(saved)

    return 'en'
}

const persistLocale = (locale: Locale) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, locale)
    }
}

const interpolate = (message: string, vars?: Record<string, string | number>) => {
    if (!vars) return message
    return message.replace(/\{(\w+)\}/g, (_, key: string) => {
        const value = vars[key]
        return value === undefined || value === null ? '' : String(value)
    })
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

export const LocaleProvider = ({ children }: { children: React.ReactNode }) => {
    const [locale, setLocaleState] = useState<Locale>(() => loadLocale())

    const setLocale = useCallback((value: Locale) => {
        const normalized = normalizeLocale(value)
        setLocaleState(normalized)
        persistLocale(normalized)
    }, [])

    const t = useCallback(
        (key: string, vars?: Record<string, string | number>) => {
            const dictionary = dictionaries[locale] ?? dictionaries.en
            const fallback = dictionaries.en
            const message = dictionary[key] ?? fallback[key] ?? key
            return interpolate(message, vars)
        },
        [locale],
    )

    const value = useMemo(() => ({ locale, setLocale, t }), [locale, setLocale, t])

    return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}

export const useLocale = () => {
    const context = useContext(LocaleContext)
    if (!context) {
        throw new Error('useLocale must be used within LocaleProvider')
    }
    return context.locale
}

export const useSetLocale = () => {
    const context = useContext(LocaleContext)
    if (!context) {
        throw new Error('useSetLocale must be used within LocaleProvider')
    }
    return context.setLocale
}

export const useT = () => {
    const context = useContext(LocaleContext)
    if (!context) {
        throw new Error('useT must be used within LocaleProvider')
    }
    return context.t
}

export const getLocaleTag = (locale: Locale) => {
    switch (locale) {
        case 'ko':
            return 'ko-KR'
        case 'ja':
            return 'ja-JP'
        case 'en':
        default:
            return 'en-US'
    }
}

const categoryKeyMap: Record<string, string> = {
    Web: 'categories.web',
    Web3: 'categories.web3',
    Pwnable: 'categories.pwnable',
    Reversing: 'categories.reversing',
    Crypto: 'categories.crypto',
    Forensics: 'categories.forensics',
    Network: 'categories.network',
    Cloud: 'categories.cloud',
    Misc: 'categories.misc',
    Programming: 'categories.programming',
    Algorithms: 'categories.algorithms',
    Math: 'categories.math',
    AI: 'categories.ai',
    Blockchain: 'categories.blockchain',
}

export const getCategoryKey = (category: string) => categoryKeyMap[category] ?? category

export const translateCategory = (translate: (key: string, vars?: Record<string, string | number>) => string, category: string) => {
    return translate(getCategoryKey(category))
}

const roleKeyMap: Record<string, string> = {
    admin: 'roles.admin',
    user: 'roles.user',
    blocked: 'roles.blocked',
}

export const getRoleKey = (role: string) => roleKeyMap[role] ?? role

export const translateRole = (translate: (key: string, vars?: Record<string, string | number>) => string, role: string) => {
    return translate(getRoleKey(role))
}
