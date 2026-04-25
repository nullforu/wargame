import { Editor, useMonaco } from '@monaco-editor/react'
import { useTheme } from '../lib/theme'
import { useEffect, useRef } from 'react'

const valueTemplates = {
    markdown: `Markdown is supported. You can make text **bold**, *italic* or \`inline code block\` and create lists as shown below.

* This is a list item.
* This is a list item.

Or:

1. This is also a list item.
2. This is also a list item.

\`\`\`typescript
console.log('This is a code block.')
\`\`\`

> Please note that not all Markdown features may be supported.
`,
    yaml: `# Must provide a valid Kubernetes Pod specification in YAML format.
# For more details, please refer to the documentation: https://github.com/nullforu/wargame-docs

apiVersion: v1
kind: Pod
metadata:
  name: challenge
spec:
  containers:
    - name: nginx
      image: nginx:stable
      ports:
        - containerPort: 80
          protocol: TCP
        - containerPort: 443
          protocol: TCP
      resources:
        requests:
          cpu: "50m"
          memory: "64Mi"
        limits:
          cpu: "50m"
          memory: "64Mi"
`,
}

interface MonacoEditorProps {
    value: string
    onChange?: (value: string) => void
    template?: keyof typeof valueTemplates
    readonly?: boolean
    language?: string
    height?: string
}

const MonacoEditor = ({ value, onChange, template, readonly = false, language, height = '200px' }: MonacoEditorProps) => {
    const { theme } = useTheme()
    const monaco = useMonaco()
    const templateBootstrappedRef = useRef(false)

    useEffect(() => {
        if (!monaco) return

        monaco.editor.defineTheme('wargame-light', {
            base: 'vs',
            inherit: true,
            rules: [],
            colors: {
                'editor.background': '#ffffff',
                'editor.foreground': '#1a202c',
                'editor.lineHighlightBackground': '#ebeef2',
                'editorCursor.foreground': '#0e8b7e',
                'editor.selectionBackground': '#28786e',
                'editor.inactiveSelectionBackground': '#28786e80',
                'editorLineNumber.foreground': '#6e7887',
                'editorLineNumber.activeForeground': '#1a202c',
                'editorIndentGuide.background': '#d8dde4',
                'editorIndentGuide.activeBackground': '#9aa4b2',
                'editorWidget.background': '#ffffff',
                'editorWidget.border': '#d8dde4',
                focusBorder: '#0e8b7e',
            },
        })

        monaco.editor.defineTheme('wargame-dark', {
            base: 'vs-dark',
            inherit: true,
            rules: [],
            colors: {
                'editor.background': '#181b1f',
                'editor.foreground': '#d2d6dc',
                'editor.lineHighlightBackground': '#1e2227',
                'editorCursor.foreground': '#26b496',
                'editor.selectionBackground': '#26786e',
                'editor.inactiveSelectionBackground': '#26786e80',
                'editorLineNumber.foreground': '#6e7682',
                'editorLineNumber.activeForeground': '#d2d6dc',
                'editorIndentGuide.background': '#3a4048',
                'editorIndentGuide.activeBackground': '#58606c',
                'editorWidget.background': '#181b1f',
                'editorWidget.border': '#3a4048',
                focusBorder: '#26b496',
            },
        })

        monaco.editor.setTheme(theme === 'dark' ? 'wargame-dark' : 'wargame-light')
    }, [monaco, theme])

    useEffect(() => {
        if (templateBootstrappedRef.current) return
        if (!template || !onChange) return
        if (value !== '') return
        onChange(valueTemplates[template])
        templateBootstrappedRef.current = true
    }, [onChange, template, value])

    useEffect(() => {
        if (value !== '') {
            templateBootstrappedRef.current = true
        }
    }, [value])

    return (
        <Editor
            height={height}
            width='100%'
            defaultLanguage={language ?? 'markdown'}
            value={value}
            onChange={(v) => onChange?.(v ?? '')}
            theme={theme === 'dark' ? 'wargame-dark' : 'wargame-light'}
            options={{
                readOnly: readonly,
                minimap: { enabled: false },
                wordWrap: 'off',
                scrollbar: {
                    vertical: 'auto',
                    horizontal: 'auto',
                },
                automaticLayout: true,
            }}
        />
    )
}

export default MonacoEditor
