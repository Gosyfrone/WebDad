'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { CircleAlert, Lock, Mail, User } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'

type FormErrors = Partial<{
  username: string
  email: string
  password: string
  passwordConfirmation: string
  form: string
}>

type RegisterResponse = {
  token?: string
  accessToken?: string
  jwt?: string
  data?: {
    token?: string
    accessToken?: string
    jwt?: string
  }
  message?: string
  error?: string
}

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z\d]).{8,}$/
// Aligné sur le user-service (^[a-zA-Z0-9_]{3,50}$ + mots réservés).
const usernamePattern = /^[a-zA-Z0-9_]{3,50}$/
const reservedUsernames = new Set([
  'me',
  'admin',
  'root',
  'users',
  'by-username',
  'null',
  'undefined',
])

function getMessage(error: unknown, fallback: string): string {
  if (typeof error === 'string' && error.trim()) return error
  if (error instanceof Error && error.message.trim()) return error.message
  return fallback
}

function extractToken(payload: RegisterResponse | null): string | null {
  if (!payload || typeof payload !== 'object') return null

  return (
    payload.token ??
    payload.accessToken ??
    payload.jwt ??
    payload.data?.token ??
    payload.data?.accessToken ??
    payload.data?.jwt ??
    null
  )
}

function mapServerError(message: string): FormErrors {
  const lowerMessage = message.toLowerCase()

  if (lowerMessage.includes('email')) return { email: message }
  if (lowerMessage.includes('username') || lowerMessage.includes('nom d')) {
    return { username: message }
  }
  if (lowerMessage.includes('password') || lowerMessage.includes('mot de passe')) {
    return { password: message }
  }

  return { form: message }
}

