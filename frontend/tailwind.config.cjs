/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ['./index.html', './src/**/*.{ts,tsx,js,jsx}'],
    darkMode: 'class',
    theme: {
        extend: {
            fontFamily: {
                display: ['"Space Grotesk"', 'ui-sans-serif', 'system-ui'],
                body: ['"IBM Plex Sans"', 'ui-sans-serif', 'system-ui'],
            },
            colors: {
                background: 'rgb(var(--app-bg) / <alpha-value>)',
                surface: {
                    DEFAULT: 'rgb(var(--color-surface) / <alpha-value>)',
                    muted: 'rgb(var(--color-surface-muted) / <alpha-value>)',
                    subtle: 'rgb(var(--color-surface-subtle) / <alpha-value>)',
                },
                border: {
                    DEFAULT: 'rgb(var(--color-border) / <alpha-value>)',
                },
                text: {
                    DEFAULT: 'rgb(var(--color-text) / <alpha-value>)',
                    muted: 'rgb(var(--color-text-muted) / <alpha-value>)',
                    subtle: 'rgb(var(--color-text-subtle) / <alpha-value>)',
                    inverse: 'rgb(var(--color-text-inverse) / <alpha-value>)',
                },
                accent: {
                    DEFAULT: 'rgb(var(--color-accent) / <alpha-value>)',
                    strong: 'rgb(var(--color-accent-strong) / <alpha-value>)',
                    foreground: 'rgb(var(--color-accent-foreground) / <alpha-value>)',
                },
                secondary: {
                    DEFAULT: 'rgb(var(--color-secondary) / <alpha-value>)',
                    foreground: 'rgb(var(--color-secondary-foreground) / <alpha-value>)',
                },
                danger: {
                    DEFAULT: 'rgb(var(--color-danger) / <alpha-value>)',
                    strong: 'rgb(var(--color-danger-strong) / <alpha-value>)',
                    foreground: 'rgb(var(--color-danger-foreground) / <alpha-value>)',
                },
                warning: {
                    DEFAULT: 'rgb(var(--color-warning) / <alpha-value>)',
                    strong: 'rgb(var(--color-warning-strong) / <alpha-value>)',
                    foreground: 'rgb(var(--color-warning-foreground) / <alpha-value>)',
                },
                success: {
                    DEFAULT: 'rgb(var(--color-success) / <alpha-value>)',
                    strong: 'rgb(var(--color-success-strong) / <alpha-value>)',
                    foreground: 'rgb(var(--color-success-foreground) / <alpha-value>)',
                },
                info: {
                    DEFAULT: 'rgb(var(--color-info) / <alpha-value>)',
                },
                ring: 'rgb(var(--color-ring) / <alpha-value>)',
                overlay: 'rgb(var(--color-overlay) / <alpha-value>)',
                contrast: {
                    DEFAULT: 'rgb(var(--color-contrast) / <alpha-value>)',
                    foreground: 'rgb(var(--color-contrast-foreground) / <alpha-value>)',
                },
                chart: {
                    1: 'rgb(var(--chart-1) / <alpha-value>)',
                    2: 'rgb(var(--chart-2) / <alpha-value>)',
                    3: 'rgb(var(--chart-3) / <alpha-value>)',
                    4: 'rgb(var(--chart-4) / <alpha-value>)',
                    5: 'rgb(var(--chart-5) / <alpha-value>)',
                    6: 'rgb(var(--chart-6) / <alpha-value>)',
                    7: 'rgb(var(--chart-7) / <alpha-value>)',
                    8: 'rgb(var(--chart-8) / <alpha-value>)',
                    9: 'rgb(var(--chart-9) / <alpha-value>)',
                    10: 'rgb(var(--chart-10) / <alpha-value>)',
                    other: 'rgb(var(--color-secondary) / <alpha-value>)',
                },
            },
        },
    },
    plugins: [],
}
