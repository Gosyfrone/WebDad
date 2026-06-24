import { FeedOverlay } from '@/components/feed/feed-overlay'
import { HelpContent } from '@/components/help/help-content'

// Page d'aide (présentation de chaque section + relance du didacticiel),
// rendue en overlay au-dessus du feed persistant du layout.
export default function AidePage() {
  return (
    <FeedOverlay>
      <HelpContent />
    </FeedOverlay>
  )
}
