'use client'

import * as React from 'react'
import {
  AtSign,
  CircleAlert,
  Eye,
  EyeOff,
  Loader2,
  Lock,
  Mail,
  RotateCw,
  UserPlus,
} from 'lucide-react'

import { createAccount } from '@/lib/admin'
import { AdminApiError } from '@/lib/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'

// Règles alignées sur le register / l'onboarding (parité de validation username).
const usernamePattern = /^[a-zA-Z0-9_]{3,24}$/
const reservedUsernames = new Set([
  'me',
  'admin',
  'root',
  'users',
  'by-username',
  'null',
  'undefined',
  'search',
  'suggestions',
])
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// Jeux de caractères sans ambigus (pas de O/0, l/1/I) pour un mot de passe
// temporaire lisible si l'admin doit le recopier.
const PWD_SETS = {
  upper: 'ABCDEFGHJKLMNPQRSTUVWXYZ',
  lower: 'abcdefghijkmnopqrstuvwxyz',
  digit: '23456789',
  symbol: '!@#$%&*?',
}

/** Entier cryptographiquement aléatoire dans [0, max[ (rejet de biais simple). */
function randomInt(max: number): number {
  return crypto.getRandomValues(new Uint32Array(1))[0] % max
}

/**
 * Génère un mot de passe temporaire sécurisé (16 chars), garantissant au moins
 * une majuscule, une minuscule, un chiffre et un symbole, puis mélangé.
 */
function generatePassword(length = 16): string {
  const all = PWD_SETS.upper + PWD_SETS.lower + PWD_SETS.digit + PWD_SETS.symbol
  const pick = (s: string) => s[randomInt(s.length)]
  const chars = [pick(PWD_SETS.upper), pick(PWD_SETS.lower), pick(PWD_SETS.digit), pick(PWD_SETS.symbol)]
  while (chars.length < length) chars.push(pick(all))
  for (let i = chars.length - 1; i > 0; i--) {
    const j = randomInt(i + 1)
    ;[chars[i], chars[j]] = [chars[j], chars[i]]
  }
  return chars.join('')
}

type Availability = 'idle' | 'checking' | 'available' | 'taken'

/**
 * Dialogue de création de compte réservé aux administrateurs (page /admin).
 *
 * L'admin saisit username + e-mail + mot de passe temporaire. À la validation,
 * `createAccount` orchestre auth → user → profil (cf. lib/admin) ; le mot de
 * passe temporaire est envoyé par e-mail côté back. Si le username est déjà pris,
 * le back le suffixe et l'utilisateur devra en choisir un autre à sa connexion
 * (gate `username_pending`). Le nouveau compte doit aussi changer son mot de
 * passe temporaire (gate `must_change_password`).
 */
