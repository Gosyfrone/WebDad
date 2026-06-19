'use client'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'

/**
 * Carte de filtres desktop (colonne de droite). Compacte par conception : un
 * libellé court + les deux pastilles toggle. Garder un faible encombrement
 * vertical est essentiel — la colonne de droite est épinglée et un encart trop
 * haut repoussait « Tendances »/« Qui suivre » sous le pli lors d'une recherche.
 */
export function ExplorerFilterCard({
  showPublications,
  showUsers,
  onTogglePublications,
  onToggleUsers,
}: {
  showPublications: boolean
  showUsers: boolean
  onTogglePublications: () => void | boolean
  onToggleUsers: () => void | boolean
}) {
  const t = useT()
  return (
    <div className="glass overflow-hidden rounded-[24px] border px-4 py-3 backdrop-blur-xl">
      <p className="brand-text mb-2 text-sm font-bold">{t('explorer.filter_title')}</p>
      <MobileFilterButtons
        showPublications={showPublications}
        showUsers={showUsers}
        onTogglePublications={onTogglePublications}
        onToggleUsers={onToggleUsers}
      />
    </div>
  )
}

/**
 * Pastilles toggle des filtres (Publications / Utilisateurs), partagées par la
 * barre mobile de l'explorateur et la carte desktop de la colonne de droite.
 * Au moins une doit rester active ; tenter de désactiver la dernière active
 * déclenche un toast explicatif.
 */
export function MobileFilterButtons({
  showPublications,
  showUsers,
  onTogglePublications,
  onToggleUsers,
}: {
  showPublications: boolean
  showUsers: boolean
  onTogglePublications: () => void | boolean
  onToggleUsers: () => void | boolean
}) {
  const t = useT()
  const { toast } = useToast()

  const handle =
    (toggle: () => void | boolean, filterType: 'publications' | 'users') => () => {
      const success = toggle()
      if (success === false) {
        toast({
          title: t('explorer.filter_required'),
          description:
            filterType === 'publications'
              ? t('explorer.at_least_one_filter_publications')
              : t('explorer.at_least_one_filter_users'),
        })
      }
    }

  return (
    <div className="flex gap-2">
      <FilterChip
        label={t('explorer.filter_publications')}
        active={showPublications}
        onClick={handle(onTogglePublications, 'publications')}
      />
      <FilterChip
        label={t('explorer.filter_users')}
        active={showUsers}
        onClick={handle(onToggleUsers, 'users')}
      />
    </div>
  )
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'rounded-full border px-4 py-1.5 text-sm font-bold transition-colors',
        active
          ? 'border-transparent bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-white'
          : 'border-border bg-background text-muted-foreground hover:bg-accent',
      )}
    >
      {label}
    </button>
  )
}

