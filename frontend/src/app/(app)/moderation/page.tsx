import { Shield } from 'lucide-react'

// TODO : implémenter le panneau de modération (rôles moderator + administrator)
//   - lister les posts signalés (via Gateway -> post-service)
//   - actions : masquer / supprimer un post, avertir un utilisateur
//   - garde d'accès : refuser si le rôle n'est pas moderator/administrator
export default function ModerationPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b border-[#D9C6FF]/70 bg-gradient-to-r from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-xl font-bold text-transparent">
          Modération
        </h1>
      </div>

      <div className="mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border border-white/55 bg-white/68 px-8 py-16 text-center shadow-[0_18px_54px_rgba(91,108,255,0.12)] backdrop-blur-xl">
        <Shield className="h-10 w-10 text-[#5B6CFF]" aria-hidden />
        <h2 className="text-lg font-bold text-slate-950">Centre de modération</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Les signalements et les actions de modération apparaîtront ici.
        </p>
      </div>
    </div>
  )
}
