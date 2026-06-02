'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { CircleAlert, Lock, Mail, User } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
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
    <main className="relative flex h-dvh w-full items-center justify-center overflow-hidden bg-slate-50 px-4 py-4 sm:px-6 sm:py-6">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(224,83,255,0.14),_transparent_34%),radial-gradient(circle_at_bottom_right,_rgba(56,189,248,0.12),_transparent_28%)]" />
      <div className="relative w-full max-w-[700px] overflow-hidden rounded-[2rem] border border-white/60 bg-white/80 px-4 py-4 shadow-[0_24px_80px_rgba(15,23,42,0.12)] backdrop-blur-xl sm:px-5 sm:py-5 lg:px-6 lg:py-6">
        <Link
          href={ROUTES.home}
          className="group absolute left-4 top-4 z-10 inline-flex items-center rounded-full transition-transform duration-300 hover:translate-y-[-1px] sm:left-5 sm:top-5 lg:left-6 lg:top-6"
        >
          <Image
            src="/logo_breezy.png"
            alt="Breezy"
            width={1106}
            height={336}
            className="h-7 w-auto object-contain transition-transform duration-300 group-hover:scale-[1.03] group-hover:drop-shadow-[0_0_28px_rgba(224,83,255,0.30)] sm:h-8"
            priority
          />
        </Link>

        <section className="flex w-full min-h-0 items-start pt-10 sm:pt-11 lg:pt-11">
          <div className="flex w-full flex-col gap-1.5 sm:gap-2">

            <h1 className="text-[1.7rem] font-semibold tracking-tight text-slate-950 sm:text-3xl xl:text-[2.4rem] xl:leading-[1.05]">
              Rejoins Breezy et commence à publier.
            </h1>

            <form className="space-y-1.5 sm:space-y-2" onSubmit={handleSubmit} noValidate>
              <div className="space-y-1 pt-0.5">
                <h2 className="text-lg font-semibold tracking-tight text-slate-950 sm:text-xl xl:text-[1.45rem]">
                  Inscription
                </h2>
                <p className="max-w-md text-[0.92rem] leading-5 text-slate-600 sm:text-sm sm:leading-6">
                  Crée ton compte avec un nom d’utilisateur, une adresse e-mail et un mot de passe sécurisé.
                </p>
              </div>

              <div className="space-y-1.5 sm:space-y-2">
                <div className="space-y-1.5">
                  <label
                    htmlFor="username"
                    className="text-sm font-medium text-slate-700"
                  >
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
                      className="h-9 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-10"
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
                  <p id="username-error" className="min-h-[1rem] text-sm leading-4 text-red-600">
                    {errors.username ?? ''}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label
                    htmlFor="email"
                    className="text-sm font-medium text-slate-700"
                  >
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
                      className="h-9 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-10"
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
                  <p id="email-error" className="min-h-[1rem] text-sm leading-4 text-red-600">
                    {errors.email ?? ''}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label
                    htmlFor="password"
                    className="text-sm font-medium text-slate-700"
                  >
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
                      className="h-9 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-10"
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
                    className="min-h-[1rem] text-[11px] leading-4 text-slate-500 sm:text-xs sm:leading-5"
                  >
                    {errors.password ?? '8 caractères minimum, une majuscule, une minuscule, un chiffre et un caractère spécial.'}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <label
                    htmlFor="passwordConfirmation"
                    className="text-sm font-medium text-slate-700"
                  >
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
                      className="h-9 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-10"
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
                        errors.passwordConfirmation ? 'password-confirmation-error' : undefined
                      }
                    />
                  </div>
                  <p
                    id="password-confirmation-error"
                    className="min-h-[1rem] text-sm leading-4 text-red-600"
                  >
                    {errors.passwordConfirmation ?? ''}
                  </p>
                </div>

                {errors.form ? (
                  <div className="flex min-h-[2.5rem] items-start gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
                    <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                    <p>{errors.form}</p>
                  </div>
                ) : null}

                <Button
                  type="submit"
                  className="mt-0.5 h-9 w-full rounded-xl bg-[#e053ff] text-white shadow-[0_18px_36px_rgba(224,83,255,0.28)] transition-transform hover:translate-y-[-1px] hover:bg-[#cf42f0] sm:h-10"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Création du compte…' : 'Créer mon compte'}
                </Button>

                <p className="pb-0 text-center text-sm text-slate-600">
                  Tu as déjà un compte ?{' '}
                  <Link
                    href={ROUTES.login}
                    className="font-medium text-[#e053ff] underline-offset-4 hover:underline"
                  >
                    Se connecter
                  </Link>
                </p>
              </div>
            </form>
          </div>
        </section>
      </div>
    </main>
  )
}
