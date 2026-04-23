export const UNKNOWN_LEVEL = 0
export const MIN_LEVEL = 1
export const MAX_LEVEL = 10

export const LEVEL_VOTE_OPTIONS = Array.from({ length: MAX_LEVEL }, (_, idx) => idx + 1)

export const normalizeLevel = (level?: number | null): number => {
    if (typeof level !== 'number' || !Number.isFinite(level)) return UNKNOWN_LEVEL
    const rounded = Math.round(level)
    if (rounded < MIN_LEVEL || rounded > MAX_LEVEL) return UNKNOWN_LEVEL
    return rounded
}

export const levelBadgeClass = (level?: number | null) => {
    const normalized = normalizeLevel(level)
    if (normalized === UNKNOWN_LEVEL) return 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-100'
    if (normalized <= 2) return 'bg-emerald-200 text-emerald-900 dark:bg-emerald-700 dark:text-emerald-100'
    if (normalized <= 4) return 'bg-teal-200 text-teal-900 dark:bg-teal-700 dark:text-teal-100'
    if (normalized <= 6) return 'bg-cyan-200 text-cyan-900 dark:bg-cyan-700 dark:text-cyan-100'
    if (normalized <= 8) return 'bg-blue-300 text-blue-900 dark:bg-blue-700 dark:text-blue-100'
    return 'bg-rose-300 text-rose-900 dark:bg-rose-700 dark:text-rose-100'
}

export const levelBarClass = (level?: number | null) => {
    const normalized = normalizeLevel(level)
    if (normalized === UNKNOWN_LEVEL) return 'bg-slate-300 dark:bg-slate-500'
    if (normalized <= 2) return 'bg-emerald-400 dark:bg-emerald-500'
    if (normalized <= 4) return 'bg-teal-400 dark:bg-teal-500'
    if (normalized <= 6) return 'bg-cyan-400 dark:bg-cyan-500'
    if (normalized <= 8) return 'bg-blue-500 dark:bg-blue-500'
    return 'bg-rose-500 dark:bg-rose-500'
}
