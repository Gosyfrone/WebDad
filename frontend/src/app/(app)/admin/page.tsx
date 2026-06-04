import { Settings2 } from 'lucide-react'

// TODO : implémenter le panneau d'administration (rôle administrator)
//   - lister / gérer les utilisateurs (via Gateway -> user-service)
//   - changer les rôles (user / moderator / administrator)
//   - garde d'accès : refuser si le rôle n'est pas administrator
export default function AdminPage() {
  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">Administration</h1>
      </div>

      <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
        <Settings2 className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <h2 className="text-lg font-bold text-foreground">Administration Breezy</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          La gestion des utilisateurs et des rôles arrivera ici.
        </p>
      </div>
    </div>
  )
}
