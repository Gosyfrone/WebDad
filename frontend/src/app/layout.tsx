import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import { ThemeProvider } from '@/components/theme-provider'
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
          Thème par défaut : clair. On n'active PAS `enableSystem` pour ne pas
          suivre la préférence sombre de l'OS/navigateur (cf. CLAUDE.md §5).
          Le mode sombre reste disponible via le futur sélecteur (setTheme('dark')).
        */}
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem={false}
          disableTransitionOnChange
        >
          {children}
          <Toaster />
        </ThemeProvider>
      </body>
    </html>
  )
}
