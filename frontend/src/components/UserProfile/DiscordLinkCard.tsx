import { useCallback, useEffect, useState } from 'react'

import { ApiError } from '../../lib/api'
import { useT } from '../../lib/i18n'
import type { DiscordStatus } from '../../lib/types'
import { useApi } from '../../lib/useApi'

type Banner = { kind: 'success' | 'error'; message: string } | null

const discordAvatarUrl = (id?: string, avatar?: string): string | null => {
    if (!id) return null
    if (avatar) {
        const ext = avatar.startsWith('a_') ? 'gif' : 'png'
        return `https://cdn.discordapp.com/avatars/${id}/${avatar}.${ext}?size=64`
    }
    try {
        const index = Number(BigInt(id) >> 22n) % 6
        return `https://cdn.discordapp.com/embed/avatars/${index}.png`
    } catch {
        return null
    }
}

const RESULT_KEYS: Record<string, { kind: 'success' | 'error'; key: string }> = {
    verified: { kind: 'success', key: 'profile.discord.resultVerified' },
    connected_not_joined: { kind: 'success', key: 'profile.discord.resultNotJoined' },
    role_failed: { kind: 'error', key: 'profile.discord.resultRoleFailed' },
    already_linked: { kind: 'error', key: 'profile.discord.resultAlreadyLinked' },
    state_invalid: { kind: 'error', key: 'profile.discord.resultStateInvalid' },
    error: { kind: 'error', key: 'profile.discord.resultError' },
}

const DiscordLinkCard = () => {
    const t = useT()
    const api = useApi()

    const [status, setStatus] = useState<DiscordStatus | null>(null)
    const [loading, setLoading] = useState(true)
    const [busy, setBusy] = useState(false)
    const [banner, setBanner] = useState<Banner>(null)

    const refresh = useCallback(async () => {
        try {
            const data = await api.discordStatus()
            setStatus(data)
        } catch {
            setBanner({ kind: 'error', message: t('profile.discord.loadError') })
        } finally {
            setLoading(false)
        }
    }, [api, t])

    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        const result = params.get('discord')
        if (result && RESULT_KEYS[result]) {
            const mapping = RESULT_KEYS[result]
            setBanner({ kind: mapping.kind, message: t(mapping.key) })
            params.delete('discord')
            const query = params.toString()
            window.history.replaceState({}, '', `${window.location.pathname}${query ? `?${query}` : ''}`)
        }
    }, [t])

    useEffect(() => {
        void refresh()
    }, [refresh])

    const onConnect = () => {
        window.location.href = api.discordConnectUrl()
    }

    const onSync = async () => {
        setBusy(true)
        setBanner(null)
        try {
            const data = await api.discordSyncRole()
            setStatus(data)
        } catch (error) {
            const message = error instanceof ApiError ? error.message : t('profile.discord.resultError')
            setBanner({ kind: 'error', message })
        } finally {
            setBusy(false)
        }
    }

    const onUnlink = async () => {
        setBusy(true)
        setBanner(null)
        try {
            await api.discordUnlink()
            setStatus({ connected: false, invite_url: status?.invite_url })
        } catch (error) {
            const message = error instanceof ApiError ? error.message : t('profile.discord.resultError')
            setBanner({ kind: 'error', message })
        } finally {
            setBusy(false)
        }
    }

    const wrapper = 'mt-6 rounded-none border-0 bg-transparent p-0 shadow-none md:rounded-lg md:border md:border-border md:bg-surface md:p-6'

    const roleStatus = status?.role_status
    const verified = roleStatus === 'VERIFIED'
    const notJoined = roleStatus === 'NOT_IN_GUILD' || roleStatus === 'LEFT_GUILD'
    const displayName = status?.discord_global_name || status?.discord_username
    const avatarUrl = discordAvatarUrl(status?.discord_user_id, status?.discord_avatar)

    return (
        <div className={wrapper}>
            <h3 className='text-lg text-text'>{t('profile.discord.title')}</h3>
            <p className='mt-2 text-xs text-text-muted'>{t('profile.discord.description')}</p>

            {banner ? <p className={`mt-3 text-sm ${banner.kind === 'success' ? 'text-accent' : 'text-danger'}`}>{banner.message}</p> : null}

            {loading ? (
                <div className='mt-4 space-y-2 animate-pulse'>
                    <div className='h-3 w-40 rounded bg-surface-muted' />
                    <div className='h-8 w-48 rounded bg-surface-muted' />
                </div>
            ) : !status?.connected ? (
                <div className='mt-4'>
                    <button type='button' className='rounded-md bg-[#5865F2] px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50' onClick={onConnect}>
                        {t('profile.discord.connect')}
                    </button>
                </div>
            ) : (
                <div className='mt-4 space-y-3 text-sm text-text'>
                    <div className='flex items-center gap-3'>
                        {avatarUrl ? <img src={avatarUrl} alt={displayName ?? 'Discord avatar'} className='h-10 w-10 shrink-0 rounded-full' loading='lazy' /> : <div className='h-10 w-10 shrink-0 rounded-full bg-surface-muted' />}
                        <div className='min-w-0'>
                            <div className='truncate text-text'>{displayName}</div>
                            {status.discord_username ? <div className='truncate text-xs text-text-muted'>@{status.discord_username}</div> : null}
                            {status.discord_user_id ? <div className='truncate font-mono text-[11px] text-text-subtle'>ID: {status.discord_user_id}</div> : null}
                        </div>
                    </div>

                    {verified ? <p className='text-accent'>{t('profile.discord.verified')}</p> : null}
                    {notJoined ? <p className='text-text-muted'>{t('profile.discord.notJoined')}</p> : null}
                    {roleStatus === 'ROLE_FAILED' ? <p className='text-danger'>{t('profile.discord.roleFailed')}</p> : null}

                    <div className='flex flex-wrap items-center gap-3'>
                        {notJoined && status.invite_url ? (
                            <a href={status.invite_url} target='_blank' rel='noreferrer' className='rounded-md bg-[#5865F2] px-3 py-1.5 text-sm font-medium text-white hover:opacity-90'>
                                {t('profile.discord.joinServer')}
                            </a>
                        ) : null}

                        {!verified ? (
                            <button type='button' className='rounded-md border border-border px-3 py-1.5 text-sm text-text disabled:opacity-50' onClick={onSync} disabled={busy}>
                                {busy ? t('profile.discord.checking') : t('profile.discord.recheck')}
                            </button>
                        ) : null}

                        <button type='button' className='text-sm text-text-subtle hover:underline disabled:opacity-50' onClick={onUnlink} disabled={busy}>
                            {busy ? t('profile.discord.unlinking') : t('profile.discord.unlink')}
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}

export default DiscordLinkCard
