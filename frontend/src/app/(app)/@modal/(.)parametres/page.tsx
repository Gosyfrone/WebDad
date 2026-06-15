import { FeedOverlay } from '@/components/feed/feed-overlay'
import { SettingsView } from '@/components/settings/settings-view'

// Intercepting route : Paramètres en overlay au-dessus du feed (gardé monté).
// Accès direct/refresh → vraie page `(app)/parametres/page.tsx`.
export default function InterceptedParametresPage() {
  return (
    <FeedOverlay>
      <SettingsView />
    </FeedOverlay>
  )
}
