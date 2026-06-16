import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'
import { ProfilView } from '@/components/profil/profil-view'

// Intercepting route : profil de l'utilisateur courant en overlay au-dessus du
// feed (gardé monté). Accès direct/refresh → vraie page `(app)/profil/page.tsx`.
export default function InterceptedProfilPage() {
  return (
    <OverlayWhenPath path="/profil">
      <FeedOverlay>
        <ProfilView />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
