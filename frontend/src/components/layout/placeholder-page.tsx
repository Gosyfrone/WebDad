'use client'

import { useT } from '@/components/language-provider'

interface PlaceholderPageProps {
  /**
   * Icône déjà rendue (JSX), passée par la page. On reçoit un `ReactNode` et non
   * un composant : une page Server Component ne peut pas passer une *fonction*
   * (le composant lucide) à ce Client Component, mais un élément JSX, oui.
   */
  icon: React.ReactNode
  /** Clé i18n du titre de l'en-tête (ex. `nav.notifications`). */
  titleKey: string
  /** Clé i18n du gros titre dans la carte (ex. `notifications.heading`). */
  headingKey: string
  /** Clé i18n du sous-texte explicatif (ex. `notifications.desc`). */
  descKey: string
}

/**
 * État vide partagé par les pages encore non implémentées (notifications,
 * messages, administration, modération, paramètres). Client component pour
 * pouvoir traduire via {@link useT} ; les pages restent de minces wrappers.
 */
export function PlaceholderPage({ icon, titleKey, headingKey, descKey }: PlaceholderPageProps) {
  const t = useT()
  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">{t(titleKey)}</h1>
      </div>
      <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
        {icon}
        <h2 className="text-lg font-bold text-foreground">{t(headingKey)}</h2>
        <p className="max-w-sm text-sm text-muted-foreground">{t(descKey)}</p>
      </div>
    </div>
  )
}
