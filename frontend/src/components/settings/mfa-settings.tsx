'use client'

import * as React from 'react'
import { CircleCheck, KeyRound, Loader2, ShieldCheck } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  disableMfa,
  enableMfa,
  getMfaStatus,
  MfaError,
  setupMfa,
  type MfaSetup,
  type MfaStatus,
} from '@/lib/mfa'

type Notice = { kind: 'error' | 'success'; text: string }
type Mode = 'idle' | 'setup' | 'disabling'

export function MfaSettings() {
  const t = useT()
  const [status, setStatus] = React.useState<MfaStatus>()
  const [mode, setMode] = React.useState<Mode>('idle')
  const [setup, setSetup] = React.useState<MfaSetup>()
  const [code, setCode] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [busy, setBusy] = React.useState(false)
  const [notice, setNotice] = React.useState<Notice>()

  React.useEffect(() => {
    let active = true
    getMfaStatus()
      .then((s) => {
        if (active) setStatus(s)
      })
      .catch(() => {
        if (active) setStatus({ enabled: false, configured: false })
      })
    return () => {
      active = false
    }
  }, [])

  const resetForms = () => {
    setSetup(undefined)
    setCode('')
    setPassword('')
  }

  const cancel = () => {
    setMode('idle')
    resetForms()
    setNotice(undefined)
  }

  // Clic sur l'interrupteur : ouvre le flux d'activation (off→on) ou de
  // désactivation (on→off). Re-cliquer pendant un flux ouvert = annuler. L'état
  // « coché » ne bascule réellement qu'après confirmation (code/mot de passe).
  const onToggle = async () => {
    if (!status?.configured || busy) return
    if (mode !== 'idle') {
      cancel()
      return
    }
    setNotice(undefined)
    if (status.enabled) {
      resetForms()
      setMode('disabling')
      return
    }
    // Activation : on récupère le secret + QR, puis on demande le code.
    setBusy(true)
    try {
      const data = await setupMfa()
      setSetup(data)
      setMode('setup')
    } catch (error) {
      const c = error instanceof MfaError ? error.code : undefined
      setNotice({
        kind: 'error',
        text:
          c === 'mfa_not_configured'
            ? t('settings.mfa.not_configured')
            : t('settings.mfa.error_generic'),
      })
    } finally {
      setBusy(false)
    }
  }

  const confirmEnable = async (event: React.FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setNotice(undefined)
    try {
      await enableMfa(code.trim())
      setStatus({ enabled: true, configured: true })
      setMode('idle')
      resetForms()
      setNotice({ kind: 'success', text: t('settings.mfa.success_enabled') })
    } catch (error) {
      const c = error instanceof MfaError ? error.code : undefined
      setNotice({
        kind: 'error',
        text:
          c === 'invalid_mfa_code'
            ? t('settings.mfa.error_code')
            : t('settings.mfa.error_generic'),
      })
    } finally {
      setBusy(false)
    }
  }

  const confirmDisable = async (event: React.FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setNotice(undefined)
    try {
      await disableMfa(code.trim() ? { code: code.trim() } : { password })
      setStatus({ enabled: false, configured: true })
      setMode('idle')
      resetForms()
      setNotice({ kind: 'success', text: t('settings.mfa.success_disabled') })
    } catch (error) {
      const c = error instanceof MfaError ? error.code : undefined
      setNotice({
        kind: 'error',
        text:
          c === 'invalid_mfa_code'
            ? t('settings.mfa.error_code')
            : t('settings.mfa.error_generic'),
      })
    } finally {
      setBusy(false)
    }
  }

  const enabled = status?.enabled ?? false
  const switchDisabled = status === undefined || status.configured === false || busy

  return (
    <div className="grid gap-4 px-4 py-5 md:grid-cols-[minmax(0,1fr)_minmax(280px,380px)]">
      <div className="flex items-start gap-3">
        <KeyRound className="mt-1 h-5 w-5 shrink-0 text-primary" aria-hidden />
        <div className="min-w-0">
          <h3 className="font-semibold">{t('settings.mfa.title')}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{t('settings.mfa.desc')}</p>
        </div>
      </div>

      <div className="panel space-y-3 rounded-xl border p-4 shadow-sm">
        {/* Interrupteur on/off (style iOS, aligné sur visibility-settings). */}
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              {t('settings.mfa.switch_label')}
            </span>
            <span className="mt-0.5 block text-xs leading-4 text-muted-foreground">
              {status?.configured === false
                ? t('settings.mfa.not_configured')
                : enabled
                  ? t('settings.mfa.state_on')
                  : t('settings.mfa.state_off')}
            </span>
          </div>

          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            aria-label={t('settings.mfa.switch_label')}
            disabled={switchDisabled}
            onClick={onToggle}
            className={cn(
              'relative inline-flex h-8 w-[3.25rem] shrink-0 items-center rounded-full border transition-colors',
              enabled
                ? 'border-primary bg-primary'
                : 'border-slate-400 bg-slate-300 dark:border-slate-500 dark:bg-slate-700',
              switchDisabled && 'cursor-not-allowed opacity-60',
            )}
          >
            <span
              className={cn(
                'absolute left-1 z-0 flex h-6 w-6 items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/10 transition-transform dark:bg-slate-100',
                enabled ? 'translate-x-5' : 'translate-x-0',
              )}
            >
              {busy ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" aria-hidden />
              ) : enabled ? (
                <ShieldCheck className="h-3.5 w-3.5 text-primary" aria-hidden />
              ) : null}
            </span>
          </button>
        </div>

        {/* Panneau d'activation : QR + clé manuelle + code. */}
        {mode === 'setup' && setup ? (
          <form className="space-y-3 border-t pt-3" onSubmit={confirmEnable}>
            <p className="text-sm text-muted-foreground">{t('settings.mfa.setup_intro')}</p>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={setup.qr_data_uri}
              alt={t('settings.mfa.qr_alt')}
              className="mx-auto h-44 w-44 rounded-lg bg-white p-2"
            />
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">{t('settings.mfa.secret_label')}</p>
              <code className="block break-all rounded-md bg-muted px-2 py-1.5 text-center text-sm font-medium tracking-wider">
                {setup.secret}
              </code>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="mfa-code">
                {t('settings.mfa.code_label')}
              </label>
              <Input
                id="mfa-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                placeholder="123456"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
              />
            </div>
            <NoticeMessage notice={notice} />
            <div className="flex gap-2">
              <Button type="button" variant="outline" className="flex-1" onClick={cancel} disabled={busy}>
                {t('settings.mfa.cancel')}
              </Button>
              <Button className="flex-1" disabled={busy || !code.trim()}>
                {busy ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                {t('settings.mfa.confirm')}
              </Button>
            </div>
          </form>
        ) : mode === 'disabling' ? (
          <form className="space-y-3 border-t pt-3" onSubmit={confirmDisable}>
            <p className="text-sm text-muted-foreground">{t('settings.mfa.disable_intro')}</p>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="mfa-disable-code">
                {t('settings.mfa.code_label')}
              </label>
              <Input
                id="mfa-disable-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                placeholder="123456"
                value={code}
                onChange={(e) => setCode(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="mfa-disable-pw">
                {t('settings.mfa.password_label')}
              </label>
              <Input
                id="mfa-disable-pw"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <NoticeMessage notice={notice} />
            <div className="flex gap-2">
              <Button type="button" variant="outline" className="flex-1" onClick={cancel} disabled={busy}>
                {t('settings.mfa.cancel')}
              </Button>
              <Button
                variant="destructive"
                className="flex-1"
                disabled={busy || (!code.trim() && !password)}
              >
                {busy ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                {t('settings.mfa.disable')}
              </Button>
            </div>
          </form>
        ) : (
          <NoticeMessage notice={notice} />
        )}
      </div>
    </div>
  )
}

function NoticeMessage({ notice }: { notice?: Notice }) {
  if (!notice) return null
  return (
    <p
      className={
        notice.kind === 'success'
          ? 'flex items-center gap-2 text-sm text-emerald-600 dark:text-emerald-400'
          : 'text-sm text-destructive'
      }
    >
      {notice.kind === 'success' ? <CircleCheck className="h-4 w-4" /> : null}
      {notice.text}
    </p>
  )
}
