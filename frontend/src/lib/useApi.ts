import { useMemo } from 'react'
import { createApi } from './api'
import { useAuth } from './auth'
import { useT } from './i18n'

export const useApi = () => {
    const { getAuth, setAuthTokens, setAuthUser, clearAuth } = useAuth()
    const t = useT()

    return useMemo(
        () =>
            createApi({
                getAuth,
                setAuthTokens,
                setAuthUser,
                clearAuth,
                translate: t,
            }),
        [getAuth, setAuthTokens, setAuthUser, clearAuth, t],
    )
}
