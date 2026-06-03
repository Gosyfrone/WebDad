import { Settings2 } from 'lucide-react'

// TODO : implémenter le panneau d'administration (rôle administrator)
//   - lister / gérer les utilisateurs (via Gateway -> user-service)
//   - changer les rôles (user / moderator / administrator)
//   - garde d'accès : refuser si le rôle n'est pas administrator
export default function AdminPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b border-white/50 bg-white/70 px-4 py-3 backdrop-blur-2xl lg:sticky lg:top-0 lg:block">
        <h1 className="bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-xl font-bold text-transparent">
          Administration
        </h1>
      </div>

      <div className="mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border border-white/55 bg-white/68 px-8 py-16 text-center shadow-[0_18px_54px_rgba(91,108,255,0.12)] backdrop-blur-xl">
        <Settings2 className="h-10 w-10 text-[#5B6CFF]" aria-hidden />
        <h2 className="text-lg font-bold text-slate-950">Administration Breezy</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          La gestion des utilisateurs et des rôles arrivera ici.
        </p>
      </div>
    </div>
  )
}
