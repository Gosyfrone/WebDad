import { FeedOverlay } from '@/components/feed/feed-overlay'
import { ProfilView } from '@/components/profil/profil-view'

// Profil de l'utilisateur connecté, en overlay au-dessus du feed persistant.
export default function ProfilPage() {
  return (
    <FeedOverlay>
      <ProfilView />
    </FeedOverlay>
  )
}
