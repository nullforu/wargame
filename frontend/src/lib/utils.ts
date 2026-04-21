import type { ApiErrorDetail } from './api'
import { ApiError } from './api'

export type FieldErrors = Record<string, string>

export const formatApiError = (error: unknown, translate: (key: string, vars?: Record<string, string | number>) => string) => {
    if (error instanceof ApiError) {
        const fieldErrors = buildFieldErrors(error.details)

        if (error.status === 429) {
            const resetSeconds = error.rateLimit?.reset_seconds
            const message = typeof resetSeconds === 'number' ? translate('errors.tooManyRequests', { seconds: resetSeconds }) : translate('errors.tooManyRequestsLater')

            return { message, fieldErrors }
        }

        return { message: error.message, fieldErrors }
    }

    console.log('Unknown error format:', error)
    return { message: translate('errors.network'), fieldErrors: {} }
}

const buildFieldErrors = (details?: ApiErrorDetail[]) => {
    if (!details || details.length === 0) return {} as FieldErrors

    return details.reduce<FieldErrors>((acc, detail) => {
        acc[detail.field] = detail.reason
        return acc
    }, {})
}

export const formatDateTime = (value: string, localeTag: string) => {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value

    return date.toLocaleString(localeTag, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'Asia/Seoul',
    })
}

export const isZipFile = (file: File) => file.name.toLowerCase().endsWith('.zip')

export const parseRouteId = (value?: string) => {
    if (!value) return null
    const parsed = Number.parseInt(value, 10)
    return Number.isNaN(parsed) ? null : parsed
}

/**
 * 문자열을 시드로 일관성 있는 숫자 해시를 생성합니다.
 * 같은 입력은 항상 같은 출력을 생성합니다.
 */
export const hashString = (str: string): number => {
    let hash = 0
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i)
        hash = (hash << 5) - hash + char
        hash = hash & hash // Convert to 32-bit integer
    }
    return Math.abs(hash)
}

/**
 * 유저 이름을 시드로 일관성 있는 HSL 색상을 생성합니다.
 * @param username 유저 이름 (시드로 사용)
 * @returns CSS에서 사용 가능한 hsl() 색상 문자열
 */
export const generateColorFromUsername = (username: string): string => {
    const hash = hashString(username)
    const hue = hash % 360 // 0-360 도 범위의 색상
    const saturation = 60 + (hash % 20) // 60-80% 포화도 (더 선명한 색상)
    const lightness = 55 + (hash % 15) // 55-70% 밝기 (적절한 명도)
    return `hsl(${hue}, ${saturation}%, ${lightness}%)`
}
