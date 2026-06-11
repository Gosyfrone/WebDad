'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import {
  Bell,
  CalendarDays,
  CircleAlert,
  CircleHelp,
  Eye,
  EyeOff,
  Heart,
  Lock,
  Mail,
  MessageCircle,
  Search,
  Sparkles,
  User,
  UserPlus,
} from 'lucide-react'
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
import { LegalLinks } from '@/components/legal/legal-links'
import { ROUTES } from '@/lib/routes'
import { useT } from '@/components/language-provider'

type FormErrors = Partial<{
  username: string
  birthDate: string
  gender: string
  email: string
  password: string
  passwordConfirmation: string
  form: string
}>

type Gender = 'male' | 'female' | ''

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z\d]).{8,}$/
const usernamePattern = /^[a-zA-Z0-9_]{3,24}$/
const maxUsernameLength = 24
const maxEmailLength = 50
const maxPasswordLength = 250
const minBirthDate = '1900-01-01'
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

function toDateInputValue(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}

function isDateInputValue(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value)
}

function clampBirthDate(value: string, maxBirthDate: string): string {
  if (!isDateInputValue(value)) return value
  if (value > maxBirthDate) return maxBirthDate
  if (value < minBirthDate) return minBirthDate

  return value
}

export default function RegisterPage() {
  const t = useT()
  const router = useRouter()
  const [username, setUsername] = React.useState('')
  const [birthDate, setBirthDate] = React.useState('')
  const [gender, setGender] = React.useState<Gender>('')
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [passwordConfirmation, setPasswordConfirmation] = React.useState('')
  const [showPassword, setShowPassword] = React.useState(false)
  const [showPasswordConfirmation, setShowPasswordConfirmation] =
    React.useState(false)
  const [errors, setErrors] = React.useState<FormErrors>({})
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [oauthLoading, setOauthLoading] = React.useState<string | null>(null)
  const todayDate = React.useMemo(() => toDateInputValue(new Date()), [])
  const minimumAgeBirthDate = React.useMemo(() => {
    const date = new Date()
    date.setFullYear(date.getFullYear() - 13)
    return toDateInputValue(date)
  }, [])

  const validate = React.useCallback((): FormErrors => {
    const nextErrors: FormErrors = {}
    const trimmedUsername = username.trim()
    const trimmedEmail = email.trim()

    if (!trimmedUsername) {
      nextErrors.username = t('auth.register.err.username_required')
    } else if (!usernamePattern.test(trimmedUsername)) {
      nextErrors.username = t('auth.register.err.username_format')
    } else if (trimmedUsername.length > maxUsernameLength) {
      nextErrors.username = t('auth.register.err.username_max')
    } else if (reservedUsernames.has(trimmedUsername.toLowerCase())) {
      nextErrors.username = t('auth.register.err.username_reserved')
    }

    if (!birthDate) {
      nextErrors.birthDate = t('auth.register.err.birthdate_required')
    } else if (!isDateInputValue(birthDate) || birthDate < minBirthDate) {
      nextErrors.birthDate = t('auth.register.err.birthdate_invalid')
    } else if (birthDate > todayDate) {
      nextErrors.birthDate = t('auth.register.err.birthdate_future')
    } else if (birthDate > minimumAgeBirthDate) {
      nextErrors.birthDate = t('auth.register.err.age')
    }

    if (!gender) {
      nextErrors.gender = t('auth.register.err.gender_required')
    }

    if (!trimmedEmail) {
      nextErrors.email = t('auth.err.email_required')
    } else if (trimmedEmail.length > maxEmailLength) {
      nextErrors.email = t('auth.err.email_max')
    } else if (!emailPattern.test(trimmedEmail)) {
      nextErrors.email = t('auth.err.email_invalid')
    }

    if (!password) {
      nextErrors.password = t('auth.err.password_required')
    } else if (password.length > maxPasswordLength) {
      nextErrors.password = t('auth.register.err.password_max')
    } else if (!passwordPattern.test(password)) {
      nextErrors.password = t('auth.register.err.password_format')
    }

    if (!passwordConfirmation) {
      nextErrors.passwordConfirmation = t('auth.register.err.confirm_required')
    } else if (passwordConfirmation.length > maxPasswordLength) {
      nextErrors.passwordConfirmation = t('auth.register.err.confirm_max')
    } else if (passwordConfirmation !== password) {
      nextErrors.passwordConfirmation = t('auth.register.err.confirm_mismatch')
    }

    return nextErrors
  }, [birthDate, email, gender, minimumAgeBirthDate, password, passwordConfirmation, todayDate, username, t])

  const handleOAuth = async (provider: string) => {
    setOauthLoading(provider)
    try {
      const response = await fetch(`/api/auth/oauth/${provider}`)
      const payload = await response.json().catch(() => null)
      if (!response.ok || !payload?.url) {
        setErrors({ form: payload?.error ?? t('auth.oauth.error') })
        setOauthLoading(null)
        return
      }
      window.location.href = payload.url
    } catch {
      setErrors({ form: t('auth.err.network') })
      setOauthLoading(null)
    }
  }

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
        setErrors({ form: t('auth.register.err.username_check') })
        return
      }
      if (!availability?.available) {
        setErrors({ username: t('auth.register.err.username_taken') })
        return
      }

      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username.trim(),
          birthDate,
          gender,
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
          t('auth.register.err.failed')

        setErrors(mapServerError(message))
        return
      }

      // Blocage dur (vérification d'e-mail) : le register ne crée AUCUNE session
      // côté client (pas de token renvoyé). On redirige vers la page « consulte
      // ta boîte mail » ; l'utilisateur devra vérifier son adresse puis se
      // connecter.
      router.replace(
        `${ROUTES.checkEmail}?email=${encodeURIComponent(email.trim())}`
      )
    } catch (error) {
      setErrors({
        form: getMessage(error, t('auth.err.network')),
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="bg-page relative flex h-dvh w-full items-center justify-center overflow-hidden px-4 py-3 sm:px-6 lg:px-10">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-white/70 dark:bg-white/10" />

      <section className="relative grid w-full max-w-6xl items-center gap-6 lg:grid-cols-[1.08fr_0.92fr]">
        <div className="hidden min-h-[560px] flex-col justify-between lg:flex">
          <Link
            href={ROUTES.home}
            className="group relative inline-flex w-fit items-center transition duration-300 hover:scale-[1.03]"
          >
            <span className="absolute inset-0 bg-gradient-to-r from-[#8D3DFF]/30 to-[#47D9FF]/25 blur-2xl" />

            <Image
              src="/logo_breezy.png"
              alt="Breezy"
              width={1106}
              height={336}
              className="relative h-16 w-auto object-contain drop-shadow-sm"
              priority
            />
          </Link>

          <div className="relative mt-4 h-[450px]">
            <div className="glass absolute left-8 top-0 w-[410px] overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(91,108,255,0.28)] backdrop-blur-2xl">
              <div className="flex items-center justify-between border-b border-white/60 px-5 py-3 dark:border-white/10">
                <div>
                  <p className="text-xs font-semibold uppercase text-[#5B6CFF]">
                    {t('auth.register.demo.kicker')}
                  </p>
                  <h1 className="text-2xl font-semibold text-foreground">
                    {t('auth.register.demo.heading')}
                  </h1>
                </div>

                <button
                  type="button"
                  aria-label={t('auth.search_aria')}
                  className="grid h-10 w-10 place-items-center rounded-full bg-white/90 text-foreground/80 shadow-sm transition hover:scale-105 hover:text-[#5B6CFF] dark:bg-white/10"
                >
                  <Search className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-1 px-4 py-4">
                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex gap-3">
                    <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-gradient-to-br from-[#8D3DFF] to-[#47D9FF] text-sm font-bold text-white">
                      TO
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1 text-sm">
                        <span className="font-bold text-foreground">
                          {t('auth.register.demo.you')}
                        </span>
                        <span className="truncate text-muted-foreground">@breezy_user</span>
                        <span className="text-muted-foreground">·</span>
                        <span className="text-muted-foreground">
                          {t('auth.register.demo.now')}
                        </span>
                      </div>
                      <p className="mt-1 text-sm leading-relaxed text-foreground/80">
                        {t('auth.register.demo.post1')}
                      </p>
                      <div className="mt-3 grid grid-cols-3 gap-2">
                        <div className="h-16 rounded-[16px] bg-gradient-to-br from-[#8D3DFF] to-[#5B6CFF]" />
                        <div className="h-16 rounded-[16px] bg-gradient-to-br from-[#47D9FF] to-[#5B6CFF]" />
                        <div className="h-16 rounded-[16px] bg-gradient-to-br from-slate-950 to-[#8D3DFF]" />
                      </div>
                      <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                          <MessageCircle className="h-4 w-4" />
                          48
                        </span>
                        <span className="inline-flex items-center gap-1 text-red-500">
                          <Heart className="h-4 w-4 fill-red-500" />
                          1.2K
                        </span>
                        <span>{t('auth.register.demo.welcome')}</span>
                      </div>
                    </div>
                  </div>
                </article>

                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-slate-950 text-sm font-bold text-white dark:bg-white/15">
                      BR
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-bold text-foreground">
                        {t('auth.register.demo.opens')}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {t('auth.register.demo.opens_sub')}
                      </p>
                    </div>
                    <UserPlus className="h-5 w-5 text-[#8D3DFF]" />
                  </div>
                </article>
              </div>
            </div>

            <div className="absolute right-8 top-20 w-64 rounded-[28px] border border-white/40 bg-slate-950/90 p-4 text-white shadow-[0_28px_70px_rgba(15,23,42,0.32)] backdrop-blur-xl dark:border-white/10">
              <div className="flex items-center justify-between">
                <span className="text-sm font-semibold">
                  {t('auth.register.demo.to_join')}
                </span>
                <Sparkles className="h-4 w-4 text-[#47D9FF]" />
              </div>

              <div className="mt-4 space-y-3">
                {['auth.register.demo.comm1', 'auth.register.demo.comm2', 'auth.register.demo.comm3'].map(
                  (communityKey, index) => (
                    <div
                      key={communityKey}
                      className="rounded-2xl bg-white/10 px-3 py-2"
                    >
                      <p className="text-xs text-white/50">
                        {t('auth.register.demo.community', { n: index + 1 })}
                      </p>
                      <p className="text-sm font-semibold">{t(communityKey)}</p>
                    </div>
                  )
                )}
              </div>
            </div>

            <div className="glass absolute bottom-0 right-24 flex w-72 items-center gap-3 rounded-[24px] border px-4 py-3 shadow-[0_24px_70px_rgba(141,61,255,0.22)] backdrop-blur-xl">
              <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-[#47D9FF]/20 text-[#5B6CFF]">
                <Bell className="h-5 w-5" />
              </div>
              <div>
                <p className="text-sm font-bold text-foreground">
                  {t('auth.register.demo.alive')}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t('auth.register.demo.alive_sub')}
                </p>
              </div>
            </div>
          </div>
        </div>

        <Card className="glass-strong relative w-full overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.22)] backdrop-blur-2xl sm:max-w-md sm:justify-self-center lg:max-w-none">
          <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />
          <div className="pointer-events-none absolute inset-x-8 top-1 h-24 bg-gradient-to-b from-white/70 to-transparent dark:hidden" />

          <CardHeader className="relative space-y-1.5 px-5 pb-1 pt-3 sm:px-7 sm:pt-4">
            <Link
              href={ROUTES.home}
              className="group inline-flex w-fit items-center transition duration-300 hover:scale-[1.03] lg:hidden"
            >
              <Image
                src="/logo_breezy.png"
                alt="Breezy"
                width={1106}
                height={336}
                className="h-9 w-auto object-contain drop-shadow-sm"
                priority
              />
            </Link>

            <div className="inline-flex w-fit items-center gap-2 rounded-full border border-white/70 bg-white/80 px-3 py-1 text-xs font-semibold text-[#5B6CFF] shadow-sm dark:border-white/15 dark:bg-white/10">
              <span className="h-2 w-2 rounded-full bg-[#47D9FF]" />
              {t('auth.register.badge')}
            </div>

            <div className="space-y-1.5">
              <CardTitle className="brand-text max-w-md text-[24px] font-semibold leading-tight sm:text-[28px]">
                {t('auth.register.title')}
              </CardTitle>

              <CardDescription className="max-w-sm text-xs leading-relaxed text-muted-foreground">
                {t('auth.register.subtitle')}
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="relative px-5 pb-3 sm:px-7 sm:pb-4">
            <form className="space-y-1" onSubmit={handleSubmit} noValidate>
              <div className="grid gap-1.5 sm:grid-cols-2">
                <div className="space-y-0.5">
                  <div className="flex min-h-5 items-center gap-1.5">
                    <label htmlFor="username" className="text-xs font-medium text-foreground/80">
                      {t('auth.register.username_label')}
                    </label>

                    <span className="group/help relative inline-flex">
                      <button
                        type="button"
                        aria-describedby="username-tooltip"
                        className="grid h-4 w-4 place-items-center rounded-full text-muted-foreground transition hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#5B6CFF]/30"
                      >
                        <CircleHelp className="h-3.5 w-3.5" />
                      </button>
                      <span
                        id="username-tooltip"
                        role="tooltip"
                        className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-56 -translate-x-1/2 rounded-2xl border border-white/70 bg-slate-950 px-3 py-2 text-[11px] font-medium leading-4 text-white opacity-0 shadow-[0_16px_40px_rgba(15,23,42,0.25)] transition group-hover/help:opacity-100 group-focus-within/help:opacity-100 dark:border-white/10"
                      >
                        {t('auth.register.username_tooltip')}
                      </span>
                    </span>
                  </div>

                  <div className="group relative">
                    <User className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                    <Input
                      id="username"
                      name="username"
                      type="text"
                      autoComplete="username"
                      placeholder="breezy_user"
                      maxLength={maxUsernameLength}
                      className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
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
                  <div className="flex min-h-5 items-center gap-1.5">
                    <label htmlFor="birthDate" className="text-xs font-medium text-foreground/80">
                      {t('auth.register.birthdate_label')}
                    </label>

                    <span className="group/help relative inline-flex">
                      <button
                        type="button"
                        aria-describedby="birth-date-tooltip"
                        className="grid h-4 w-4 place-items-center rounded-full text-muted-foreground transition hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#5B6CFF]/30"
                      >
                        <CircleHelp className="h-3.5 w-3.5" />
                      </button>
                      <span
                        id="birth-date-tooltip"
                        role="tooltip"
                        className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-56 -translate-x-1/2 rounded-2xl border border-white/70 bg-slate-950 px-3 py-2 text-[11px] font-medium leading-4 text-white opacity-0 shadow-[0_16px_40px_rgba(15,23,42,0.25)] transition group-hover/help:opacity-100 group-focus-within/help:opacity-100 dark:border-white/10"
                      >
                        {t('auth.register.birthdate_tooltip')}
                      </span>
                    </span>
                  </div>

                  <div className="group relative">
                    <CalendarDays className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                    <Input
                      id="birthDate"
                      name="birthDate"
                      type="date"
                      autoComplete="bday"
                      min={minBirthDate}
                      max={todayDate}
                      className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all [color-scheme:light] hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5 dark:[color-scheme:dark]"
                      value={birthDate}
                      onChange={(event) => {
                        // Valeur BRUTE pendant la frappe : ne pas clamper ici.
                        // Taper l'année chiffre par chiffre produit des années
                        // intermédiaires minuscules (0001→0019→…→1995) ; les
                        // clamper à 1900 mid-saisie réécrirait le segment et
                        // bloquerait l'utilisateur à l'an 1900. Le cap est appliqué
                        // au blur (ci-dessous) + à la validation au submit.
                        setBirthDate(event.target.value)
                        if (errors.birthDate) {
                          setErrors((current) => ({ ...current, birthDate: undefined }))
                        }
                      }}
                      onBlur={(event) => {
                        // Saisie terminée : on cale la date dans [1900-01-01, aujourd'hui].
                        setBirthDate(clampBirthDate(event.currentTarget.value, todayDate))
                      }}
                      aria-invalid={Boolean(errors.birthDate)}
                      aria-describedby={
                        errors.birthDate ? 'birth-date-error' : undefined
                      }
                    />
                  </div>

                  {errors.birthDate ? (
                    <p id="birth-date-error" className="text-[11px] leading-4 text-red-600">
                      {errors.birthDate}
                    </p>
                  ) : null}
                </div>

                <fieldset className="space-y-0.5 sm:col-span-2">
                  <legend className="text-xs font-medium text-foreground/80">
                    {t('auth.register.gender_label')}
                  </legend>

                  <div className="grid grid-cols-2 gap-2">
                    <label className="group relative">
                      <input
                        type="radio"
                        name="gender"
                        value="male"
                        checked={gender === 'male'}
                        className="peer sr-only"
                        onChange={() => {
                          setGender('male')
                          if (errors.gender) {
                            setErrors((current) => ({
                              ...current,
                              gender: undefined,
                            }))
                          }
                        }}
                      />

                      <span className="flex h-8 cursor-pointer items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 text-xs font-semibold text-foreground/80 shadow-sm transition peer-checked:border-[#5B6CFF]/70 peer-checked:bg-[#5B6CFF]/10 peer-checked:text-[#5B6CFF] group-hover:bg-white dark:border-white/15 dark:bg-white/10 dark:peer-checked:bg-[#5B6CFF]/20 dark:group-hover:bg-white/20">
                        <User className="h-4 w-4" />
                        {t('auth.register.gender_male')}
                      </span>
                    </label>

                    <label className="group relative">
                      <input
                        type="radio"
                        name="gender"
                        value="female"
                        checked={gender === 'female'}
                        className="peer sr-only"
                        onChange={() => {
                          setGender('female')
                          if (errors.gender) {
                            setErrors((current) => ({
                              ...current,
                              gender: undefined,
                            }))
                          }
                        }}
                      />

                      <span className="flex h-8 cursor-pointer items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 text-xs font-semibold text-foreground/80 shadow-sm transition peer-checked:border-[#8D3DFF]/70 peer-checked:bg-[#8D3DFF]/10 peer-checked:text-[#8D3DFF] group-hover:bg-white dark:border-white/15 dark:bg-white/10 dark:peer-checked:bg-[#8D3DFF]/25 dark:group-hover:bg-white/20">
                        <User className="h-4 w-4" />
                        {t('auth.register.gender_female')}
                      </span>
                    </label>
                  </div>

                  {errors.gender ? (
                    <p id="gender-error" className="text-[11px] leading-4 text-red-600">
                      {errors.gender}
                    </p>
                  ) : null}
                </fieldset>
              </div>

              <div className="space-y-0.5">
                <label htmlFor="email" className="text-xs font-medium text-foreground/80">
                  {t('auth.email_label')}
                </label>

                <div className="group relative">
                  <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    inputMode="email"
                    placeholder={t('auth.email_placeholder')}
                    maxLength={maxEmailLength}
                    className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
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
                <label htmlFor="password" className="text-xs font-medium text-foreground/80">
                  {t('auth.password_label')}
                </label>

                <div className="group relative">
                  <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="new-password"
                    placeholder="••••••••"
                    maxLength={maxPasswordLength}
                    className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 pr-12 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
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

                  <button
                    type="button"
                    aria-label={
                      showPassword ? t('auth.hide_password') : t('auth.show_password')
                    }
                    aria-pressed={showPassword}
                    className="absolute right-3 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition hover:bg-[#5B6CFF]/10 hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                    onClick={() => setShowPassword((current) => !current)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                </div>

                <p
                  id={errors.password ? 'password-error' : 'password-help'}
                  className={`text-[10px] leading-3 ${
                    errors.password ? 'text-red-600' : 'text-muted-foreground'
                  }`}
                >
                  {errors.password ?? t('auth.register.password_help')}
                </p>
              </div>

              <div className="space-y-0.5">
                <label
                  htmlFor="passwordConfirmation"
                  className="text-xs font-medium text-foreground/80"
                >
                  {t('auth.register.password_confirm_label')}
                </label>

                <div className="group relative">
                  <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="passwordConfirmation"
                    name="passwordConfirmation"
                    type={showPasswordConfirmation ? 'text' : 'password'}
                    autoComplete="new-password"
                    placeholder="••••••••"
                    maxLength={maxPasswordLength}
                    className="h-9 rounded-2xl border-white/70 bg-white/90 pl-11 pr-12 text-sm shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
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

                  <button
                    type="button"
                    aria-label={
                      showPasswordConfirmation
                        ? t('auth.register.hide_password_confirm')
                        : t('auth.register.show_password_confirm')
                    }
                    aria-pressed={showPasswordConfirmation}
                    className="absolute right-3 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition hover:bg-[#5B6CFF]/10 hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                    onClick={() =>
                      setShowPasswordConfirmation((current) => !current)
                    }
                  >
                    {showPasswordConfirmation ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
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
                <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2 text-xs text-red-700 shadow-sm dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <p>{errors.form}</p>
                </div>
              ) : null}

              <Button
                type="submit"
                className="h-9 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-sm font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] hover:shadow-[0_24px_56px_rgba(91,108,255,0.42)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
                disabled={isSubmitting}
              >
                {isSubmitting ? t('auth.register.submitting') : t('auth.register.submit')}
              </Button>

              <div className="space-y-2 pt-0.5">
                <div className="flex items-center gap-3">
                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />

                  <span className="text-[11px] font-medium text-muted-foreground">
                    {t('auth.register.or')}
                  </span>

                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />
                </div>

                <div className="grid grid-cols-1 gap-2">
                  <button
                    type="button"
                    disabled={oauthLoading !== null || isSubmitting}
                    onClick={() => handleOAuth('google')}
                    className="flex h-9 items-center justify-center gap-2 rounded-2xl border border-gray-300 bg-white transition hover:scale-[1.01] hover:bg-white hover:shadow-md disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <Image
                      src="/google-logo.jpg"
                      alt="Google"
                      width={17}
                      height={17}
                    />
                    <span className="text-sm font-medium text-gray-800">
                      {oauthLoading === 'google' ? t('auth.oauth.loading') : 'Google'}
                    </span>
                  </button>
                </div>
              </div>

              <p className="text-center text-xs text-muted-foreground">
                {t('auth.register.have_account')}{' '}
                <Link
                  href={ROUTES.login}
                  className="font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  {t('auth.login.submit')}
                </Link>
              </p>

              <LegalLinks className="items-center text-center" />
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  )
}
