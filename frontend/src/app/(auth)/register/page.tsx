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

function getMessage(error: unknown, fallback: string): string {
  if (typeof error === 'string' && error.trim()) {
    return error
  }

  if (error instanceof Error && error.message.trim()) {
    return error.message
  }

  return fallback
}

function extractToken(payload: RegisterResponse | null): string | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

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

  if (lowerMessage.includes('email')) {
    return { email: message }
  }

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
    } else if (trimmedUsername.length < 3) {
      nextErrors.username = 'Le nom d’utilisateur doit contenir au moins 3 caractères.'
    } else if (trimmedUsername.length > 30) {
      nextErrors.username = 'Le nom d’utilisateur ne doit pas dépasser 30 caractères.'
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
        'Le mot de passe doit contenir 8 caractères minimum, une majuscule, une minuscule, un chiffre et un caractère spécial.'
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

    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setIsSubmitting(true)

    try {
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
          (payload && typeof payload === 'object' && (payload.error ?? payload.message)) ||
          (typeof payload === 'string' && payload.trim()) ||
          'L’inscription a échoué. Vérifie les informations saisies.'

        setErrors(mapServerError(message))
        return
      }

      const token = extractToken(payload as RegisterResponse | null)

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
    <main className="relative flex min-h-dvh w-full items-center justify-center overflow-hidden bg-slate-50 px-4 py-6 sm:px-6 sm:py-8 lg:px-8">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(224,83,255,0.14),_transparent_34%),radial-gradient(circle_at_bottom_right,_rgba(56,189,248,0.12),_transparent_28%)]" />

      <Card className="relative w-full max-w-xl border-white/70 bg-white/90 shadow-[0_24px_80px_rgba(15,23,42,0.12)] backdrop-blur-xl">
        <CardHeader className="space-y-5 px-5 pb-4 pt-5 sm:px-8 sm:pt-8">
          <Link
            href={ROUTES.home}
            className="group inline-flex w-fit items-center transition-transform duration-300 hover:scale-[1.03]"
          >
            <Image
              src="/logo_breezy.png"
              alt="Breezy"
              width={1106}
              height={336}
              className="h-8 w-auto object-contain sm:h-9 lg:h-10 group-hover:drop-shadow-[0_0_18px_rgba(224,83,255,0.28)]"
              priority
            />
          </Link>

          <div className="space-y-1 sm:space-y-2">
            <CardTitle className="text-3xl tracking-tight text-slate-950 sm:text-4xl">
              Rejoins Breezy et commence à publier.
            </CardTitle>
            <CardDescription className="max-w-md text-sm text-slate-600 sm:text-base">
              Crée ton compte avec un nom d’utilisateur, une adresse e-mail et un mot de passe sécurisé.
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="px-5 pb-5 sm:px-8 sm:pb-8">
          <form className="space-y-4 sm:space-y-5" onSubmit={handleSubmit} noValidate>
            <div className="space-y-2">
              <div className="space-y-2">
                <label htmlFor="username" className="text-sm font-medium text-slate-700">
                  Nom d’utilisateur
                </label>
                <div className="relative">
                  <User className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="username"
                    name="username"
                    type="text"
                    autoComplete="username"
                    placeholder="breezy_user"
                    className="h-11 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff]"
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
                <p id="username-error" className="text-sm text-red-600">
                  {errors.username ?? ''}
                </p>
              </div>

              <div className="space-y-2">
                <label htmlFor="email" className="text-sm font-medium text-slate-700">
                  Adresse e-mail
                </label>
                <div className="relative">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    inputMode="email"
                    placeholder="toi@exemple.com"
                    className="h-11 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff]"
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
                <p id="email-error" className="text-sm text-red-600">
                  {errors.email ?? ''}
                </p>
              </div>

              <div className="space-y-2">
                <label htmlFor="password" className="text-sm font-medium text-slate-700">
                  Mot de passe
                </label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="password"
                    name="password"
                    type="password"
                    autoComplete="new-password"
                    placeholder="••••••••"
                    className="h-11 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff]"
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
                <p id={errors.password ? 'password-error' : 'password-help'} className="text-[11px] leading-5 text-slate-500 sm:text-xs sm:leading-5">
                  {errors.password ?? '8 caractères minimum, une majuscule, une minuscule, un chiffre et un caractère spécial.'}
                </p>
              </div>

              <div className="space-y-2">
                <label htmlFor="passwordConfirmation" className="text-sm font-medium text-slate-700">
                  Confirmation du mot de passe
                </label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  <Input
                    id="passwordConfirmation"
                    name="passwordConfirmation"
                    type="password"
                    autoComplete="new-password"
                    placeholder="••••••••"
                    className="h-11 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff]"
                    value={passwordConfirmation}
                    onChange={(event) => {
                      setPasswordConfirmation(event.target.value)
                      if (errors.passwordConfirmation) {
                        setErrors((current) => ({ ...current, passwordConfirmation: undefined }))
                      }
                    }}
                    aria-invalid={Boolean(errors.passwordConfirmation)}
                    aria-describedby={errors.passwordConfirmation ? 'password-confirmation-error' : undefined}
                  />
                </div>
                {errors.passwordConfirmation ? (<p id="password-confirmation-error" className="text-sm text-red-600">{errors.passwordConfirmation}</p>) : null}
              </div>

              {errors.form ? (
                <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <p>{errors.form}</p>
                </div>
              ) : null}

              <Button type="submit" className="h-11 w-full rounded-xl bg-[#e053ff] text-white shadow-[0_18px_36px_rgba(224,83,255,0.28)] transition-transform hover:translate-y-[-1px] hover:bg-[#cf42f0]" disabled={isSubmitting}>
                {isSubmitting ? 'Création du compte…' : 'Créer mon compte'}
              </Button>

              <p className="pb-1 text-center text-sm text-slate-600 sm:pb-0">
                Tu as déjà un compte ?{' '}
                <Link href={ROUTES.login} className="font-medium text-[#e053ff] underline-offset-4 hover:underline">Se connecter</Link>
              </p>
            </div>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
