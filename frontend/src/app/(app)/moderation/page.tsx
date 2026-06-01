// TODO : implémenter le panneau de modération (rôles moderator + administrator)
//   - lister les posts signalés (via Gateway -> post-service)
//   - actions : masquer / supprimer un post, avertir un utilisateur
//   - garde d'accès : refuser si le rôle n'est pas moderator/administrator
export default function ModerationPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Modération</h1>
    </main>
  )
}
