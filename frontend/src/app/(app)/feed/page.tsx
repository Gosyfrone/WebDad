// TODO : implémenter le fil principal
//   - GET ${NEXT_PUBLIC_API_URL}/posts (via Gateway -> post-service)
//   - composant de création de post (limité à 280 caractères)
//   - liste paginée / scroll infini
//   - actions : like, commenter, signaler
export default function FeedPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Fil d&apos;actualité</h1>
    </main>
  )
}
