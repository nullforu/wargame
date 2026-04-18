import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

type Theme = 'light' | 'dark'

interface ThemeContextValue {
    theme: Theme
    setTheme: (theme: Theme) => void
    toggleTheme: () => void
}

const THEME_KEY = 'wargame.theme'

const loadTheme = (): Theme => {
    if (typeof localStorage === 'undefined') return 'light'

    try {
        const saved = localStorage.getItem(THEME_KEY)
        return saved === 'dark' ? 'dark' : 'light'
    } catch {
        return 'light'
    }
}

const persistTheme = (theme: Theme) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(THEME_KEY, theme)
    }
}

export const toggleThemeValue = (theme: Theme): Theme => (theme === 'light' ? 'dark' : 'light')

const ThemeContext = createContext<ThemeContextValue | null>(null)

export const ThemeProvider = ({ children }: { children: React.ReactNode }) => {
    const [theme, setThemeState] = useState<Theme>(() => loadTheme())

    useEffect(() => {
        persistTheme(theme)
    }, [theme])

    const setTheme = useCallback((value: Theme) => {
        setThemeState(value)
    }, [])

    const toggleTheme = useCallback(() => {
        setThemeState((prev) => toggleThemeValue(prev))
    }, [])

    const value = useMemo(() => ({ theme, setTheme, toggleTheme }), [theme, setTheme, toggleTheme])

    return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export const useTheme = () => {
    const context = useContext(ThemeContext)
    if (!context) {
        throw new Error('useTheme must be used within ThemeProvider')
    }
    return context
}
