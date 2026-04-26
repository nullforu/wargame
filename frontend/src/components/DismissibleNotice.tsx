import type { ReactNode } from 'react'
import { useState } from 'react'

interface DismissibleNoticeProps {
    storageKey: string
    children: ReactNode
    className?: string
    closeAriaLabel?: string
}

const DismissibleNotice = ({ storageKey, children, className = '', closeAriaLabel = '안내 메시지 닫기' }: DismissibleNoticeProps) => {
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
        <div className={`flex items-start justify-between gap-3 border border-accent/40 bg-accent/10 p-4 text-sm text-text ${className}`}>
            <p>{children}</p>
            <button aria-label={closeAriaLabel} className='shrink-0 text-base leading-none text-text-muted hover:text-text' onClick={handleDismiss} type='button'>
                &times;
            </button>
        </div>
    )
}

export default DismissibleNotice
