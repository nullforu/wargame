import { useCallback, useEffect, useMemo, useState } from 'react'
import useEmblaCarousel from 'embla-carousel-react'
import { useApi } from '../lib/useApi'
import { useT } from '../lib/i18n'
import { formatApiError } from '../lib/utils'
import { mediaURL } from '../lib/media'
import type { Popup } from '../lib/types'

const HIDDEN_UNTIL_KEY = 'popups_hidden_until'
const DAY_MS = 24 * 60 * 60 * 1000
const AUTO_ADVANCE_MS = 10000
const POPUP_ENTER_DELAY_MS = 250
const POPUP_ENTER_TRANSITION_MS = 260

const isHiddenToday = () => {
    const hiddenUntil = Number(window.localStorage.getItem(HIDDEN_UNTIL_KEY) ?? '0')
    return Number.isFinite(hiddenUntil) && hiddenUntil > Date.now()
}

const PopupCarousel = () => {
    const api = useApi()
    const t = useT()
    const [popups, setPopups] = useState<Popup[]>([])
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [visible, setVisible] = useState(false)
    const [entered, setEntered] = useState(false)
    const [errorMessage, setErrorMessage] = useState('')
    const [autoAdvanceNonce, setAutoAdvanceNonce] = useState(0)
    const [emblaRef, emblaApi] = useEmblaCarousel({ align: 'start', containScroll: 'trimSnaps', dragFree: false, slidesToScroll: 1 })
    const [canScrollPrev, setCanScrollPrev] = useState(false)
    const [canScrollNext, setCanScrollNext] = useState(false)

    useEffect(() => {
        let mounted = true
        let showTimer: number | null = null

        const load = async () => {
            if (isHiddenToday()) return

            try {
                const data = await api.activePopups()
                if (!mounted) return

                const rows = data.popups.filter((popup) => mediaURL(popup.image_key))
                setPopups(rows)
                setSelectedIndex(0)
                if (rows.length > 0) {
                    showTimer = window.setTimeout(() => {
                        if (!mounted) return
                        setVisible(true)
                        window.requestAnimationFrame(() => {
                            window.requestAnimationFrame(() => {
                                if (mounted) setEntered(true)
                            })
                        })
                    }, POPUP_ENTER_DELAY_MS)
                }
            } catch (error) {
                if (mounted) setErrorMessage(formatApiError(error, t).message)
            }
        }

        void load()

        return () => {
            mounted = false
            if (showTimer !== null) window.clearTimeout(showTimer)
        }
    }, [api, t])

    const updateEmblaButtons = useCallback(() => {
        if (!emblaApi) return
        setCanScrollPrev(emblaApi.canScrollPrev())
        setCanScrollNext(emblaApi.canScrollNext())
        setSelectedIndex(emblaApi.selectedScrollSnap())
    }, [emblaApi])

    useEffect(() => {
        if (!emblaApi || !visible) return
        updateEmblaButtons()
        emblaApi.on('select', updateEmblaButtons)
        emblaApi.on('reInit', updateEmblaButtons)
        return () => {
            emblaApi.off('select', updateEmblaButtons)
            emblaApi.off('reInit', updateEmblaButtons)
        }
    }, [emblaApi, updateEmblaButtons, visible])

    useEffect(() => {
        if (!visible || !emblaApi || popups.length <= 1) return

        const timer = window.setInterval(() => {
            if (emblaApi.canScrollNext()) {
                emblaApi.scrollNext()
                return
            }
            emblaApi.scrollTo(0)
        }, AUTO_ADVANCE_MS)

        return () => window.clearInterval(timer)
    }, [autoAdvanceNonce, emblaApi, popups.length, visible])

    const current = popups[selectedIndex]
    const imageURL = useMemo(() => mediaURL(current?.image_key), [current?.image_key])

    if (!visible || !current || !imageURL) return null

    const resetAutoAdvance = () => setAutoAdvanceNonce((prev) => prev + 1)
    const previous = () => {
        resetAutoAdvance()
        emblaApi?.scrollPrev()
    }
    const next = () => {
        resetAutoAdvance()
        emblaApi?.scrollNext()
    }
    const goTo = (popupIndex: number) => {
        resetAutoAdvance()
        emblaApi?.scrollTo(popupIndex)
    }
    const close = () => {
        setEntered(false)
        window.setTimeout(() => setVisible(false), POPUP_ENTER_TRANSITION_MS)
    }
    const hideForDay = () => {
        window.localStorage.setItem(HIDDEN_UNTIL_KEY, String(Date.now() + DAY_MS))
        close()
    }

    return (
        <div className={`fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4 py-6 transition-opacity duration-300 ease-out ${entered ? 'opacity-100' : 'opacity-0'}`}>
            <section
                className={`relative w-full max-w-[min(92vw,520px)] overflow-hidden rounded-lg border border-border bg-surface shadow-2xl transition-all duration-300 ease-out ${entered ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-3 scale-[0.985] opacity-0'}`}
            >
                <button type='button' className='absolute right-3 top-3 z-20 flex h-6 w-6 items-center justify-center rounded-full bg-black/20 text-xs text-white transition hover:bg-black/30' onClick={close} aria-label={t('common.close')}>
                    &times;
                </button>

                <div className='relative bg-white' style={{ aspectRatio: '210 / 297' }}>
                    {popups.length > 1 ? (
                        <button
                            type='button'
                            className='absolute left-3 top-1/2 z-10 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-black/30 text-md text-white shadow transition hover:bg-black/40 disabled:opacity-20'
                            onClick={previous}
                            disabled={!canScrollPrev}
                            aria-label={t('popup.previous')}
                        >
                            {'<'}
                        </button>
                    ) : null}

                    <div className='h-full w-full overflow-hidden' ref={emblaRef}>
                        <div className='flex h-full'>
                            {popups.map((popup) => (
                                <div key={popup.id} className='h-full min-w-full bg-white'>
                                    {popup.link_url ? (
                                        <a className='block h-full w-full' href={popup.link_url} target='_blank' rel='noopener noreferrer'>
                                            <img className='h-full w-full object-contain' src={mediaURL(popup.image_key)} alt={popup.title} />
                                        </a>
                                    ) : (
                                        <img className='h-full w-full object-contain' src={mediaURL(popup.image_key)} alt={popup.title} />
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>

                    {popups.length > 1 ? (
                        <button
                            type='button'
                            className='absolute right-3 top-1/2 z-10 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-black/30 text-md text-white shadow transition hover:bg-black/40 disabled:opacity-20'
                            onClick={next}
                            disabled={!canScrollNext}
                            aria-label={t('popup.next')}
                        >
                            {'>'}
                        </button>
                    ) : null}
                </div>

                <div className='grid grid-cols-[1fr_auto_1fr] items-center gap-3 bg-surface px-4 py-3'>
                    <button type='button' className='min-w-0 justify-self-start text-xs text-text-muted underline-offset-4 hover:text-text hover:underline' onClick={hideForDay}>
                        {t('popup.hideForDay')}
                    </button>

                    {popups.length > 1 ? (
                        <div className='justify-self-center flex gap-1.5'>
                            {popups.map((popup, popupIndex) => (
                                <button
                                    key={popup.id}
                                    type='button'
                                    className={`h-2 w-2 rounded-full transition ${popupIndex === selectedIndex ? 'bg-accent' : 'bg-border hover:bg-text-subtle'}`}
                                    onClick={() => goTo(popupIndex)}
                                    aria-label={t('popup.goTo', { index: popupIndex + 1 })}
                                />
                            ))}
                        </div>
                    ) : (
                        <div />
                    )}

                    <span className='min-w-0 justify-self-end truncate text-xs text-danger'>{errorMessage}</span>
                </div>
            </section>
        </div>
    )
}

export default PopupCarousel
