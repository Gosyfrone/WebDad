import { Search } from 'lucide-react'

// TODO (issue post/profil) : recherche de posts et de comptes via l'API Gateway.
export default function ExplorerPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b border-white/50 bg-white/70 px-4 py-3 backdrop-blur-2xl lg:sticky lg:top-0 lg:block">
        <h1 className="bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-xl font-bold text-transparent">
          Explorer
        </h1>
      </div>
      <div className="mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border border-white/55 bg-white/68 px-8 py-16 text-center shadow-[0_18px_54px_rgba(91,108,255,0.12)] backdrop-blur-xl">
        <Search className="h-10 w-10 text-[#5B6CFF]" aria-hidden />
        <h2 className="text-lg font-bold text-slate-950">Rechercher sur Breezy</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          La recherche de comptes et de posts arrivera ici une fois l&apos;API branchée.
        </p>
      </div>
    </div>
  )
}
