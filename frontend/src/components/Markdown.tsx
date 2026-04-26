import { Children, isValidElement } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import 'katex/dist/katex.min.css'

export default function Markdown({ content, className }: { content: string; className?: string }) {
    return (
        <div className={`markdown-content min-w-0 ${className ?? ''}`}>
            <ReactMarkdown
                remarkPlugins={[remarkGfm, remarkMath]}
                rehypePlugins={[rehypeKatex]}
                components={{
                    h1: ({ children }) => <h1 className='mt-8 mb-5 text-3xl font-bold leading-tight'>{children}</h1>,
                    h2: ({ children }) => <h2 className='mt-7 mb-4 text-2xl font-bold leading-tight'>{children}</h2>,
                    h3: ({ children }) => <h3 className='mt-6 mb-4 text-xl font-semibold leading-tight'>{children}</h3>,
                    h4: ({ children }) => <h4 className='mt-5 mb-3 text-lg font-semibold leading-tight'>{children}</h4>,
                    h5: ({ children }) => <h5 className='mt-4 mb-2 text-base font-semibold leading-tight'>{children}</h5>,
                    h6: ({ children }) => <h6 className='mt-3 mb-1 text-base font-semibold leading-tight'>{children}</h6>,
                    p: ({ children }) => {
                        const nodes = Children.toArray(children)
                        const firstNode = nodes[0]
                        const isMathOnlyParagraph = nodes.length === 1 && isValidElement<{ className?: string }>(firstNode) && typeof firstNode.props.className === 'string' && firstNode.props.className.includes('katex')

                        return <p className={`mb-4 text-sm leading-7 md:text-base wrap-break-word ${isMathOnlyParagraph ? 'text-center' : ''}`}>{children}</p>
                    },
                    ul: ({ children }) => <ul className='mb-4 list-disc pl-6 text-sm leading-7 md:text-base'>{children}</ul>,
                    ol: ({ children }) => <ol className='mb-4 list-decimal pl-6 text-sm leading-7 md:text-base'>{children}</ol>,
                    li: ({ children }) => <li className='mb-2'>{children}</li>,
                    strong: ({ children }) => <strong className='font-bold'>{children}</strong>,
                    em: ({ children }) => <em className='italic'>{children}</em>,
                    code: ({ children, className }) => {
                        const text = String(children)
                        const isBlock = Boolean(className?.includes('language-') || text.includes('\n'))
                        return isBlock ? (
                            <code className={`inline-block min-w-max whitespace-pre font-mono text-[13px] leading-6 ${className ?? ''}`}>{children}</code>
                        ) : (
                            <code className={`rounded bg-surface-muted px-1 py-0.5 font-mono text-[0.9em] text-text ${className ?? ''}`}>{children}</code>
                        )
                    },
                    pre: ({ children }) => <pre className='mb-4 block w-full max-w-full overflow-x-auto overflow-y-hidden rounded bg-surface-muted p-4'>{children}</pre>,
                    table: ({ children }) => <table className='my-4 block w-full overflow-x-auto border-collapse text-sm md:text-base'>{children}</table>,
                    thead: ({ children }) => <thead className='bg-surface-muted'>{children}</thead>,
                    tbody: ({ children }) => <tbody>{children}</tbody>,
                    tr: ({ children }) => <tr className='border-b border-border/60'>{children}</tr>,
                    th: ({ children }) => <th className='border border-border/60 px-3 py-2 text-left font-semibold text-text'>{children}</th>,
                    td: ({ children }) => <td className='border border-border/60 px-3 py-2 align-top text-text-muted'>{children}</td>,
                    hr: () => <hr className='my-7 border-border/70' />,
                    img: ({ src, alt }) => (
                        <a href={String(src)} target='_blank' rel='noopener noreferrer' className='block mb-4'>
                            <img src={src} alt={alt} className='max-w-full h-auto mb-8 mt-8 transition-transform duration-300 rounded-lg' loading='lazy' />
                        </a>
                    ),
                    a: ({ href, children }) => (
                        <a href={href} className='text-accent hover:underline' target='_blank' rel='noopener noreferrer'>
                            {children}
                        </a>
                    ),
                    blockquote: ({ children }) => <blockquote className='my-4 border-l-4 border-border pl-4 italic text-text-muted'>{children}</blockquote>,
                }}
            >
                {content}
            </ReactMarkdown>
        </div>
    )
}
