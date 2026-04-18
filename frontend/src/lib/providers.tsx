import { AuthProvider } from './auth'
import { ThemeProvider } from './theme'
import { LocaleProvider } from './i18n'

export const AppProviders = ({ children }: { children: React.ReactNode }) => {
    return (
        <LocaleProvider>
            <AuthProvider>
                <ThemeProvider>{children}</ThemeProvider>
            </AuthProvider>
        </LocaleProvider>
    )
}
