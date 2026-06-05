import { Search } from 'lucide-react'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'

interface SuggestedUser {
  name: string
  handle: string
  initials: string
}

const SUGGESTED_USERS: SuggestedUser[] = [
  { name: 'Zaid', handle: '@zaid_dev', initials: 'Z' },
  { name: 'Perujan', handle: '@perujan', initials: 'P' },
  { name: 'Candis', handle: '@candis', initials: 'C' },
  { name: 'Théo', handle: '@theo_tech', initials: 'T' },
]

const TRENDS = [
  { category: 'Technologie', topic: '#Microservices', posts: '12,4 K posts' },
  { category: 'Dev', topic: '#NextJS', posts: '8,1 K posts' },
  { category: 'Cloud', topic: '#Docker', posts: '5,6 K posts' },
]

export function SidebarRight() {
  return (
    <aside className="sticky top-0 hidden h-screen w-[350px] flex-col gap-4 overflow-y-auto px-4 py-4 xl:flex">
      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <input
          type="search"
          placeholder="Rechercher"
          disabled
          className="glass w-full rounded-full border py-2.5 pl-10 pr-4 text-sm backdrop-blur placeholder:text-muted-foreground focus:border-[#5B6CFF] focus:bg-white focus:outline-none disabled:cursor-not-allowed dark:focus:bg-white/10"
        />
      </div>

      {/* Tendances */}
      <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
        <h2 className="brand-text px-4 py-3 text-xl font-bold">
          Tendances
        </h2>
        {TRENDS.map((trend) => (
          <div
            key={trend.topic}
            className="flex cursor-not-allowed flex-col gap-0.5 px-4 py-3 transition-colors hover:bg-accent"
          >
            <span className="text-xs text-muted-foreground">{trend.category} · Tendance</span>
            <span className="font-bold text-foreground">{trend.topic}</span>
            <span className="text-xs text-muted-foreground">{trend.posts}</span>
          </div>
        ))}
      </div>

      {/* Qui suivre */}
      <div className="glass overflow-hidden rounded-[24px] border backdrop-blur-xl">
        <h2 className="brand-text px-4 py-3 text-xl font-bold">
          Qui suivre
        </h2>
        {SUGGESTED_USERS.map((user) => (
          <div
            key={user.handle}
            className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-accent"
          >
            <Avatar className="h-10 w-10 shrink-0">
              <AvatarFallback>{user.initials}</AvatarFallback>
            </Avatar>
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-sm font-bold">{user.name}</span>
              <span className="truncate text-sm text-muted-foreground">{user.handle}</span>
            </div>
            {/* TODO (issue profil) : brancher l'action "suivre" */}
            <Button
              variant="default"
              size="sm"
              className="rounded-full bg-slate-950 font-bold text-white dark:bg-white dark:text-slate-950"
              disabled
            >
              Suivre
            </Button>
          </div>
        ))}
      </div>
    </aside>
  )
}
