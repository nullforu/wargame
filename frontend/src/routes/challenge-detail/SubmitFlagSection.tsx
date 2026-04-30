import { navigate } from '../../lib/router'

interface SubmissionState {
    status: 'idle' | 'loading' | 'success' | 'error'
    message?: string
}

interface SubmitFlagSectionProps {
    flagInput: string
    isSubmissionDisabled: boolean
    submission: SubmissionState
    isAuthenticated: boolean
    onFlagInputChange: (value: string) => void
    onSubmit: () => void
    t: (key: string, vars?: Record<string, string | number>) => string
}

const SubmitFlagSection = ({ flagInput, isSubmissionDisabled, submission, isAuthenticated, onFlagInputChange, onSubmit, t }: SubmitFlagSectionProps) => {
    return (
        <form
            className='rounded-md bg-surface-muted p-3 sm:p-4 mt-4 shadow-sm border border-border/30'
            onSubmit={(e) => {
                e.preventDefault()
                onSubmit()
            }}
        >
            <label className='text-sm font-semibold text-text'>{t('challenge.enterFlag')}</label>

            <div className='mt-3 flex flex-col gap-2 sm:flex-row'>
                <input
                    className='min-w-0 flex-1 rounded-md border border-border/40 bg-surface px-3 py-2.5 text-sm text-text focus:border-accent focus:outline-none'
                    type='text'
                    value={flagInput}
                    onChange={(e) => onFlagInputChange(e.target.value)}
                    disabled={isSubmissionDisabled}
                />

                <button className='w-full rounded-md bg-accent px-4 py-2.5 text-sm text-white hover:bg-accent-strong disabled:opacity-60 sm:w-auto sm:min-w-30' disabled={isSubmissionDisabled}>
                    {submission.status === 'loading' ? t('challenge.submitting') : t('challenge.submit')}
                </button>
            </div>

            {!isAuthenticated ? (
                <p className='mt-2 text-xs text-warning'>
                    {t('challenge.loginToSubmitPrefix')}{' '}
                    <a className='underline cursor-pointer' href='/login' onClick={(e) => navigate('/login', e)}>
                        {t('auth.loginLink')}
                    </a>{' '}
                    {t('challenge.loginToSubmitSuffix')}
                </p>
            ) : null}

            {submission.message && <p className={`mt-2 text-sm ${submission.status === 'success' ? 'text-success' : 'text-danger'}`}>{submission.message}</p>}
        </form>
    )
}

export default SubmitFlagSection
