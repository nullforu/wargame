import { useMemo } from 'react'
import { createApi } from './api'
import { useAuth } from './auth'
import { useT } from './i18n'

export const useApi = () => {
    const { setAuthUser, clearAuth } = useAuth()
    const t = useT()

    return useMemo(
        () =>
            createApi({
                setAuthUser,
                clearAuth,
                translate: t,
            }),
        [setAuthUser, clearAuth, t],
    )
}
