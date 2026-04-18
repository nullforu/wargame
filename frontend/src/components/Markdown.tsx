import ReactMarkdown from 'react-markdown'

export default function Markdown({ content, className }: { content: string; className?: string }) {
    return (
        <div className={className}>
            <ReactMarkdown
                components={{
                    h1: ({ children }) => <h1 className='text-3xl font-bold mb-5 mt-8'>{children}</h1>,
                    h2: ({ children }) => <h2 className='text-2xl font-bold mb-4 mt-7'>{children}</h2>,
                    h3: ({ children }) => <h3 className='text-xl font-semibold mt-6 mb-4'>{children}</h3>,
                    h4: ({ children }) => <h4 className='text-lg font-semibold mt-5 mb-3'>{children}</h4>,
                    h5: ({ children }) => <h5 className='text-lg font-semibold mt-4 mb-2'>{children}</h5>,
                    h6: ({ children }) => <h6 className='text-lg font-semibold mt-3 mb-1'>{children}</h6>,
                    p: ({ children }) => <p className='text-base md:text-lg mb-4 wrap-break-word'>{children}</p>,
                    ul: ({ children }) => <ul className='list-disc list-inside mb-4 pl-1'>{children}</ul>,
                    ol: ({ children }) => <ol className='list-decimal list-inside mb-4'>{children}</ol>,
                    li: ({ children }) => <li className='mb-2'>{children}</li>,
                    strong: ({ children }) => <strong className='font-bold'>{children}</strong>,
                    em: ({ children }) => <em className='italic'>{children}</em>,
                    code: ({ children }) => <code className='bg-gray-100 dark:bg-gray-800 p-1 rounded'>{children}</code>,
                    pre: ({ children }) => <pre className='bg-gray-100 dark:bg-gray-800 p-4 rounded mb-4'>{children}</pre>,
                    hr: () => <hr className='border-gray-200 dark:border-gray-700 my-7' />,
                    img: ({ src, alt }) => (
                        <a href={String(src)} target='_blank' rel='noopener noreferrer' className='block mb-4'>
                            <img src={src} alt={alt} className='max-w-full h-auto mb-8 mt-8 transition-transform duration-300 rounded-lg' loading='lazy' />
                        </a>
                    ),
                    a: ({ href, children }) => (
                        <a href={href} className='text-blue-600 dark:text-blue-400 hover:underline' target='_blank' rel='noopener noreferrer'>
                            {children}
                        </a>
                    ),
                    blockquote: ({ children }) => <blockquote className='border-l-4 border-gray-300 dark:border-gray-600 pl-4 italic text-gray-600 dark:text-gray-300 my-4'>{children}</blockquote>,
                }}
            >
                {content}
            </ReactMarkdown>
        </div>
    )
}
