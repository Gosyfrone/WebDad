'use client'

import { useEffect, useState } from 'react'
import { Loader2, Search } from 'lucide-react'

import { searchGifs, type GiphyGif } from '@/lib/giphy'
import { useT } from '@/components/language-provider'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

/**
 * Sélecteur de GIF GIPHY (recherche + grille), partagé par le composer de post et
 * le composer de commentaire/réponse. Le parent reçoit le GIF choisi via `onSelect`
 * et se charge de la capture serveur (→ MinIO) + des toasts ; ici on ne fait que
 * rechercher et fermer la popover à la sélection.
 */
export function GifPicker({
  disabled,
  onSelect,
}: {
  disabled?: boolean
  onSelect: (gif: GiphyGif) => void
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [gifs, setGifs] = useState<GiphyGif[]>([])
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const handle = window.setTimeout(() => {
      setLoading(true)
      setFailed(false)
      searchGifs(query)
        .then((items) => {
          if (cancelled) return
          setGifs(items)
        })
        .catch(() => {
          if (cancelled) return
          setFailed(true)
          setGifs([])
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }, query.trim() ? 250 : 0)

    return () => {
      cancelled = true
      window.clearTimeout(handle)
    }
  }, [open, query])

  function choose(gif: GiphyGif) {
    // Le toast succès/échec est émis par le parent une fois la capture serveur
    // terminée : on ferme juste la popover ici.
    onSelect(gif)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={(next) => !disabled && setOpen(next)}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={t('composer.add_gif')}
          disabled={disabled}
          className="rounded-full px-2 py-2 text-sm font-black leading-none transition-colors hover:bg-primary/10 disabled:opacity-40"
        >
          GIF
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[min(22rem,calc(100vw-2rem))] p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('composer.gif_search')}
            className="h-10 w-full rounded-md border border-border bg-background py-2 pl-9 pr-3 text-sm outline-none focus:border-primary"
            autoFocus
          />
        </div>

        <div className="mt-3 h-72 overflow-y-auto pr-1">
          {loading && gifs.length === 0 ? (
            <div className="grid h-full place-items-center text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : failed ? (
            <div className="grid h-full place-items-center px-6 text-center text-sm text-muted-foreground">
              {t('composer.gif_failed')}
            </div>
          ) : gifs.length === 0 ? (
            <div className="grid h-full place-items-center px-6 text-center text-sm text-muted-foreground">
              {t('composer.gif_empty')}
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2">
              {gifs.map((gif) => (
                <button
                  key={gif.id}
                  type="button"
                  onClick={() => choose(gif)}
                  className="group overflow-hidden rounded-md border border-border bg-muted transition hover:border-primary"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={gif.previewUrl}
                    alt={gif.title}
                    loading="lazy"
                    className="aspect-square h-full w-full object-cover transition group-hover:scale-[1.03]"
                  />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-2 text-right text-[10px] font-bold uppercase tracking-wide text-muted-foreground">
          GIPHY
        </div>
      </PopoverContent>
    </Popover>
  )
}
