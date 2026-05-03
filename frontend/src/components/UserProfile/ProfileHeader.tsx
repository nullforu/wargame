import type { UserDetail } from '../../lib/types'
import { getRoleKey, useT } from '../../lib/i18n'
import UserAvatar from '../UserAvatar'

interface ProfileHeaderProps {
    user: UserDetail
}

const ProfileHeader = ({ user }: ProfileHeaderProps) => {
    const t = useT()

    const roleClasses = (role: string) => (role === 'admin' ? 'bg-secondary/20 text-secondary' : role === 'blocked' ? 'bg-danger/20 text-danger' : 'bg-accent/20 text-accent-strong')

    return (
        <div className='flex flex-wrap items-end justify-between gap-4'>
            <div className='flex items-center gap-4.75'>
                <UserAvatar username={user.username} profileImage={user.profile_image} size='xl' />
                <div>
                    <h2 className='text-2xl text-text sm:text-3xl'>{user.username}</h2>
                    <p className='mt-1 text-sm text-text-muted'>{user.affiliation?.trim() ? user.affiliation : ''}</p>
                    {user.bio ? (
                        <p className='mt-1 max-w-xl text-sm text-text-subtle' style={{ display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                            {user.bio}
                        </p>
                    ) : null}
                </div>
            </div>

            <span className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium uppercase ${roleClasses(user.role)}`}>{t(getRoleKey(user.role))}</span>
        </div>
    )
}

export default ProfileHeader
