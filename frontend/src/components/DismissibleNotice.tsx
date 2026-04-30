import type { ReactNode } from 'react'
import { useState } from 'react'

interface DismissibleNoticeProps {
    storageKey: string
    children: ReactNode
    className?: string
    size?: 'small' | 'medium'
}

const DismissibleNotice = ({ storageKey, children, className = '', size = 'medium' }: DismissibleNoticeProps) => {
    const [isVisible, setIsVisible] = useState<boolean>(() => {
        try {
            return window.localStorage.getItem(storageKey) !== 'true'
        } catch {
            return true
        }
    })

    if (!isVisible) {
        return null
    }

    const handleDismiss = () => {
        setIsVisible(false)
        try {
            window.localStorage.setItem(storageKey, 'true')
        } catch {}
    }

    return (
        <div className={`flex items-start justify-between gap-3 border text-sm text-text-muted ${size === 'small' ? 'border-accent/25 bg-accent/7 p-3' : 'border-accent/40 bg-accent/10 p-4'} ${className}`}>
            <p>{children}</p>
            <button aria-label='Close' className='shrink-0 text-base leading-none text-text-muted hover:text-text' onClick={handleDismiss} type='button'>
                &times;
            </button>
        </div>
    )
}

export default DismissibleNotice
