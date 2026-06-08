import { Mail } from 'lucide-react'

import { PlaceholderPage } from '@/components/layout/placeholder-page'

// TODO (issue post/profil) : messagerie privée (DMs) via l'API Gateway.
export default function MessagesPage() {
  return (
    <PlaceholderPage
      icon={<Mail className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />}
      titleKey="nav.messages"
      headingKey="messages.heading"
      descKey="messages.desc"
    />
  )
}
