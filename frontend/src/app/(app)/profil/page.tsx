import type { PostCardProps } from '@/components/feed/post-card'
import { ProfilView } from '@/components/profil/profil-view'
import type { ProfilDetails } from '@/types'

// Données fictives pour le squelette. À remplacer par les appels API :
//   - profil : GET ${NEXT_PUBLIC_API_URL}/profils/me (Gateway -> profil-service)
//   - posts  : GET ${NEXT_PUBLIC_API_URL}/posts?author=<userId> (post-service)
const STUB_PROFIL: ProfilDetails = {
  userId: '1',
  displayName: 'Zaid',
  username: 'zaid_dev',
  role: 'administrator',
  bio: 'Étudiant FISA INFO A3 · J\'aime les architectures microservices, le Go et le café. On construit Breezy 🚀',
  avatarUrl: '',
  bannerUrl: '',
  joinedAt: '2026-06-01T00:00:00.000Z',
  followersCount: 1240,
  followingCount: 187,
  postsCount: 2,
}

const STUB_POSTS: PostCardProps[] = [
  {
    id: '1',
    name: STUB_PROFIL.displayName,
    handle: `@${STUB_PROFIL.username}`,
    initials: STUB_PROFIL.displayName.charAt(0),
    timestamp: '2h',
    content:
      'Première journée sur le projet Breezy 🚀 Architecture microservices en Go, on attaque l\'auth-service ce soir !',
    likes: 24,
    comments: 3,
    reposts: 5,
    views: 1200,
  },
  {
    id: '5',
    name: STUB_PROFIL.displayName,
    handle: `@${STUB_PROFIL.username}`,
    initials: STUB_PROFIL.displayName.charAt(0),
    timestamp: '1j',
    content:
      'Petite question à la team : on part sur un refresh token dans un cookie httpOnly ou on garde tout en localStorage ? Sécurité > confort ?',
    likes: 9,
    comments: 11,
    reposts: 1,
    views: 620,
  },
]

export default function ProfilPage() {
  // isOwner=true : /profil affiche toujours l'utilisateur courant.
  // TODO (issue auth) : déduire l'utilisateur courant du JWT.
  return <ProfilView profil={STUB_PROFIL} posts={STUB_POSTS} isOwner />
}
