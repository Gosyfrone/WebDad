import { FeedOverlay } from '@/components/feed/feed-overlay'
import { ProfilView } from '@/components/profil/profil-view'

interface PublicProfilPageProps {
  params: {
    username: string
  }
}

// Profil public, en overlay au-dessus du feed persistant du layout.
export default function PublicProfilPage({ params }: PublicProfilPageProps) {
  return (
    <FeedOverlay>
      <ProfilView username={params.username} />
    </FeedOverlay>
  )
}
