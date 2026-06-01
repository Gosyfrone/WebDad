// TODO : implémenter le panneau d'administration (rôle administrator)
//   - lister / gérer les utilisateurs (via Gateway -> user-service)
//   - changer les rôles (user / moderator / administrator)
//   - garde d'accès : refuser si le rôle n'est pas administrator
export default function AdminPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Administration</h1>
    </main>
  )
}
