import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { AuthUser } from './types'

export interface AuthState {
    accessToken: string | null
    refreshToken: string | null
    user: AuthUser | null
}

interface AuthContextValue {
    state: AuthState
    getAuth: () => AuthState
    setAuthTokens: (accessToken: string, refreshToken: string) => void
    setAuthUser: (user: AuthUser | null) => void
    clearAuth: () => void
}

const STORAGE_KEY = 'wargame.auth'

const emptyAuth = (): AuthState => ({ accessToken: null, refreshToken: null, user: null })

const loadAuth = (): AuthState => {
    if (typeof localStorage === 'undefined') return emptyAuth()

    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (!raw) return emptyAuth()
        return JSON.parse(raw) as AuthState
    } catch {
        return emptyAuth()
    }
}

const persistAuth = (state: AuthState) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    }
}

const AuthContext = createContext<AuthContextValue | null>(null)

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
    const [state, setState] = useState<AuthState>(() => loadAuth())
    const stateRef = useRef(state)

    useEffect(() => {
        stateRef.current = state
        persistAuth(state)
    }, [state])

    const setAuthTokens = useCallback((accessToken: string, refreshToken: string) => {
        setState((prev) => ({ ...prev, accessToken, refreshToken }))
    }, [])

    const setAuthUser = useCallback((user: AuthUser | null) => {
        setState((prev) => ({ ...prev, user }))
    }, [])

    const clearAuth = useCallback(() => {
        setState(emptyAuth())
    }, [])

    const value = useMemo<AuthContextValue>(
        () => ({
            state,
            getAuth: () => stateRef.current,
            setAuthTokens,
            setAuthUser,
            clearAuth,
        }),
        [state, setAuthTokens, setAuthUser, clearAuth],
    )

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export const useAuth = () => {
    const context = useContext(AuthContext)
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider')
    }
    return context
}
