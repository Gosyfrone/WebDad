import { FeedOverlay } from '@/components/feed/feed-overlay'
import { SettingsView } from '@/components/settings/settings-view'

// Paramètres, en overlay au-dessus du feed persistant du layout.
export default function ParametresPage() {
  return (
    <FeedOverlay>
      <SettingsView />
    </FeedOverlay>
  )
}