export default function RegisterPage() {
  const router = useRouter()
  const [username, setUsername] = React.useState('')
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [passwordConfirmation, setPasswordConfirmation] = React.useState('')
  const [errors, setErrors] = React.useState<FormErrors>({})
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  const validate = React.useCallback((): FormErrors => {
    const nextErrors: FormErrors = {}
    const trimmedUsername = username.trim()
    const trimmedEmail = email.trim()

    if (!trimmedUsername) {
      nextErrors.username = 'Le nom d’utilisateur est requis.'
    } else if (!usernamePattern.test(trimmedUsername)) {
      nextErrors.username =
        '3 à 50 caractères : lettres, chiffres et tiret bas (_) uniquement.'
    } else if (reservedUsernames.has(trimmedUsername.toLowerCase())) {
      nextErrors.username = 'Ce nom d’utilisateur n’est pas autorisé.'
    }

    if (!trimmedEmail) {
      nextErrors.email = 'L’adresse e-mail est requise.'
    } else if (!emailPattern.test(trimmedEmail)) {
      nextErrors.email = 'Saisis une adresse e-mail valide.'
    }

    if (!password) {
      nextErrors.password = 'Le mot de passe est requis.'
    } else if (!passwordPattern.test(password)) {
      nextErrors.password =
        '8 caractères minimum, une majuscule, une minuscule, un chiffre et un caractère spécial.'
    }

    if (!passwordConfirmation) {
      nextErrors.passwordConfirmation = 'Confirme ton mot de passe.'
    } else if (passwordConfirmation !== password) {
      nextErrors.passwordConfirmation = 'Les mots de passe ne correspondent pas.'
    }

    return nextErrors
  }, [email, password, passwordConfirmation, username])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const nextErrors = validate()
    setErrors(nextErrors)

    if (Object.keys(nextErrors).length > 0) return

    setIsSubmitting(true)

    try {
      // Pré-vérification de la disponibilité du nom d'utilisateur (avant de créer
      // le compte), pour afficher l'erreur sur le champ plutôt qu'après coup.
      const availabilityResponse = await fetch(
        `/api/users/check-username?username=${encodeURIComponent(username.trim())}`
      )
      const availability = await availabilityResponse.json().catch(() => null)

      if (!availabilityResponse.ok) {
        setErrors({
          form: 'Impossible de vérifier le nom d’utilisateur. Réessaie dans un instant.',
        })
        return
      }
      if (!availability?.available) {
        setErrors({ username: 'Ce nom d’utilisateur est déjà pris.' })
        return
      }

      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username.trim(),
          email: email.trim(),
          password,
        }),
      })

      const payload = await response.json().catch(() => null)

      if (!response.ok) {
        const message =
          (payload &&
            typeof payload === 'object' &&
            (payload.error ?? payload.message)) ||
          (typeof payload === 'string' && payload.trim()) ||
          'L’inscription a échoué. Vérifie les informations saisies.'

        setErrors(mapServerError(message))
        return
      }

      const token = extractToken(payload as RegisterResponse | null)

      // L'access token court (15 min) vit en localStorage ; le refresh token
      // a été posé en cookie httpOnly par le BFF (/api/auth/register).
      if (token) {
        setAccessToken(token)
      }

      router.replace(token ? ROUTES.feed : ROUTES.login)
      router.refresh()
    } catch (error) {
      setErrors({
        form: getMessage(
          error,
          'Impossible de contacter l’API. Réessaie dans un instant.'
        ),
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main
      className="relative flex h-dvh w-full items-center justify-center overflow-hidden px-4 py-2 sm:px-6"
      style={{
        background:
          'linear-gradient(140deg, #f8f3ff 0%, #eadcff 28%, #d9c6ff 62%, #ebe8ff 100%)',
      }}
    >
      <div className="pointer-events-none absolute -left-28 top-[-80px] h-[560px] w-[560px] rounded-full bg-[#a855f7]/18 blur-[140px]" />
      <div className="pointer-events-none absolute left-[18%] top-[10%] h-[340px] w-[340px] rounded-full bg-[#c084fc]/20 blur-[110px]" />
      <div className="pointer-events-none absolute right-[-120px] top-[18%] h-[300px] w-[300px] rounded-full bg-[#47D9FF]/10 blur-[120px]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,255,255,0.28),transparent_62%)]" />

      <Card className="relative w-full max-w-md overflow-hidden rounded-[24px] border border-white/25 bg-white/82 shadow-[0_28px_80px_rgba(0,0,0,0.22)] backdrop-blur-2xl">
        <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />

        <CardHeader className="space-y-2 px-5 pb-1 pt-4 sm:px-6">
          <Link
            href={ROUTES.home}
            className="group relative inline-flex w-fit items-center transition duration-300 hover:scale-[1.03]"
          >
            <div className="absolute inset-0 rounded-full bg-gradient-to-r from-[#8D3DFF]/35 to-[#47D9FF]/30 blur-2xl" />

            <Image
              src="/logo_breezy.png"
              alt="Breezy"
              width={1106}
              height={336}
              className="relative h-7 w-auto object-contain drop-shadow-sm sm:h-8"
              priority
            />
          </Link>

          <div className="space-y-1">
            <CardTitle className="max-w-md bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-[23px] font-semibold leading-tight tracking-[-0.04em] text-transparent sm:text-[26px]">
              Rejoins Breezy et commence à publier.
            </CardTitle>

            <CardDescription className="max-w-sm text-xs leading-snug text-slate-500">
              Crée ton compte avec un nom d’utilisateur, une adresse e-mail et un mot de passe sécurisé.
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="px-5 pb-4 sm:px-6">
          <form className="space-y-2" onSubmit={handleSubmit} noValidate>
            <div className="space-y-0.5">
              <label htmlFor="username" className="text-xs font-medium text-slate-700">
                Nom d’utilisateur
              </label>

              <div className="group relative">
                <User className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition group-focus-within:text-[#5B6CFF]" />

                <Input
                  id="username"
                  name="username"
                  type="text"
                  autoComplete="username"
                  placeholder="breezy_user"
                  className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                  value={username}
                  onChange={(event) => {
                    setUsername(event.target.value)
                    if (errors.username) {
                      setErrors((current) => ({ ...current, username: undefined }))
                    }
                  }}
                  aria-invalid={Boolean(errors.username)}
                  aria-describedby={errors.username ? 'username-error' : undefined}
                />
              </div>

              {errors.username ? (
                <p id="username-error" className="text-[11px] leading-4 text-red-600">
                  {errors.username}
                </p>
              ) : null}
            </div>

            <div className="space-y-0.5">
              <label htmlFor="email" className="text-xs font-medium text-slate-700">
                Adresse e-mail
              </label>

              <div className="group relative">
                <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition group-focus-within:text-[#5B6CFF]" />

                <Input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  inputMode="email"
                  placeholder="toi@exemple.com"
                  className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value)
                    if (errors.email) {
                      setErrors((current) => ({ ...current, email: undefined }))
                    }
                  }}
                  aria-invalid={Boolean(errors.email)}
                  aria-describedby={errors.email ? 'email-error' : undefined}
                />
              </div>

              {errors.email ? (
                <p id="email-error" className="text-[11px] leading-4 text-red-600">
                  {errors.email}
                </p>
              ) : null}
            </div>

            <div className="space-y-0.5">
              <label htmlFor="password" className="text-xs font-medium text-slate-700">
                Mot de passe
              </label>

              <div className="group relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition group-focus-within:text-[#5B6CFF]" />

                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  placeholder="••••••••"
                  className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value)
                    if (errors.password) {
                      setErrors((current) => ({ ...current, password: undefined }))
                    }
                  }}
                  aria-invalid={Boolean(errors.password)}
                  aria-describedby={errors.password ? 'password-error' : 'password-help'}
                />
              </div>

              <p
                id={errors.password ? 'password-error' : 'password-help'}
                className={`text-[10px] leading-3 ${
                  errors.password ? 'text-red-600' : 'text-slate-500'
                }`}
              >
                {errors.password ??
                  '8 caractères min., majuscule, minuscule, chiffre et caractère spécial.'}
              </p>
            </div>

            <div className="space-y-0.5">
              <label
                htmlFor="passwordConfirmation"
                className="text-xs font-medium text-slate-700"
              >
                Confirmation du mot de passe
              </label>

              <div className="group relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition group-focus-within:text-[#5B6CFF]" />

                <Input
                  id="passwordConfirmation"
                  name="passwordConfirmation"
                  type="password"
                  autoComplete="new-password"
                  placeholder="••••••••"
                  className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                  value={passwordConfirmation}
                  onChange={(event) => {
                    setPasswordConfirmation(event.target.value)
                    if (errors.passwordConfirmation) {
                      setErrors((current) => ({
                        ...current,
                        passwordConfirmation: undefined,
                      }))
                    }
                  }}
                  aria-invalid={Boolean(errors.passwordConfirmation)}
                  aria-describedby={
                    errors.passwordConfirmation
                      ? 'password-confirmation-error'
                      : undefined
                  }
                />
              </div>

              {errors.passwordConfirmation ? (
                <p
                  id="password-confirmation-error"
                  className="text-[11px] leading-4 text-red-600"
                >
                  {errors.passwordConfirmation}
                </p>
              ) : null}
            </div>

            {errors.form ? (
              <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2 text-xs text-red-700 shadow-sm">
                <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                <p>{errors.form}</p>
              </div>
            ) : null}

            <Button
              type="submit"
              className="h-9 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-sm font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] hover:shadow-[0_24px_56px_rgba(91,108,255,0.42)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Création du compte…' : 'Créer mon compte'}
            </Button>

            <div className="space-y-2 pt-0.5">
              <div className="flex items-center gap-3">
                <div className="h-px flex-1 bg-gradient-to-r from-transparent via-slate-300 to-transparent" />

                <span className="text-[11px] font-medium text-slate-500">
                  Ou créer mon compte avec
                </span>

                <div className="h-px flex-1 bg-gradient-to-r from-transparent via-slate-300 to-transparent" />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  className="flex h-9 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md"
                >
                  <Image
                    src="/google-logo.jpg"
                    alt="Google"
                    width={17}
                    height={17}
                  />
                  <span className="text-sm font-medium text-slate-700">
                    Google
                  </span>
                </button>

                <button
                  type="button"
                  className="flex h-9 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md"
                >
                  <Image
                    src="/microsoft-logo.png"
                    alt="Microsoft"
                    width={17}
                    height={17}
                  />
                  <span className="text-sm font-medium text-slate-700">
                    Microsoft
                  </span>
                </button>
              </div>
            </div>

            <p className="text-center text-xs text-slate-500">
              Tu as déjà un compte ?{' '}
              <Link
                href={ROUTES.login}
                className="font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
              >
                Se connecter
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}