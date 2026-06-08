import { Bell } from 'lucide-react'

import { PlaceholderPage } from '@/components/layout/placeholder-page'

// TODO (issue post/profil) : flux de notifications (likes, abonnements, mentions).
export default function NotificationsPage() {
  return (
    <PlaceholderPage
      icon={<Bell className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
      titleKey="nav.notifications"
      headingKey="notifications.heading"
      descKey="notifications.desc"
    />
  )
}
