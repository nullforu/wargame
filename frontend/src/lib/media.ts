const S3_MEDIA_CDN_BASE = String(import.meta.env.VITE_S3_MEDIA_CDN_BASE_URL ?? '')
    .trim()
    .replace(/\/+$/, '')

export const mediaURL = (key?: string | null) => {
    const normalizedKey = String(key ?? '').replace(/^\/+/, '')
    if (!S3_MEDIA_CDN_BASE || !normalizedKey) return ''
    return `${S3_MEDIA_CDN_BASE}/${normalizedKey}`
}
