import { Settings2 } from 'lucide-react'

import { PlaceholderPage } from '@/components/layout/placeholder-page'

// TODO : implémenter le panneau d'administration (rôle administrator)
//   - lister / gérer les utilisateurs (via Gateway -> user-service)
//   - changer les rôles (user / moderator / administrator)
//   - garde d'accès : refuser si le rôle n'est pas administrator
export default function AdminPage() {
  return (
    <PlaceholderPage
      icon={<Settings2 className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
      titleKey="nav.admin"
      headingKey="admin.heading"
      descKey="admin.desc"
    />
  )
}
