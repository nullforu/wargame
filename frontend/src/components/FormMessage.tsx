import type { ReactNode } from 'react'

type Variant = 'error' | 'success' | 'warning' | 'info'

interface FormMessageProps {
    variant?: Variant
    message?: string
    className?: string
    children?: ReactNode
}

const styles: Record<Variant, string> = {
    error: 'border-danger/40 bg-danger/10 text-danger ',
    success: 'border-accent/40 bg-accent/10 text-accent-strong ',
    warning: 'border-warning/40 bg-warning/10 text-warning-strong ',
    info: 'border-border/70 bg-surface/60 text-text   ',
}

const FormMessage = ({ variant = 'error', message, className = '', children }: FormMessageProps) => {
    return <p className={`rounded-xl border px-4 py-2 text-xs ${styles[variant]} ${className}`}>{message ? message : children}</p>
}

export default FormMessage
