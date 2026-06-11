'use client'

import { useMemo, useState } from 'react'
import { Eye, EyeOff, KeyRound, Loader2, ShieldCheck, ShieldQuestion } from 'lucide-react'

import { cn } from '@/lib/utils'
import {
  estimateStrength,
  isPassphraseAcceptable,
  type StrengthLevel,
} from '@/lib/passphrase-strength'
import { IdentityLockedError, setupPassphrase, unlockWithPassphrase } from '@/lib/messages'
import { useLanguage } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface PassphraseGateProps {
  /** `setup` : 1ère définition · `unlock` : déblocage sur un nouvel appareil. */
  mode: 'setup' | 'unlock'
  /** Appelé après définition/déblocage réussi (identité disponible localement). */
  onUnlocked: () => void
}

/** Couleurs de la barre de complexité par niveau. */
const STRENGTH_COLOR: Record<StrengthLevel, string> = {
  empty: 'bg-transparent',
  weak: 'bg-red-500',
  fair: 'bg-amber-500',
  good: 'bg-lime-500',
  strong: 'bg-emerald-500',
}

/**
 * Écran de protection de la messagerie : tant que l'identité E2EE n'est pas
 * disponible sur cet appareil, la vue est floutée derrière cet overlay qui
 * demande de DÉFINIR (1er usage) ou DÉBLOQUER (autre appareil) une phrase de
 * passe. La phrase ne quitte jamais le navigateur (cf. lib/key-backup.ts).
 */
export function PassphraseGate({ mode, onUnlocked }: PassphraseGateProps) {
  const { t } = useLanguage()
  const [value, setValue] = useState('')
  const [confirm, setConfirm] = useState('')
  const [reveal, setReveal] = useState(false)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isSetup = mode === 'setup'
  const strength = useMemo(() => estimateStrength(value), [value])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (isSetup) {
      if (!isPassphraseAcceptable(value)) {
        setError(t('messages.passphrase.too_weak'))
        return
      }
      if (value !== confirm) {
        setError(t('messages.passphrase.mismatch'))
        return
      }
    } else if (!value) {
      return
    }

    setPending(true)
    try {
      if (isSetup) {
        await setupPassphrase(value)
      } else {
        await unlockWithPassphrase(value)
      }
      onUnlocked()
    } catch (err) {
      if (isSetup) {
        setError(t('messages.passphrase.setup_failed'))
      } else {
        // Mauvaise passphrase → échec d'authentification au déballage.
        const wrong = !(err instanceof IdentityLockedError) && err instanceof Error
        setError(wrong ? t('messages.passphrase.wrong') : t('messages.passphrase.unlock_failed'))
      }
    } finally {
      setPending(false)
    }
  }

  const strengthLabel =
    strength.level === 'empty' || strength.level === 'weak'
      ? t('messages.passphrase.strength.weak')
      : strength.level === 'fair'
        ? t('messages.passphrase.strength.fair')
        : strength.level === 'good'
          ? t('messages.passphrase.strength.good')
          : t('messages.passphrase.strength.strong')

  return (
    <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/60 p-4 backdrop-blur-md">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-md rounded-2xl border bg-card p-6 shadow-xl"
      >
        <div className="mb-4 flex items-center gap-3">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            {isSetup ? <ShieldCheck className="h-6 w-6" /> : <ShieldQuestion className="h-6 w-6" />}
          </span>
          <div>
            <h2 className="text-lg font-semibold">
              {t(isSetup ? 'messages.passphrase.setup_title' : 'messages.passphrase.unlock_title')}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t(isSetup ? 'messages.passphrase.setup_desc' : 'messages.passphrase.unlock_desc')}
            </p>
          </div>
        </div>

        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-sm font-medium" htmlFor="passphrase">
              {t('messages.passphrase.field')}
            </label>
            <div className="relative">
              <Input
                id="passphrase"
                type={reveal ? 'text' : 'password'}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder={t('messages.passphrase.field_placeholder')}
                autoComplete={isSetup ? 'new-password' : 'current-password'}
                autoFocus
                disabled={pending}
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setReveal((r) => !r)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                aria-label={t(reveal ? 'messages.passphrase.hide' : 'messages.passphrase.show')}
                tabIndex={-1}
              >
                {reveal ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>

          {isSetup && (
            <>
              {/* Barre de complexité (4 segments). */}
              <div className="space-y-1">
                <div className="flex gap-1">
                  {[1, 2, 3, 4].map((seg) => (
                    <span
                      key={seg}
                      className={cn(
                        'h-1.5 flex-1 rounded-full transition-colors',
                        seg <= strength.score ? STRENGTH_COLOR[strength.level] : 'bg-muted',
                      )}
                    />
                  ))}
                </div>
                {value && (
                  <p className="text-xs text-muted-foreground">{strengthLabel}</p>
                )}
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium" htmlFor="passphrase-confirm">
                  {t('messages.passphrase.confirm_field')}
                </label>
                <Input
                  id="passphrase-confirm"
                  type={reveal ? 'text' : 'password'}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  autoComplete="new-password"
                  disabled={pending}
                />
              </div>

              <p className="flex items-start gap-2 rounded-lg bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
                <KeyRound className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                {t('messages.passphrase.warning')}
              </p>
            </>
          )}

          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

          <Button type="submit" className="w-full" disabled={pending || !value}>
            {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {pending
              ? t('messages.passphrase.working')
              : t(isSetup ? 'messages.passphrase.setup_cta' : 'messages.passphrase.unlock_cta')}
          </Button>
        </div>
      </form>
    </div>
  )
}
