import { Search } from 'lucide-react'

// TODO (issue post/profil) : recherche de posts et de comptes via l'API Gateway.
export default function ExplorerPage() {
  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">Explorer</h1>
      </div>
      <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
        <Search className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <h2 className="text-lg font-bold text-foreground">Rechercher sur Breezy</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          La recherche de comptes et de posts arrivera ici une fois l&apos;API branchée.
        </p>
      </div>
    </div>
  )
}
