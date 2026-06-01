// TODO : implémenter la page de profil utilisateur
//   - GET ${NEXT_PUBLIC_API_URL}/profils/me (via Gateway -> profil-service)
//   - afficher avatar, bio, compteurs (followers / following)
//   - bouton "éditer" → formulaire de mise à jour (bio, avatarUrl)
//   - lister les posts de l'utilisateur (via post-service)
export default function ProfilPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Profil</h1>
    </main>
  )
}
