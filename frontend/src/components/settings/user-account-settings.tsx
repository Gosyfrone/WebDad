'use client'

import * as React from 'react'
import { CircleCheck, Loader2, Mail, ShieldCheck } from 'lucide-react'

import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getAccessToken, setAccessToken } from '@/lib/auth-client'
import { useSession } from '@/lib/session'

type Notice = { kind: 'error' | 'success'; text: string }

export function UserAccountSettings() {
  const t = useT()
  const session = useSession()
  const [currentPassword, setCurrentPassword] = React.useState('')
  const [newPassword, setNewPassword] = React.useState('')
  const [confirmPassword, setConfirmPassword] = React.useState('')
  const [passwordNotice, setPasswordNotice] = React.useState<Notice>()
  const [passwordBusy, setPasswordBusy] = React.useState(false)
  const [email, setEmail] = React.useState('')
  const [emailNotice, setEmailNotice] = React.useState<Notice>()
  const [emailBusy, setEmailBusy] = React.useState(false)

  const changePassword = async (event: React.FormEvent) => {
    event.preventDefault()
    setPasswordNotice(undefined)
    if (newPassword.length < 8) {
      setPasswordNotice({ kind: 'error', text: t('settings.password.error_length') })
      return
    }
    if (newPassword !== confirmPassword) {
      setPasswordNotice({ kind: 'error', text: t('settings.password.error_match') })
      return
    }
    if (newPassword === currentPassword) {
      setPasswordNotice({ kind: 'error', text: t('settings.password.error_same') })
      return
    }

    setPasswordBusy(true)
    try {
      const response = await fetch('/api/auth/password/change', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getAccessToken() ?? ''}`,
        },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      })
      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        setPasswordNotice({
          kind: 'error',
          text: payload?.code === 'invalid_current_password'
            ? t('settings.password.error_current')
            : t('settings.password.error_generic'),
        })
        return
      }
      if (payload?.accessToken) setAccessToken(payload.accessToken)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setPasswordNotice({ kind: 'success', text: t('settings.password.success') })
    } catch {
      setPasswordNotice({ kind: 'error', text: t('settings.password.error_generic') })
    } finally {
      setPasswordBusy(false)
    }
  }

  const requestEmailChange = async (event: React.FormEvent) => {
    event.preventDefault()
    setEmailNotice(undefined)
    setEmailBusy(true)
    try {
      const response = await fetch('/api/auth/email/change/request', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getAccessToken() ?? ''}`,
        },
        body: JSON.stringify({ email: email.trim() }),
      })
      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        const key = payload?.code === 'same_email'
          ? 'settings.email.error_same'
          : payload?.code === 'email_taken'
            ? 'settings.email.error_taken'
            : payload?.code === 'email_delivery_failed'
              ? 'settings.email.error_delivery'
              : 'settings.email.error_generic'
        setEmailNotice({ kind: 'error', text: t(key) })
        return
      }
      setEmail('')
      setEmailNotice({ kind: 'success', text: t('settings.email.success') })
    } catch {
      setEmailNotice({ kind: 'error', text: t('settings.email.error_generic') })
    } finally {
      setEmailBusy(false)
    }
  }

  return (
    <div className="divide-y">
      <div className="grid gap-4 px-4 py-5 md:grid-cols-[minmax(0,1fr)_minmax(280px,380px)]">
        <div className="flex items-start gap-3">
          <ShieldCheck className="mt-1 h-5 w-5 shrink-0 text-primary" aria-hidden />
          <div>
            <h3 className="font-semibold">{t('settings.password.title')}</h3>
            <p className="mt-1 text-sm text-muted-foreground">{t('settings.password.desc')}</p>
          </div>
        </div>
        <form className="panel space-y-3 rounded-xl border p-4 shadow-sm" onSubmit={changePassword}>
          <Field id="current-password" label={t('settings.password.current')} value={currentPassword} onChange={setCurrentPassword} autoComplete="current-password" />
          <Field id="new-password" label={t('settings.password.new')} value={newPassword} onChange={setNewPassword} autoComplete="new-password" />
          <Field id="confirm-password" label={t('settings.password.confirm')} value={confirmPassword} onChange={setConfirmPassword} autoComplete="new-password" />
          <NoticeMessage notice={passwordNotice} />
          <Button className="w-full" disabled={passwordBusy}>
            {passwordBusy ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            {t('settings.password.submit')}
          </Button>
        </form>
      </div>

      <div className="grid gap-4 px-4 py-5 md:grid-cols-[minmax(0,1fr)_minmax(280px,380px)]">
        <div className="flex items-start gap-3">
          <Mail className="mt-1 h-5 w-5 shrink-0 text-primary" aria-hidden />
          <div>
            <h3 className="font-semibold">{t('settings.email.title')}</h3>
            <p className="mt-1 text-sm text-muted-foreground">{t('settings.email.desc')}</p>
            {session?.email ? <p className="mt-2 text-sm font-medium">{session.email}</p> : null}
          </div>
        </div>
        <form className="panel space-y-3 rounded-xl border p-4 shadow-sm" onSubmit={requestEmailChange}>
          <label className="text-sm font-medium" htmlFor="new-email">{t('settings.email.new')}</label>
          <Input id="new-email" type="email" inputMode="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
          <NoticeMessage notice={emailNotice} />
          <Button className="w-full" disabled={emailBusy || !email.trim()}>
            {emailBusy ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            {t('settings.email.submit')}
          </Button>
        </form>
      </div>
    </div>
  )
}

function Field({ id, label, value, onChange, autoComplete }: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  autoComplete: string
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm font-medium" htmlFor={id}>{label}</label>
      <Input id={id} type="password" required value={value} autoComplete={autoComplete} onChange={(event) => onChange(event.target.value)} />
    </div>
  )
}

function NoticeMessage({ notice }: { notice?: Notice }) {
  if (!notice) return null
  return (
    <p className={notice.kind === 'success'
      ? 'flex items-center gap-2 text-sm text-emerald-600 dark:text-emerald-400'
      : 'text-sm text-destructive'}>
      {notice.kind === 'success' ? <CircleCheck className="h-4 w-4" /> : null}
      {notice.text}
    </p>
  )
}
