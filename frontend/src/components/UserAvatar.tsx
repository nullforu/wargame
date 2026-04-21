import { generateColorFromUsername } from '../lib/utils'

interface UserAvatarProps {
    username: string
    size?: 'sm' | 'md' | 'lg'
}

const UserAvatar = ({ username, size = 'md' }: UserAvatarProps) => {
    const firstLetter = username.charAt(0).toUpperCase()
    const backgroundColor = generateColorFromUsername(username)

    const sizeClasses = {
        sm: 'h-8 w-8 text-xs',
        md: 'h-10 w-10 text-sm',
        lg: 'h-12 w-12 text-base',
    }

    return (
        <div className={`${sizeClasses[size]} inline-flex items-center justify-center rounded-full font-semibold text-white`} style={{ backgroundColor }} title={username}>
            {firstLetter}
        </div>
    )
}

export default UserAvatar
