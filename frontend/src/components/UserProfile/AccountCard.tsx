import type { Affiliation, PaginationMeta, UserDetail } from '../../lib/types'
import { useT } from '../../lib/i18n'
import UserAvatar from '../UserAvatar'

interface AccountCardProps {
    user: UserDetail
    authEmail?: string
    savingUsername: boolean
    onSave: () => void
    editingUsername: boolean
    usernameInput: string
    onEditingUsernameChange: (value: boolean) => void
    onUsernameInputChange: (value: string) => void
    editingBio: boolean
    bioInput: string
    savingBio: boolean
    onEditingBioChange: (value: boolean) => void
    onBioInputChange: (value: string) => void
    onSaveBio: () => void
    editingAffiliation: boolean
    onEditingAffiliationChange: (value: boolean) => void
    affiliationQuery: string
    onAffiliationQueryChange: (value: string) => void
    selectedAffiliationID: number | null
    onSelectedAffiliationIDChange: (value: number | null) => void
    affiliations: Affiliation[]
    affiliationPagination: PaginationMeta
    loadingAffiliations: boolean
    savingAffiliation: boolean
    onAffiliationPageChange: (value: number) => void
    onSaveAffiliation: () => void
}

const AccountCard = ({
    user,
    authEmail,
    savingUsername,
    onSave,
    editingUsername,
    usernameInput,
    onEditingUsernameChange,
    onUsernameInputChange,
    editingBio,
    bioInput,
    savingBio,
    onEditingBioChange,
    onBioInputChange,
    onSaveBio,
    editingAffiliation,
    onEditingAffiliationChange,
    affiliationQuery,
    onAffiliationQueryChange,
    selectedAffiliationID,
    onSelectedAffiliationIDChange,
    affiliations,
    affiliationPagination,
    loadingAffiliations,
    savingAffiliation,
    onAffiliationPageChange,
    onSaveAffiliation,
}: AccountCardProps) => {
    const t = useT()

    const cancelEdit = () => {
        onEditingUsernameChange(false)
        onUsernameInputChange(user.username)
    }

    const cancelAffiliationEdit = () => {
        onEditingAffiliationChange(false)
        onSelectedAffiliationIDChange(user.affiliation_id)
    }

    const cancelBioEdit = () => {
        onEditingBioChange(false)
        onBioInputChange(user.bio ?? '')
    }

    return (
        <div className='mt-6 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-lg md:border md:border-border md:bg-surface md:p-6'>
            <h3 className='text-lg text-text'>{t('profile.account')}</h3>

            <div className='mt-4 space-y-2 text-sm text-text'>
                <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4'>
                    <span className='text-text-muted'>{t('common.username')}</span>

                    {editingUsername ? (
                        <div className='flex w-full flex-wrap items-center gap-2 sm:w-auto'>
                            <input className='w-full rounded-md border border-border bg-surface px-2 py-1 text-sm sm:w-auto' value={usernameInput} onChange={(event) => onUsernameInputChange(event.target.value)} disabled={savingUsername} />
                            <button className='text-sm text-accent hover:underline disabled:opacity-50 cursor-pointer' disabled={savingUsername} onClick={onSave}>
                                {t('profile.save')}
                            </button>
                            <button className='text-sm text-text-subtle hover:underline cursor-pointer' onClick={cancelEdit}>
                                {t('profile.cancel')}
                            </button>
                        </div>
                    ) : (
                        <div className='flex items-center gap-3.75 self-start sm:self-auto'>
                            <UserAvatar username={user.username} size='md' />
                            <span>{user.username}</span>
                            <button className='text-xs text-accent hover:underline cursor-pointer' onClick={() => onEditingUsernameChange(true)}>
                                {t('profile.edit')}
                            </button>
                        </div>
                    )}
                </div>

                <div className='flex flex-col gap-1 sm:flex-row sm:justify-between'>
                    <span className='text-text-muted'>{t('common.email')}</span>
                    <span>{authEmail}</span>
                </div>

                <div className='flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between'>
                    <span className='text-text-muted'>{t('profile.bio')}</span>
                    {editingBio ? (
                        <div className='w-full max-w-sm space-y-2'>
                            <textarea
                                className='h-24 w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                value={bioInput}
                                onChange={(event) => onBioInputChange(event.target.value)}
                                maxLength={400}
                                disabled={savingBio}
                                placeholder={t('profile.bioPlaceholder')}
                            />
                            <div className='flex items-center gap-3'>
                                <button className='text-sm text-accent hover:underline disabled:opacity-50 cursor-pointer' disabled={savingBio} onClick={onSaveBio}>
                                    {t('profile.save')}
                                </button>
                                <button className='text-sm text-text-subtle hover:underline cursor-pointer' onClick={cancelBioEdit}>
                                    {t('profile.cancel')}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className='flex max-w-sm items-start gap-2 self-start sm:self-auto'>
                            <span className='text-right text-sm text-text-muted' style={{ display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                                {(user.bio ?? '').trim() === '' ? t('profile.noBio') : user.bio}
                            </span>
                            <button className='text-xs text-accent hover:underline cursor-pointer' onClick={() => onEditingBioChange(true)}>
                                {t('profile.edit')}
                            </button>
                        </div>
                    )}
                </div>

                <div className='flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between'>
                    <span className='text-text-muted'>{t('common.affiliation')}</span>

                    {editingAffiliation ? (
                        <div className='w-full max-w-sm space-y-2'>
                            <input
                                type='text'
                                className='w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text focus:border-accent focus:outline-none'
                                value={affiliationQuery}
                                onChange={(event) => onAffiliationQueryChange(event.target.value)}
                                placeholder={t('profile.affiliationSearchPlaceholder')}
                                disabled={savingAffiliation || loadingAffiliations}
                            />
                            <div className='max-h-52 overflow-y-auto rounded-md border border-border bg-surface'>
                                <button
                                    type='button'
                                    className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-surface-muted ${selectedAffiliationID === null ? 'bg-surface-muted text-accent' : 'text-text'}`}
                                    onClick={() => onSelectedAffiliationIDChange(null)}
                                    disabled={savingAffiliation || loadingAffiliations}
                                >
                                    <span />
                                </button>
                                {affiliations.map((item) => (
                                    <button
                                        key={item.id}
                                        type='button'
                                        className={`flex w-full items-center justify-between border-t border-border/60 px-3 py-2 text-left text-sm hover:bg-surface-muted ${selectedAffiliationID === item.id ? 'bg-surface-muted text-accent' : 'text-text'}`}
                                        onClick={() => onSelectedAffiliationIDChange(item.id)}
                                        disabled={savingAffiliation || loadingAffiliations}
                                    >
                                        <span className='truncate'>{item.name}</span>
                                    </button>
                                ))}
                                {!loadingAffiliations && affiliations.length === 0 ? <p className='px-3 py-2 text-xs text-text-subtle'>{t('profile.affiliationSearchEmpty')}</p> : null}
                            </div>
                            {loadingAffiliations ? <p className='text-xs text-text-subtle'>{t('common.loading')}</p> : null}
                            <div className='flex items-center justify-between text-xs text-text-subtle'>
                                <button
                                    type='button'
                                    className='rounded-md border border-border px-2 py-1 disabled:opacity-50'
                                    disabled={!affiliationPagination.has_prev || savingAffiliation || loadingAffiliations}
                                    onClick={() => onAffiliationPageChange(Math.max(1, affiliationPagination.page - 1))}
                                >
                                    {t('common.previous')}
                                </button>
                                <span>
                                    {affiliationPagination.page} / {affiliationPagination.total_pages || 1}
                                </span>
                                <button
                                    type='button'
                                    className='rounded-md border border-border px-2 py-1 disabled:opacity-50'
                                    disabled={!affiliationPagination.has_next || savingAffiliation || loadingAffiliations}
                                    onClick={() => onAffiliationPageChange(affiliationPagination.page + 1)}
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                            <div className='flex items-center gap-3'>
                                <button className='text-sm text-accent hover:underline disabled:opacity-50 cursor-pointer' disabled={savingAffiliation || loadingAffiliations} onClick={onSaveAffiliation}>
                                    {t('profile.save')}
                                </button>
                                <button className='text-sm text-text-subtle hover:underline cursor-pointer' onClick={cancelAffiliationEdit}>
                                    {t('profile.cancel')}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className='flex items-center gap-2 self-start sm:self-auto'>
                            <span>{user.affiliation?.trim() ? user.affiliation : ''}</span>
                            <button className='text-xs text-accent hover:underline cursor-pointer' onClick={() => onEditingAffiliationChange(true)}>
                                {t('profile.edit')}
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default AccountCard
