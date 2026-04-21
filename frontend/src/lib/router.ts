export const navigate = (path: string, event?: { preventDefault: () => void }) => {
    if (typeof window === 'undefined') return

    event?.preventDefault()

    const target = new URL(path.startsWith('/') ? path : `/${path}`, window.location.origin)
    const next = `${target.pathname}${target.search}${target.hash}`
    const current = `${window.location.pathname}${window.location.search}${window.location.hash}`
    if (current === next) return

    window.history.pushState({}, '', next)
    window.dispatchEvent(new PopStateEvent('popstate'))
}
