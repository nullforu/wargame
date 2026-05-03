import { generateColorFromUsername } from '../lib/utils'

interface UserAvatarProps {
    username: string
    size?: 'sm' | 'md' | 'lg' | 'xl'
    profileImage?: string | null
}

const PROFILE_IMAGE_CDN_BASE = String(import.meta.env.VITE_S3_MEDIA_CDN_BASE_URL ?? '')
    .trim()
    .replace(/\/+$/, '')

const joinCDNURL = (base: string, key: string) => {
    const normalizedKey = key.replace(/^\/+/, '')
    if (!base || !normalizedKey) return ''
    return `${base}/${normalizedKey}`
}

const UserAvatar = ({ username, size = 'md', profileImage }: UserAvatarProps) => {
    const firstLetter = username.charAt(0).toUpperCase()
    const backgroundColor = generateColorFromUsername(username)
    const imageURL = profileImage ? joinCDNURL(PROFILE_IMAGE_CDN_BASE, profileImage) : ''

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
