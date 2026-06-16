import { BookmarksView } from '@/components/bookmarks/bookmarks-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'

// Intercepting route : Signets en overlay au-dessus du feed (gardé monté).
// Accès direct/refresh → vraie page `(app)/signets/page.tsx`.
export default function InterceptedSignetsPage() {
  return (
    <OverlayWhenPath path="/signets">
      <FeedOverlay>
        <BookmarksView />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
