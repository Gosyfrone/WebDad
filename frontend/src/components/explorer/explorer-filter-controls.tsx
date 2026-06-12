'use client'

import { MoreHorizontal } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

export function ExplorerFilterCard({
  showPublications,
  showUsers,
  onTogglePublications,
  onToggleUsers,
}: {
  showPublications: boolean
  showUsers: boolean
  onTogglePublications: () => void
  onToggleUsers: () => void
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
  onTogglePublications: () => void
  onToggleUsers: () => void
}) {
  const t = useT()
  return (
    <div className="flex flex-wrap gap-2" aria-label={t('explorer.filters_hint')}>
      <FilterCheck
        label={t('explorer.filter_publications')}
        checked={showPublications}
        disabled={showPublications && !showUsers}
        onChange={onTogglePublications}
      />
      <FilterCheck
        label={t('explorer.filter_users')}
        checked={showUsers}
        disabled={showUsers && !showPublications}
        onChange={onToggleUsers}
      />
    </div>
  )
}

export function MobileFilterMenu({
  showPublications,
  showUsers,
  onTogglePublications,
  onToggleUsers,
}: {
  showPublications: boolean
  showUsers: boolean
  onTogglePublications: () => void
  onToggleUsers: () => void
}) {
  const t = useT()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <ButtonLikeDots label={t('explorer.filter_title')} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56 rounded-2xl">
        <DropdownMenuLabel>{t('explorer.filter_title')}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuCheckboxItem
          checked={showPublications}
          disabled={showPublications && !showUsers}
          onCheckedChange={onTogglePublications}
        >
          {t('explorer.filter_publications')}
        </DropdownMenuCheckboxItem>
        <DropdownMenuCheckboxItem
          checked={showUsers}
          disabled={showUsers && !showPublications}
          onCheckedChange={onToggleUsers}
        >
          {t('explorer.filter_users')}
        </DropdownMenuCheckboxItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ButtonLikeDots({ label }: { label: string }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className="grid h-9 w-9 place-items-center rounded-full border border-border bg-background text-muted-foreground shadow-sm transition hover:bg-accent hover:text-foreground"
    >
      <MoreHorizontal className="h-5 w-5" aria-hidden />
    </button>
  )
}

function FilterCheck({
  label,
  checked,
  disabled,
  onChange,
}: {
  label: string
  checked: boolean
  disabled: boolean
  onChange: () => void
}) {
  return (
    <label
      className={cn(
        'flex cursor-pointer items-center gap-2 rounded-full border px-3 py-2 text-xs font-bold transition-colors',
        checked
          ? 'border-[#5B6CFF] bg-[#5B6CFF]/10 text-[#5B6CFF] dark:border-[#9aa6ff] dark:bg-[#9aa6ff]/15 dark:text-[#c7ceff]'
          : 'border-border text-muted-foreground hover:bg-accent',
        disabled && 'cursor-not-allowed opacity-70',
      )}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={onChange}
        className="h-3.5 w-3.5 accent-[#5B6CFF]"
      />
      {label}
    </label>
  )
}
