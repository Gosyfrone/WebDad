import { BookmarksView } from '@/components/bookmarks/bookmarks-view'
import { FeedOverlay } from '@/components/feed/feed-overlay'

/** Signets : posts enregistrés, organisés en collections (post-service via gateway).
 *  Rendu en overlay au-dessus du feed persistant du layout. */
export default function SignetsPage() {
  return (
    <FeedOverlay headerOffset={false}>
      <BookmarksView />
    </FeedOverlay>
  )
}
