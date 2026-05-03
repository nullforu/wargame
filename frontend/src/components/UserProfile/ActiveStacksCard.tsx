import type { Stack } from '../../lib/types'
import { useT } from '../../lib/i18n'

interface ActiveStacksCardProps {
    activeStacks: Stack[]
    stacksError: string
    stacksLoading: boolean
    stackDeletingId: number | null
    onRefresh: () => void
    onDelete: (challengeId: number) => void
    formatOptionalDateTime: (value?: string | null) => string
}

const ActiveStacksCard = ({ activeStacks, stacksError, stacksLoading, stackDeletingId, onRefresh, onDelete, formatOptionalDateTime }: ActiveStacksCardProps) => {
    const t = useT()
    const formatEndpoints = (stack: Stack) => {
        if (!stack.node_public_ip || stack.ports.length === 0) return t('common.pending')
        return stack.ports.map((port) => `${port.protocol} ${stack.node_public_ip}:${port.node_port}`).join(', ')
    }

    const formatChallengeTitle = (stack: Stack) => {
        if (stack.challenge_title) {
            return t('profile.challengeTitle', { title: stack.challenge_title, id: stack.challenge_id })
        }
        return t('profile.challengeLabel', { id: stack.challenge_id })
    }

    return (
        <div className='mt-6 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-lg md:border md:border-border md:bg-surface md:p-6'>
            <div className='flex flex-wrap items-center justify-between gap-4'>
                <h3 className='text-lg text-text'>{t('profile.activeStacks')}</h3>
                <button className='text-xs uppercase tracking-wide text-text-subtle hover:text-text disabled:opacity-60 cursor-pointer' onClick={onRefresh} disabled={stacksLoading}>
                    {stacksLoading ? t('common.loading') : t('common.refresh')}
                </button>
            </div>

            {stacksError ? (
                <p className='mt-4 rounded-none border-0 bg-danger/10 px-3 py-2 text-xs text-danger md:rounded-xl md:border md:border-danger/40 md:px-4'>{stacksError}</p>
            ) : activeStacks.length === 0 ? (
                <div className='mt-4 rounded-none border-0 bg-surface-muted p-4 text-center md:rounded-xl md:border md:border-border md:p-5'>
                    <p className='text-sm text-text-muted'>{t('profile.noActiveStacks')}</p>
                </div>
            ) : (
                <div className='mt-4 divide-y divide-border/50 md:divide-y-0 md:space-y-3'>
                    {activeStacks.map((stack) => (
                        <div key={stack.challenge_id} className='rounded-none border-0 bg-transparent p-3 md:rounded-xl md:border md:border-border md:bg-surface-muted md:p-5'>
                            <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                                <div>
                                    <p className='text-sm font-medium text-text'>{formatChallengeTitle(stack)}</p>
                                    <p className='mt-1 text-xs text-text-subtle'>{t('profile.statusLabel', { status: stack.status })}</p>
                                    <p className='mt-1 text-xs text-text-subtle'>{t('profile.createdBy', { username: stack.created_by_username })}</p>
                                </div>
                                <div className='flex w-full flex-col gap-2 text-xs text-text-muted sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:gap-3'>
                                    <span className='break-all'>{formatEndpoints(stack)}</span>
                                    <button
                                        className='w-full rounded-lg border border-danger/30 px-3 py-1.5 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60 sm:w-auto cursor-pointer'
                                        type='button'
                                        onClick={() => onDelete(stack.challenge_id)}
                                        disabled={stackDeletingId === stack.challenge_id || stacksLoading}
                                    >
                                        {stackDeletingId === stack.challenge_id ? t('profile.deleting') : t('profile.delete')}
                                    </button>
                                </div>
                            </div>
                            <div className='mt-2 text-xs text-text-subtle'>{t('profile.ttlLabel', { time: formatOptionalDateTime(stack.ttl_expires_at) })}</div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}

export default ActiveStacksCard
