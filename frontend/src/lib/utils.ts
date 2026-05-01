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

export const utf8ByteLength = (value: string) => new TextEncoder().encode(value).length

export const trimToMaxUtf8Bytes = (value: string, maxBytes: number) => {
    if (utf8ByteLength(value) <= maxBytes) return value

    let out = ''
    for (const ch of value) {
        const next = out + ch
        if (utf8ByteLength(next) > maxBytes) break
        out = next
    }
    return out
}

export const parseRouteId = (value?: string) => {
    if (!value) return null
    const parsed = Number.parseInt(value, 10)
    return Number.isNaN(parsed) ? null : parsed
}

export const hashString = (str: string): number => {
    let hash = 0
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i)
        hash = (hash << 5) - hash + char
        hash = hash & hash
    }
    return Math.abs(hash)
}

export const generateColorFromUsername = (username: string): string => {
    const hash = hashString(username)
    const hue = hash % 360
    const saturation = 60 + (hash % 20)
    const lightness = 55 + (hash % 15)
    return `hsl(${hue}, ${saturation}%, ${lightness}%)`
}
