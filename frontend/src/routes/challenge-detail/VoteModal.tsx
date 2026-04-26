import type { CSSProperties } from 'react'

type VoteModalMode = 'solved' | 'revote'

interface VoteModalProps {
    isOpen: boolean
    mode: VoteModalMode
    title: string
    subtitle: string
    hint: string
    headline: string
    level: number
    description: string
    submitting: boolean
    canSubmit: boolean
    cancelLabel: string
    submitLabel: string
    voteAriaLabel: string
    onClose: () => void
    onLevelChange: (next: number) => void
    onSubmit: () => void
}

const toneByLevel = (level: number) => {
    if (level <= 2) return { circle: 'bg-emerald-400 ring-emerald-200', slider: 'rgb(52 211 153)', text: 'text-emerald-500' }
    if (level <= 4) return { circle: 'bg-teal-400 ring-teal-200', slider: 'rgb(45 212 191)', text: 'text-teal-500' }
    if (level <= 6) return { circle: 'bg-cyan-400 ring-cyan-200', slider: 'rgb(34 211 238)', text: 'text-cyan-500' }
    if (level <= 8) return { circle: 'bg-blue-500 ring-blue-200', slider: 'rgb(59 130 246)', text: 'text-blue-500' }
    return { circle: 'bg-rose-500 ring-rose-200', slider: 'rgb(244 63 94)', text: 'text-rose-500' }
}

const VoteModal = ({ isOpen, mode: _, title, subtitle, hint, headline, level, description, submitting, canSubmit, cancelLabel, submitLabel, voteAriaLabel, onClose, onLevelChange, onSubmit }: VoteModalProps) => {
    if (!isOpen) return null

    const clampedLevel = Math.max(1, Math.min(10, level))
    const tone = toneByLevel(clampedLevel)
    const sliderProgress = `${((clampedLevel - 1) / 9) * 100}%`
    const sliderStyle = {
        '--vote-level-color': tone.slider,
        '--vote-level-progress': sliderProgress,
    } as CSSProperties

    return (
        <div
            className='fixed inset-0 z-50 flex items-center justify-center bg-overlay/60 p-4'
            onClick={(event) => {
                if (event.target === event.currentTarget) onClose()
            }}
        >
            <div className='max-h-[calc(100vh-2rem)] w-full max-w-140 overflow-y-auto rounded-2xl border border-border bg-surface shadow-xl'>
                <div className='flex items-center justify-between border-b border-border px-5 py-4'>
                    <h4 className='text-lg font-semibold text-text sm:text-xl'>{title}</h4>
                    <button type='button' className='rounded-md p-1 text-text-subtle hover:bg-surface-muted hover:text-text' onClick={onClose} aria-label={cancelLabel}>
                        <svg viewBox='0 0 24 24' className='h-5 w-5' fill='none' stroke='currentColor' strokeWidth='2'>
                            <path d='M6 6l12 12M18 6l-12 12' />
                        </svg>
                    </button>
                </div>

                <div className='space-y-1 p-5 sm:p-6'>
                    <div className='space-y-2 text-center mb-4'>
                        <p className={`wrap-break-word text-2xl font-semibold leading-tight sm:text-3xl ${tone.text}`}>{headline}</p>
                        <p className='text-lg font-semibold text-text sm:text-xl'>{subtitle}</p>
                        <p className='text-sm text-text-muted'>{hint}</p>
                    </div>

                    <div className='rounded-xl border border-border bg-surface-muted p-4 sm:p-5 mb-4'>
                        <div className='mx-auto flex max-w-95 items-center justify-center gap-6 sm:gap-8'>
                            <button
                                type='button'
                                className='h-11 w-11 rounded-full border border-border text-3xl leading-none text-text hover:bg-surface-subtle disabled:opacity-40'
                                onClick={() => onLevelChange(clampedLevel - 1)}
                                disabled={submitting}
                            >
                                -
                            </button>
                            <div className={`flex h-16 w-16 items-center justify-center rounded-full text-4xl font-bold text-white shadow-sm ring-4 ${tone.circle}`}>{clampedLevel}</div>
                            <button
                                type='button'
                                className='h-11 w-11 rounded-full border border-border text-3xl leading-none text-text hover:bg-surface-subtle disabled:opacity-40'
                                onClick={() => onLevelChange(clampedLevel + 1)}
                                disabled={submitting}
                            >
                                +
                            </button>
                        </div>

                        <p className='mt-4 text-center text-sm text-text-muted sm:px-4'>{description}</p>
                        <input
                            type='range'
                            min={1}
                            max={10}
                            step={1}
                            value={clampedLevel}
                            onChange={(event) => onLevelChange(Number(event.target.value))}
                            disabled={submitting}
                            className='vote-level-slider mt-4 w-full'
                            style={sliderStyle}
                            aria-label={voteAriaLabel}
                        />
                    </div>

                    <div className='flex flex-col-reverse items-center justify-center gap-3 sm:flex-row'>
                        <button type='button' className='rounded-md px-3 py-2 text-sm text-text-muted hover:bg-surface-muted' onClick={onClose} disabled={submitting}>
                            {cancelLabel}
                        </button>
                        <button type='button' className='rounded-md bg-accent px-5 py-2 text-sm font-semibold text-white hover:bg-accent-strong disabled:opacity-60' onClick={onSubmit} disabled={!canSubmit || submitting}>
                            {submitLabel}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default VoteModal
