'use client'

import { useState } from 'react'
import { Image as ImageIcon, Smile, BarChart2 } from 'lucide-react'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

const MAX_CHARS = 280

export function CreatePost() {
  const [content, setContent] = useState('')
  const remaining = MAX_CHARS - content.length
  const isEmpty = content.trim().length === 0
  const isOver = remaining < 0

  return (
    <div className="flex gap-3 border-b px-4 py-3">
      <Avatar className="mt-1 h-10 w-10 shrink-0">
        {/* TODO (issue auth) : avatar de l'utilisateur courant */}
        <AvatarFallback>U</AvatarFallback>
      </Avatar>

      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Quoi de neuf ?"
          rows={3}
          className="w-full resize-none bg-transparent text-xl placeholder:text-muted-foreground focus:outline-none"
        />

        <Separator />

        {/* Toolbar */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1 text-primary">
            <ActionIcon icon={ImageIcon} label="Ajouter une image" />
            <ActionIcon icon={Smile} label="Ajouter un emoji" />
            <ActionIcon icon={BarChart2} label="Ajouter un sondage" />
          </div>

          <div className="flex items-center gap-3">
            {/* Compteur de caractères */}
            {content.length > 0 && (
              <span
                className={
                  isOver
                    ? 'text-sm font-bold text-destructive'
                    : remaining <= 20
                      ? 'text-sm text-amber-500'
                      : 'text-sm text-muted-foreground'
                }
              >
                {remaining}
              </span>
            )}

            {/* TODO (issue post) : brancher l'envoi vers POST /posts via l'API Gateway */}
            <Button
              size="sm"
              className="rounded-full font-bold"
              disabled={isEmpty || isOver}
            >
              Poster
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function ActionIcon({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <button
      type="button"
      aria-label={label}
      className="rounded-full p-2 transition-colors hover:bg-primary/10"
    >
      <Icon className="h-5 w-5" />
    </button>
  )
}
