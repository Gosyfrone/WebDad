import { CreatePost } from '@/components/feed/create-post'
import { PostCard, type PostCardProps } from '@/components/feed/post-card'

// Données fictives pour le squelette (remplacées par l'API dans l'issue post)
const STUB_POSTS: PostCardProps[] = [
  {
    id: '1',
    name: 'Zaid',
    handle: '@zaid_dev',
    initials: 'Z',
    timestamp: '2h',
    content:
      'Première journée sur le projet Breezy 🚀 Architecture microservices en Go, on attaque l\'auth-service ce soir !',
    likes: 24,
    comments: 3,
    reposts: 5,
    views: 1200,
  },
  {
    id: '2',
    name: 'Perujan',
    handle: '@perujan',
    initials: 'P',
    timestamp: '4h',
    content:
      'Docker Compose configuré pour tous les services. Plus aucun conflit de ports grâce à la plage 8080–8084 côté backend et 3000 côté frontend. Propre.',
    likes: 41,
    comments: 7,
    reposts: 12,
    views: 3400,
  },
  {
    id: '3',
    name: 'Candis',
    handle: '@candis',
    initials: 'C',
    timestamp: '6h',
    content:
      'Next.js 14 App Router + Tailwind + shadcn/ui → combo parfait pour aller vite sans sacrifier la qualité du design.',
    likes: 88,
    comments: 14,
    reposts: 22,
    views: 8700,
  },
  {
    id: '4',
    name: 'Théo',
    handle: '@theo_tech',
    initials: 'T',
    timestamp: '1j',
    content:
      'Le JWT va passer dans le header Authorization sur toutes les routes protégées du Gateway. Simple, standard, ça colle aux exigences de la grille d\'eval.',
    likes: 17,
    comments: 2,
    reposts: 3,
    views: 940,
  },
  {
    id: '5',
    name: 'Zaid',
    handle: '@zaid_dev',
    initials: 'Z',
    timestamp: '1j',
    content:
      'Petite question à la team : on part sur un refresh token dans un cookie httpOnly ou on garde tout en localStorage ? Sécurité > confort ?',
    likes: 9,
    comments: 11,
    reposts: 1,
    views: 620,
  },
]

export default function FeedPage() {
  return (
    <div className="flex flex-col">
      {/* En-tête sticky */}
      <div className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
        <h1 className="px-4 py-3 text-xl font-bold">Fil d&apos;actualité</h1>
        {/* Onglets Pour toi / Abonnements */}
        <div className="flex">
          <button className="flex-1 border-b-2 border-primary py-3 text-sm font-bold">
            Pour toi
          </button>
          <button className="flex-1 py-3 text-sm font-normal text-muted-foreground transition-colors hover:bg-muted/30">
            Abonnements
          </button>
        </div>
      </div>

      {/* Zone de création de post */}
      <CreatePost />

      {/* Liste des posts */}
      <div className="divide-y">
        {STUB_POSTS.map((post) => (
          <PostCard key={post.id} {...post} />
        ))}
      </div>
    </div>
  )
}
