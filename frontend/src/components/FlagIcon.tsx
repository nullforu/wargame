interface Props {
    className?: string
}

export default ({ className }: Props) => (
    <span className={className ?? ''}>
        <svg viewBox='0 0 24 24' className='w-full h-full'>
            <path d='M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z' fill='currentColor' opacity='0.7' />
            <path d='M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z' fill='none' stroke='currentColor' strokeLinecap='round' strokeLinejoin='round' />
        </svg>
    </span>
)
