import { Settings } from 'lucide-react'

import { PlaceholderPage } from '@/components/layout/placeholder-page'

// TODO : préférences du compte (sélecteur de thème/accent cf. lib/themes.ts, langue, confidentialité…).
export default function ParametresPage() {
  return (
    <PlaceholderPage
      icon={<Settings className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
      titleKey="settings.title"
      headingKey="settings.account_title"
      descKey="settings.account_desc"
    />
  )
}
