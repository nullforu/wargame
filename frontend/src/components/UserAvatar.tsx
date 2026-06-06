import { generateColorFromUsername } from '../lib/utils'
import { mediaURL } from '../lib/media'

interface UserAvatarProps {
    username: string
    size?: 'sm' | 'md' | 'lg' | 'xl'
    profileImage?: string | null
}

const UserAvatar = ({ username, size = 'md', profileImage }: UserAvatarProps) => {
    const firstLetter = username.charAt(0).toUpperCase()
    const backgroundColor = generateColorFromUsername(username)
    const imageURL = mediaURL(profileImage)

    const sizeClasses = {
        sm: 'h-8 w-8 text-xs',
        md: 'h-10 w-10 text-sm',
        lg: 'h-12 w-12 text-base',
        xl: 'h-16 w-16 text-lg',
    }

    if (imageURL) {
        return <img className={`${sizeClasses[size]} shrink-0 rounded-full object-cover`} src={imageURL} alt={username} title={username} />
    }

    return (
        <div className={`${sizeClasses[size]} shrink-0 inline-flex items-center justify-center rounded-full font-semibold text-white`} style={{ backgroundColor }} title={username}>
            {firstLetter}
        </div>
    )
}

export default UserAvatar
