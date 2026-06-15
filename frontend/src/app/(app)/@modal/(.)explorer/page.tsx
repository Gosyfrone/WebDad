import { ExplorerView } from '@/components/explorer/explorer-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'

// Intercepting route : Explorer en overlay au-dessus du feed (gardé monté).
// Accès direct/refresh → vraie page `(app)/explorer/page.tsx`.
export default function InterceptedExplorerPage() {
  return (
    <FeedOverlay>
      <ExplorerView />
    </FeedOverlay>
  )
}
