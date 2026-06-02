import { Settings } from 'lucide-react'

// TODO : préférences du compte (sélecteur de thème/accent cf. lib/themes.ts, langue, confidentialité…).
export default function ParametresPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b bg-background/80 px-4 py-3 backdrop-blur lg:sticky lg:top-0 lg:block">
        <h1 className="text-xl font-bold">Paramètres</h1>
      </div>
      <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
        <Settings className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h2 className="text-lg font-bold">Paramètres du compte</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Le thème, la langue et les préférences de confidentialité arriveront ici.
        </p>
      </div>
    </div>
  )
}
