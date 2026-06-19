import type { Metadata } from 'next'
import localFont from 'next/font/local'
import { ThemeProvider } from '@/components/theme-provider'
import { LanguageProvider } from '@/components/language-provider'
import { RouteOriginTracker } from '@/components/route-origin-tracker'
import { SessionBootstrap } from '@/components/session-bootstrap'
import { Toaster } from '@/components/ui/toaster'
import { CUSTOM_THEME_INLINE_SCRIPT } from '@/lib/custom-theme'
import './globals.css'

// Police Inter auto-hébergée (next/font/local) : le woff2 variable (latin, axe de
// graisse 100→900) est versionné dans le repo, donc le build ne dépend plus d'un
// fetch réseau vers Google Fonts — builds Docker/CI reproductibles et hors-ligne.
const inter = localFont({
  src: './fonts/inter-latin-wght-normal.woff2',
  weight: '100 900',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'Breezy',
  description: 'Réseau social distribué en microservices.',
  icons: {
    icon: '/logo_only.png',
    apple: '/logo_only.png',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="fr" suppressHydrationWarning>
      <head>
        {/*
          Applique le thème personnalisé (variables CSS pré-calculées en
          localStorage) AVANT le premier paint, pour éviter tout flash de
          couleur. Logique dans lib/custom-theme.ts. */}
        <script dangerouslySetInnerHTML={{ __html: CUSTOM_THEME_INLINE_SCRIPT }} />
      </head>
      <body className={inter.className}>
        {/*
          Thème par défaut : clair. `enableSystem` est activé pour permettre
          l'option « Système » du sélecteur (ThemeToggle), qui suit la préférence
          de l'OS/navigateur. Le défaut reste clair (et non system).
        */}
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem
          disableTransitionOnChange
        >
          <LanguageProvider>
            <SessionBootstrap />
            <RouteOriginTracker />
            {children}
            <Toaster />
          </LanguageProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
