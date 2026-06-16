import { ExplorerView } from '@/components/explorer/explorer-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'

// Intercepting route : Explorer en overlay au-dessus du feed (gardé monté).
// Accès direct/refresh → vraie page `(app)/explorer/page.tsx`.
export default function InterceptedExplorerPage() {
  return (
    <OverlayWhenPath path="/explorer">
      <FeedOverlay>
        <ExplorerView />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
