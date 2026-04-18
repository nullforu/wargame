import type { UserDetail } from '../../lib/types'
import { getRoleKey, useT } from '../../lib/i18n'

interface AccountCardProps {
    user: UserDetail
    authEmail?: string
    savingUsername: boolean
    onSave: () => void
    editingUsername: boolean
    usernameInput: string
    onEditingUsernameChange: (value: boolean) => void
    onUsernameInputChange: (value: string) => void
}

const AccountCard = ({ user, authEmail, savingUsername, onSave, editingUsername, usernameInput, onEditingUsernameChange, onUsernameInputChange }: AccountCardProps) => {
    const t = useT()

    const cancelEdit = () => {
        onEditingUsernameChange(false)
        onUsernameInputChange(user.username)
    }

    return (
        <div className='mt-6 rounded-2xl border border-border bg-surface p-6'>
            <h3 className='text-lg text-text'>{t('profile.account')}</h3>

            <div className='mt-4 space-y-2 text-sm text-text'>
                <div className='flex items-center justify-between gap-4'>
                    <span className='text-text-muted'>{t('common.username')}</span>

                    {editingUsername ? (
                        <div className='flex items-center gap-2'>
                            <input className='rounded-md border border-border bg-surface px-2 py-1 text-sm' value={usernameInput} onChange={(event) => onUsernameInputChange(event.target.value)} disabled={savingUsername} />
                            <button className='text-sm text-accent hover:underline disabled:opacity-50 cursor-pointer' disabled={savingUsername} onClick={onSave}>
                                {t('profile.save')}
                            </button>
                            <button className='text-sm text-text-subtle hover:underline cursor-pointer' onClick={cancelEdit}>
                                {t('profile.cancel')}
                            </button>
                        </div>
                    ) : (
                        <div className='flex items-center gap-3'>
                            <span>{user.username}</span>
                            <button className='text-xs text-accent hover:underline cursor-pointer' onClick={() => onEditingUsernameChange(true)}>
                                {t('profile.edit')}
                            </button>
                        </div>
                    )}
                </div>

                <div className='flex justify-between'>
                    <span className='text-text-muted'>{t('common.email')}</span>
                    <span>{authEmail}</span>
                </div>

                <div className='flex justify-between'>
                    <span className='text-text-muted'>{t('common.role')}</span>
                    <span className='uppercase text-accent'>{t(getRoleKey(user.role))}</span>
                </div>
            </div>
        </div>
    )
}

export default AccountCard
