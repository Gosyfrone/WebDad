import { Shield } from 'lucide-react'

// TODO : implémenter le panneau de modération (rôles moderator + administrator)
//   - lister les posts signalés (via Gateway -> post-service)
//   - actions : masquer / supprimer un post, avertir un utilisateur
//   - garde d'accès : refuser si le rôle n'est pas moderator/administrator
export default function ModerationPage() {
  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">Modération</h1>
      </div>

      <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
        <Shield className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <h2 className="text-lg font-bold text-foreground">Centre de modération</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Les signalements et les actions de modération apparaîtront ici.
        </p>
      </div>
    </div>
  )
}
