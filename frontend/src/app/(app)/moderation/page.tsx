import { Shield } from 'lucide-react'

import { PlaceholderPage } from '@/components/layout/placeholder-page'

// TODO : implémenter le panneau de modération (rôles moderator + administrator)
//   - lister les posts signalés (via Gateway -> post-service)
//   - actions : masquer / supprimer un post, avertir un utilisateur
//   - garde d'accès : refuser si le rôle n'est pas moderator/administrator
export default function ModerationPage() {
  return (
    <PlaceholderPage
      icon={<Shield className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
      titleKey="nav.moderation"
      headingKey="moderation.heading"
      descKey="moderation.desc"
    />
  )
}
