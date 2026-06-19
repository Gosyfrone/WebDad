'use client'

import { useEffect, useState } from 'react'
import { Eye, Lock, ShieldAlert } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { getMyProfil, saveMyNsfw } from '@/lib/profil-client'
import { cn } from '@/lib/utils'

/**
 * Carte de préférence « contenu sensible (NSFW) ». Le toggle (style Apple)
 * pilote l'affichage du contenu marqué NSFW. Pour un MINEUR (isAdult=false,
 * calculé serveur depuis la date de naissance), la carte est grisée et
 * verrouillée : elle se débloque automatiquement à la majorité (le serveur
 * reste autoritaire — un mineur ne voit jamais le NSFW même si la préférence
 * stockée vaut true).
 */
export function NsfwSettings() {
  const t = useT()
  const { toast } = useToast()
  const [nsfwEnabled, setNsfwEnabled] = useState(true)
  const [isAdult, setIsAdult] = useState(true)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false

    async function loadNsfw() {
      try {
        const profil = await getMyProfil()
        if (!cancelled) {
          setNsfwEnabled(profil.nsfwEnabled)
          setIsAdult(profil.isAdult)
        }
      } catch {
        if (!cancelled) {
          toast({ title: t('nsfw.load_failed'), variant: 'destructive' })
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadNsfw()
    return () => {
      cancelled = true
    }
  }, [toast, t])

  async function updateNsfw(next: boolean) {
    if (next === nsfwEnabled || saving || !isAdult) return
    const previous = nsfwEnabled
    setNsfwEnabled(next)
    setSaving(true)
    try {
      const updated = await saveMyNsfw(next)
      setNsfwEnabled(updated.nsfwEnabled)
      setIsAdult(updated.isAdult)
      toast({ title: t('nsfw.saved') })
    } catch {
      setNsfwEnabled(previous)
      toast({ title: t('nsfw.save_failed'), variant: 'destructive' })
    } finally {
      setSaving(false)
    }
  }

  // Un mineur ne peut pas activer le NSFW : on affiche le toggle « éteint »,
  // grisé, avec la note de verrouillage (débloqué à la majorité côté serveur).
  const checked = isAdult && nsfwEnabled
  const disabled = loading || saving || !isAdult

  return (
    <div className="panel rounded-xl border px-3 py-3 shadow-sm">
      <div className="flex items-center justify-between gap-3 rounded-lg px-1 py-1">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <ShieldAlert className="h-4 w-4" aria-hidden />
          </span>
          <div className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              {t('nsfw.toggle_label')}
            </span>
            <span className="mt-0.5 block text-xs leading-4 text-muted-foreground">
              {isAdult ? t('nsfw.toggle_desc') : t('nsfw.minor_locked')}
            </span>
          </div>
        </div>

        <button
          type="button"
          role="switch"
          aria-checked={checked}
          aria-label={t('nsfw.toggle_aria')}
          disabled={disabled}
          onClick={() => updateNsfw(!nsfwEnabled)}
          className={cn(
            'relative inline-flex h-8 w-[3.25rem] shrink-0 items-center rounded-full border transition-colors',
            checked
              ? 'border-primary bg-primary'
              : 'border-slate-400 bg-slate-300 dark:border-slate-500 dark:bg-slate-700',
            disabled && 'cursor-not-allowed opacity-60',
          )}
        >
          <span
            className={cn(
              'absolute left-1 z-0 flex h-6 w-6 items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/10 transition-transform dark:bg-slate-100',
              checked ? 'translate-x-5' : 'translate-x-0',
            )}
          >
            {!isAdult ? (
              <Lock className="h-3.5 w-3.5 text-slate-500" aria-hidden />
            ) : checked ? (
              <Eye className="h-3.5 w-3.5 text-primary" aria-hidden />
            ) : null}
          </span>
        </button>
      </div>
    </div>
  )
}
