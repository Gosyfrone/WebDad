import { FeedOverlay } from '@/components/feed/feed-overlay'
import { OverlayWhenPath } from '@/components/feed/overlay-when-path'
import { ProfilView } from '@/components/profil/profil-view'

interface InterceptedPublicProfilPageProps {
  params: {
    username: string
  }
}

// Intercepting route : profil public en overlay au-dessus du feed (gardé monté).
// Accès direct/refresh → vraie page `(app)/profil/[username]/page.tsx`.
export default function InterceptedPublicProfilPage({
  params,
}: InterceptedPublicProfilPageProps) {
  return (
    <OverlayWhenPath path="/profil/" exact={false}>
      <FeedOverlay>
        <ProfilView username={params.username} />
      </FeedOverlay>
    </OverlayWhenPath>
  )
}
