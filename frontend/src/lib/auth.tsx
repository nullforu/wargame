import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { AuthUser } from './types'

export interface AuthState {
    user: AuthUser | null
}

interface AuthContextValue {
    state: AuthState
    getAuth: () => AuthState
    setAuthUser: (user: AuthUser | null) => void
    clearAuth: () => void
}

const emptyAuth = (): AuthState => ({ user: null })

const AuthContext = createContext<AuthContextValue | null>(null)

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
    const [state, setState] = useState<AuthState>(emptyAuth)
    const stateRef = useRef(state)

    useEffect(() => {
        stateRef.current = state
    }, [state])

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
            setAuthUser,
            clearAuth,
        }),
        [state, setAuthUser, clearAuth],
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
