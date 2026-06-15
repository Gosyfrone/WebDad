import { FeedOverlay } from '@/components/feed/feed-overlay'
import { ProfilView } from '@/components/profil/profil-view'

// Intercepting route : profil de l'utilisateur courant en overlay au-dessus du
// feed (gardé monté). Accès direct/refresh → vraie page `(app)/profil/page.tsx`.
export default function InterceptedProfilPage() {
  return (
    <FeedOverlay>
      <ProfilView />
    </FeedOverlay>
  )
}
