import { Bell } from 'lucide-react'

// TODO (issue post/profil) : flux de notifications (likes, abonnements, mentions).
export default function NotificationsPage() {
  return (
    <div className="flex flex-col">
      <div className="z-10 hidden border-b bg-background/80 px-4 py-3 backdrop-blur lg:sticky lg:top-0 lg:block">
        <h1 className="text-xl font-bold">Notifications</h1>
      </div>
      <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
        <Bell className="h-10 w-10 text-muted-foreground" aria-hidden />
        <h2 className="text-lg font-bold">Rien pour le moment</h2>
        <p className="max-w-sm text-sm text-muted-foreground">
          Vos notifications (likes, abonnements, mentions) apparaîtront ici.
        </p>
      </div>
    </div>
  )
}
