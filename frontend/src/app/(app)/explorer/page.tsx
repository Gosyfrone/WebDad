import { Search } from 'lucide-react'

// TODO (issue post/profil) : recherche de posts et de comptes via l'API Gateway.
export default function ExplorerPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b bg-background/80 px-4 py-3 backdrop-blur lg:sticky lg:top-0 lg:block">
        <h1 className="text-xl font-bold">Explorer</h1>
      </div>
      <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
        <Search className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h2 className="text-lg font-bold">Rechercher sur Breezy</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          La recherche de comptes et de posts arrivera ici une fois l&apos;API branchée.
        </p>
      </div>
    </div>
  )
}
