'use client'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'

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
    <div className="glass overflow-hidden rounded-[24px] border p-4 backdrop-blur-xl">
      <div className="mb-3">
        <p className="brand-text text-lg font-bold">{t('explorer.filter_title')}</p>
        <p className="mt-1 text-xs text-muted-foreground">{t('explorer.filters_hint')}</p>
      </div>
      <FilterControls
        showPublications={showPublications}
        showUsers={showUsers}
        onTogglePublications={onTogglePublications}
        onToggleUsers={onToggleUsers}
      />
    </div>
  )
}

export function FilterControls({
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
    <div className="flex flex-col" aria-label={t('explorer.filters_hint')}>
      <FilterRow
        label={t('explorer.filter_publications')}
        checked={showPublications}
        locked={showPublications && !showUsers}
        onChange={onTogglePublications}
        filterType="publications"
      />
      <div className="border-t border-border/50" />
      <FilterRow
        label={t('explorer.filter_users')}
        checked={showUsers}
        locked={showUsers && !showPublications}
        onChange={onToggleUsers}
        filterType="users"
      />
    </div>
  )
}

/**
 * Filtres mobiles : deux pastilles toggle toujours visibles (Publications /
 * Utilisateurs). Au moins une doit rester active ; tenter de désactiver la
 * dernière active déclenche un toast explicatif.
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

function FilterRow({
  label,
  checked,
  locked,
  onChange,
  filterType,
}: {
  label: string
  checked: boolean
  /** Dernier filtre actif : grisé, mais cliquable pour afficher l'avertissement. */
  locked: boolean
  onChange: () => void | boolean
  filterType?: 'publications' | 'users'
}) {
  const t = useT()
  const { toast } = useToast()

  const handleChange = () => {
    const success = onChange()
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
    <label
      className={cn(
        'flex cursor-pointer items-center justify-between px-4 py-3 transition-colors hover:bg-accent',
        locked && 'opacity-60',
      )}
    >
      <span className="text-sm font-medium text-foreground">{label}</span>
      <input type="checkbox" checked={checked} onChange={handleChange} className="sr-only" />
      <CircleIndicator checked={checked} />
    </label>
  )
}

function CircleIndicator({ checked }: { checked: boolean }) {
  return (
    <span
      className={cn(
        'flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 transition-colors',
        checked
          ? 'border-[#5B6CFF] bg-[#5B6CFF] dark:border-[#9aa6ff] dark:bg-[#9aa6ff]'
          : 'border-muted-foreground/50 bg-transparent',
      )}
    >
      {checked && <span className="h-2 w-2 rounded-full bg-white" />}
    </span>
  )
}
