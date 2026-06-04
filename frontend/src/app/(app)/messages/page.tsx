import { Mail } from 'lucide-react'

// TODO (issue post/profil) : messagerie privée (DMs) via l'API Gateway.
export default function MessagesPage() {
  return (
    <div className="flex flex-col">
      <div className="panel z-10 hidden border-b px-4 py-3 lg:sticky lg:top-0 lg:block">
        <h1 className="brand-text text-xl font-bold">Messages</h1>
      </div>
      <div className="glass mx-4 mt-6 flex flex-col items-center gap-2 rounded-[26px] border px-8 py-16 text-center backdrop-blur-xl">
        <Mail className="h-10 w-10 text-[#5B6CFF] dark:text-[#9aa6ff]" aria-hidden />
        <h2 className="text-lg font-bold text-foreground">Aucune conversation</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Vos messages privés apparaîtront ici une fois la messagerie disponible.
        </p>
      </div>
    </div>
  )
}
