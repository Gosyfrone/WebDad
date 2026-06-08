import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import { ThemeProvider } from '@/components/theme-provider'
import { LanguageProvider } from '@/components/language-provider'
import { RouteOriginTracker } from '@/components/route-origin-tracker'
import { Toaster } from '@/components/ui/toaster'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

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
            <RouteOriginTracker />
            {children}
            <Toaster />
          </LanguageProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
