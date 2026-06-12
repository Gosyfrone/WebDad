'use client'

import { MoreHorizontal } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useT } from '@/components/language-provider'
import {
  DropdownMenu,
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
    <div className="flex flex-col" aria-label={t('explorer.filters_hint')}>
      <FilterRow
        label={t('explorer.filter_publications')}
        checked={showPublications}
        disabled={showPublications && !showUsers}
        onChange={onTogglePublications}
      />
      <div className="border-t border-border/50" />
      <FilterRow
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
      <DropdownMenuContent align="end" className="w-64 overflow-hidden rounded-2xl p-0">
        <DropdownMenuLabel className="px-4 py-3 text-base font-bold">
          {t('explorer.filter_title')}
        </DropdownMenuLabel>
        <DropdownMenuSeparator className="m-0" />
        <MobileFilterRow
          label={t('explorer.filter_publications')}
          checked={showPublications}
          disabled={showPublications && !showUsers}
          onChange={onTogglePublications}
        />
        <DropdownMenuSeparator className="m-0" />
        <MobileFilterRow
          label={t('explorer.filter_users')}
          checked={showUsers}
          disabled={showUsers && !showPublications}
          onChange={onToggleUsers}
        />
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

function FilterRow({
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
        'flex cursor-pointer items-center justify-between px-4 py-3 transition-colors hover:bg-accent',
        disabled && 'cursor-not-allowed opacity-60',
      )}
    >
      <span className="text-sm font-medium text-foreground">{label}</span>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={onChange} className="sr-only" />
      <CircleIndicator checked={checked} />
    </label>
  )
}

function MobileFilterRow({
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
    <button
      type="button"
      disabled={disabled}
      onClick={onChange}
      className={cn(
        'flex w-full items-center justify-between px-4 py-3 text-sm font-medium transition-colors hover:bg-accent',
        disabled && 'cursor-not-allowed opacity-60',
      )}
    >
      {label}
      <CircleIndicator checked={checked} />
    </button>
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