export function CreateAccountDialog() {
  const t = useT()
  const { toast } = useToast()

  const [open, setOpen] = React.useState(false)
  const [username, setUsername] = React.useState('')
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [availability, setAvailability] = React.useState<Availability>('idle')
  const [usernameError, setUsernameError] = React.useState<string>()
  const [emailError, setEmailError] = React.useState<string>()
  const [passwordError, setPasswordError] = React.useState<string>()
  const [showPassword, setShowPassword] = React.useState(false)
  const [submitting, setSubmitting] = React.useState(false)

  // Réinitialise le formulaire à chaque ouverture.
  React.useEffect(() => {
    if (open) {
      setUsername('')
      setEmail('')
      setPassword('')
      setAvailability('idle')
      setUsernameError(undefined)
      setEmailError(undefined)
      setPasswordError(undefined)
      setShowPassword(false)
    }
  }, [open])

  // Génère un mot de passe temporaire sécurisé et le révèle (l'admin doit
  // pouvoir le noter — il sera communiqué à l'utilisateur par e-mail).
  function handleGeneratePassword() {
    setPassword(generatePassword())
    setPasswordError(undefined)
    setShowPassword(true)
  }

  // Vérification de disponibilité du username (débounce), même contrat que le register.
  React.useEffect(() => {
    const candidate = username.trim()
    if (
      !candidate ||
      !usernamePattern.test(candidate) ||
      reservedUsernames.has(candidate.toLowerCase())
    ) {
      setAvailability('idle')
      return
    }
    setAvailability('checking')
    const handle = setTimeout(async () => {
      try {
        const res = await fetch(
          `/api/users/check-username?username=${encodeURIComponent(candidate)}`,
        )
        const data = (await res.json().catch(() => null)) as { available?: boolean } | null
        if (!res.ok) {
          setAvailability('idle')
          return
        }
        setAvailability(data?.available ? 'available' : 'taken')
      } catch {
        setAvailability('idle')
      }
    }, 400)
    return () => clearTimeout(handle)
  }, [username])

  function validate(): boolean {
    let ok = true
    const u = username.trim()
    if (!u) {
      setUsernameError(t('auth.register.err.username_required'))
      ok = false
    } else if (!usernamePattern.test(u)) {
      setUsernameError(t('auth.register.err.username_format'))
      ok = false
    } else if (reservedUsernames.has(u.toLowerCase())) {
      setUsernameError(t('auth.register.err.username_reserved'))
      ok = false
    }
    if (!emailPattern.test(email.trim())) {
      setEmailError(t('admin.create.err.email_invalid'))
      ok = false
    }
    if (password.length < 8) {
      setPasswordError(t('admin.create.password_hint'))
      ok = false
    }
    return ok
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setUsernameError(undefined)
    setEmailError(undefined)
    setPasswordError(undefined)

    if (!validate()) return
    // Le username peut tout de même être pris (course) : le back suffixera alors.
    setSubmitting(true)
    try {
      const created = await createAccount({
        username: username.trim(),
        email: email.trim(),
        password,
      })
      toast({
        title: created.usernamePending
          ? t('admin.create.success_suffixed', {
              username: created.username,
              email: created.email,
            })
          : t('admin.create.success', { email: created.email }),
      })
      setOpen(false)
    } catch (err) {
      if (err instanceof AdminApiError && err.status === 409) {
        setEmailError(t('admin.create.err.email_taken'))
      } else {
        toast({ title: t('admin.create.err.generic'), variant: 'destructive' })
      }
    } finally {
      setSubmitting(false)
    }
  }

  const canSubmit = !submitting && availability !== 'checking'

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          size="sm"
          className="rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
        >
          <UserPlus className="mr-1.5 h-4 w-4" aria-hidden />
          {t('admin.create_account')}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('admin.create.title')}</DialogTitle>
          <DialogDescription>{t('admin.create.subtitle')}</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {/* Username */}
          <div className="flex flex-col gap-1">
            <label htmlFor="create-username" className="text-xs font-medium text-foreground/80">
              {t('admin.create.username_label')}
            </label>
            <div className="relative">
              <AtSign className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="create-username"
                value={username}
                onChange={(e) => {
                  setUsername(e.target.value)
                  setUsernameError(undefined)
                }}
                placeholder={t('onboarding.username_placeholder')}
                maxLength={24}
                autoComplete="off"
                className="pl-9"
              />
            </div>
            {usernameError ? (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {usernameError}
              </p>
            ) : availability === 'checking' ? (
              <p className="flex items-center gap-1 text-xs text-muted-foreground">
                <Loader2 className="h-3 w-3 animate-spin" />
                {t('onboarding.username_checking')}
              </p>
            ) : availability === 'taken' ? (
              <p className="text-xs text-muted-foreground">{t('onboarding.username_hint')}</p>
            ) : availability === 'available' ? (
              <p className="text-xs font-medium text-emerald-600 dark:text-emerald-400">
                {t('onboarding.username_available')}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">{t('onboarding.username_hint')}</p>
            )}
          </div>

          {/* E-mail */}
          <div className="flex flex-col gap-1">
            <label htmlFor="create-email" className="text-xs font-medium text-foreground/80">
              {t('admin.create.email_label')}
            </label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="create-email"
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value)
                  setEmailError(undefined)
                }}
                placeholder={t('admin.create.email_placeholder')}
                autoComplete="off"
                className="pl-9"
              />
            </div>
            {emailError && (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {emailError}
              </p>
            )}
          </div>

          {/* Mot de passe temporaire */}
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between">
              <label htmlFor="create-password" className="text-xs font-medium text-foreground/80">
                {t('admin.create.password_label')}
              </label>
              <button
                type="button"
                onClick={handleGeneratePassword}
                className="flex items-center gap-1 text-xs font-medium text-[#5B6CFF] transition hover:text-[#8D3DFF] dark:text-[#9aa6ff]"
              >
                <RotateCw className="h-3 w-3" aria-hidden />
                {t('admin.create.generate_password')}
              </button>
            </div>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="create-password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value)
                  setPasswordError(undefined)
                }}
                autoComplete="off"
                className="pl-9 pr-9"
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={
                  showPassword
                    ? t('admin.create.hide_password')
                    : t('admin.create.show_password')
                }
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground transition hover:text-foreground"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {passwordError ? (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <CircleAlert className="h-3 w-3" />
                {passwordError}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">{t('admin.create.password_hint')}</p>
            )}
          </div>

          <Button
            type="submit"
            disabled={!canSubmit}
            className="mt-1 w-full rounded-full bg-gradient-to-r from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)] font-bold text-white"
          >
            {submitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t('admin.create.submitting')}
              </span>
            ) : (
              t('admin.create.submit')
            )}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
