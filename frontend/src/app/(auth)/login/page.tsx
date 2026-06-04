'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import {
  Bell,
  CircleAlert,
  Eye,
  EyeOff,
  Heart,
  Lock,
  Mail,
  MessageCircle,
  Search,
  Sparkles,
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
import { setAccessToken } from '@/lib/auth-client'
import { ROUTES } from '@/lib/routes'

type FormErrors = Partial<{
  email: string
  password: string
  form: string
}>

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function getMessage(error: unknown, fallback: string): string {
  if (typeof error === 'string' && error.trim()) return error
  if (error instanceof Error && error.message.trim()) return error.message
  return fallback
}

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [showPassword, setShowPassword] = React.useState(false)
  const [errors, setErrors] = React.useState<FormErrors>({})
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  const validate = React.useCallback((): FormErrors => {
    const nextErrors: FormErrors = {}
    const trimmedEmail = email.trim()

    if (!trimmedEmail) {
      nextErrors.email = 'L’adresse e-mail est requise.'
    } else if (!emailPattern.test(trimmedEmail)) {
      nextErrors.email = 'Saisis une adresse e-mail valide.'
    }

    if (!password) {
      nextErrors.password = 'Le mot de passe est requis.'
    } else if (password.length < 8) {
      nextErrors.password = 'Le mot de passe doit contenir au moins 8 caractères.'
    }

    return nextErrors
  }, [email, password])

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const nextErrors = validate()
    setErrors(nextErrors)

    if (Object.keys(nextErrors).length > 0) return

    setIsSubmitting(true)

    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: email.trim(),
          password,
        }),
      })

      const payload = await response.json().catch(() => null)

      if (!response.ok) {
        setErrors({
          form:
            payload?.error ??
            payload?.message ??
            'La connexion a échoué. Vérifie tes identifiants.',
        })
        return
      }

      // L'access token court (15 min) vit en localStorage ; le refresh token
      // a été posé en cookie httpOnly par le BFF (/api/auth/login).
      if (payload?.accessToken) {
        setAccessToken(payload.accessToken)
      }

      router.replace(ROUTES.feed)
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
    <main className="bg-page relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-5 sm:px-6 lg:px-10">
      <div className="bg-page-glow-1 pointer-events-none absolute inset-0" />
      <div className="bg-page-glow-2 pointer-events-none absolute inset-0" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-white/70 dark:bg-white/10" />

      <section className="relative grid w-full max-w-6xl items-center gap-6 lg:grid-cols-[1.08fr_0.92fr]">
        <div className="hidden min-h-[620px] flex-col justify-between lg:flex">
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
              className="relative h-12 w-auto object-contain drop-shadow-sm"
              priority
            />
          </Link>

          <div className="relative mt-8 h-[500px]">
            <div className="glass absolute left-8 top-0 w-[410px] overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(91,108,255,0.28)] backdrop-blur-2xl">
              <div className="flex items-center justify-between border-b border-white/60 px-5 py-4 dark:border-white/10">
                <div>
                  <p className="text-xs font-semibold uppercase text-[#5B6CFF]">
                    Fil en direct
                  </p>
                  <h1 className="text-2xl font-semibold text-foreground">
                    Retrouve ton monde.
                  </h1>
                </div>

                <button
                  type="button"
                  aria-label="Rechercher"
                  className="grid h-10 w-10 place-items-center rounded-full bg-white/90 text-foreground/80 shadow-sm transition hover:scale-105 hover:text-[#5B6CFF] dark:bg-white/10"
                >
                  <Search className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-1 px-4 py-4">
                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex gap-3">
                    <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-gradient-to-br from-[#8D3DFF] to-[#47D9FF] text-sm font-bold text-white">
                      ML
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1 text-sm">
                        <span className="font-bold text-foreground">Mila</span>
                        <span className="truncate text-muted-foreground">@mila</span>
                        <span className="text-muted-foreground">·</span>
                        <span className="text-muted-foreground">2 min</span>
                      </div>
                      <p className="mt-1 text-sm leading-relaxed text-foreground/80">
                        Nouvelle playlist, nouveaux débats, même énergie Breezy.
                      </p>
                      <div className="mt-3 h-28 rounded-[18px] bg-gradient-to-br from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] p-px">
                        <div className="h-full rounded-[17px] bg-white/20 p-3">
                          <div className="h-full rounded-[14px] bg-white/25" />
                        </div>
                      </div>
                      <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                          <MessageCircle className="h-4 w-4" />
                          124
                        </span>
                        <span className="inline-flex items-center gap-1 text-red-500">
                          <Heart className="h-4 w-4 fill-red-500" />
                          2.8K
                        </span>
                        <span>18.4K vues</span>
                      </div>
                    </div>
                  </div>
                </article>

                <article className="glass rounded-[22px] border p-4 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-slate-950 text-sm font-bold text-white dark:bg-white/15">
                      NO
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-bold text-foreground">
                        Noa a rejoint la conversation
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Découvre les sujets qui montent ce soir.
                      </p>
                    </div>
                    <UserPlus className="h-5 w-5 text-[#8D3DFF]" />
                  </div>
                </article>
              </div>
            </div>

            <div className="absolute right-8 top-20 w-64 rounded-[28px] border border-white/40 bg-slate-950/90 p-4 text-white shadow-[0_28px_70px_rgba(15,23,42,0.32)] backdrop-blur-xl dark:border-white/10">
              <div className="flex items-center justify-between">
                <span className="text-sm font-semibold">Tendances</span>
                <Sparkles className="h-4 w-4 text-[#47D9FF]" />
              </div>

              <div className="mt-4 space-y-3">
                {['#DesignSprint', '#CampusLife', '#DevDistribue'].map(
                  (trend, index) => (
                    <div key={trend} className="rounded-2xl bg-white/10 px-3 py-2">
                      <p className="text-xs text-white/50">#{index + 1} sur Breezy</p>
                      <p className="text-sm font-semibold">{trend}</p>
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
                  17 nouvelles interactions
                </p>
                <p className="text-xs text-muted-foreground">
                  Ton fil t’attend, frais et vivant.
                </p>
              </div>
            </div>
          </div>
        </div>

        <Card className="glass-strong relative w-full overflow-hidden rounded-[30px] border shadow-[0_30px_90px_rgba(0,0,0,0.22)] backdrop-blur-2xl sm:max-w-md sm:justify-self-center lg:max-w-none">
          <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />
          <div className="pointer-events-none absolute inset-x-8 top-1 h-24 bg-gradient-to-b from-white/70 to-transparent dark:hidden" />

          <CardHeader className="relative space-y-4 px-5 pb-2 pt-5 sm:px-8 sm:pt-7">
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
              Connexion au réseau
            </div>

            <div className="space-y-2">
              <CardTitle className="brand-text max-w-md text-[31px] font-semibold leading-tight sm:text-[38px]">
                Reprends ton fil là où tu l’as laissé.
              </CardTitle>

              <CardDescription className="max-w-sm text-sm leading-relaxed text-muted-foreground">
                Connecte-toi à Breezy, retrouve tes messages, tes posts et les conversations qui bougent.
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="relative px-5 pb-5 sm:px-8 sm:pb-7">
            <form className="space-y-3.5" onSubmit={handleSubmit} noValidate>
              <div className="space-y-1.5">
                <label htmlFor="email" className="text-sm font-medium text-foreground/80">
                  Adresse e-mail
                </label>

                <div className="group relative">
                  <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    inputMode="email"
                    placeholder="toi@exemple.com"
                    className="h-12 rounded-2xl border-white/70 bg-white/90 pl-11 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
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
                  <p id="email-error" className="text-xs text-red-600">
                    {errors.email}
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <label htmlFor="password" className="text-sm font-medium text-foreground/80">
                  Mot de passe
                </label>

                <div className="group relative">
                  <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground transition group-focus-within:text-[#5B6CFF]" />

                  <Input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    placeholder="••••••••"
                    className="h-12 rounded-2xl border-white/70 bg-white/90 pl-11 pr-12 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-muted-foreground hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5"
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value)
                      if (errors.password) {
                        setErrors((current) => ({
                          ...current,
                          password: undefined,
                        }))
                      }
                    }}
                    aria-invalid={Boolean(errors.password)}
                    aria-describedby={errors.password ? 'password-error' : undefined}
                  />

                  <button
                    type="button"
                    aria-label={
                      showPassword
                        ? 'Masquer le mot de passe'
                        : 'Afficher le mot de passe'
                    }
                    aria-pressed={showPassword}
                    className="absolute right-3 top-1/2 grid h-8 w-8 -translate-y-1/2 place-items-center rounded-full text-muted-foreground transition hover:bg-[#5B6CFF]/10 hover:text-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                    onClick={() => setShowPassword((current) => !current)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                </div>

                {errors.password ? (
                  <p id="password-error" className="text-xs text-red-600">
                    {errors.password}
                  </p>
                ) : null}
              </div>

              <div className="flex items-center justify-between text-sm">
                <Link
                  href={ROUTES.home}
                  className="font-medium text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  Mot de passe oublié ?
                </Link>
              </div>

              {errors.form ? (
                <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2.5 text-sm text-red-700 shadow-sm dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <p>{errors.form}</p>
                </div>
              ) : null}

              <Button
                type="submit"
                className="h-12 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] hover:shadow-[0_24px_56px_rgba(91,108,255,0.42)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
                disabled={isSubmitting}
              >
                {isSubmitting ? 'Connexion en cours…' : 'Se connecter'}
              </Button>

              <div className="space-y-3 pt-1">
                <div className="flex items-center gap-3">
                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />

                  <span className="text-xs font-medium text-muted-foreground">
                    Ou se connecter avec
                  </span>

                  <div className="h-px flex-1 bg-gradient-to-r from-transparent via-border to-transparent" />
                </div>

                <div className="grid grid-cols-2 gap-2.5">
                  <button
                    type="button"
                    className="flex h-11 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
                  >
                    <Image
                      src="/google-logo.jpg"
                      alt="Google"
                      width={18}
                      height={18}
                    />
                    <span className="text-sm font-medium text-foreground/80">
                      Google
                    </span>
                  </button>

                  <button
                    type="button"
                    className="flex h-11 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md dark:border-white/15 dark:bg-white/10 dark:hover:bg-white/20"
                  >
                    <Image
                      src="/microsoft-logo.png"
                      alt="Microsoft"
                      width={18}
                      height={18}
                    />
                    <span className="text-sm font-medium text-foreground/80">
                      Microsoft
                    </span>
                  </button>
                </div>
              </div>

              <p className="text-center text-sm text-muted-foreground">
                Pas encore de compte ?{' '}
                <Link
                  href={ROUTES.register}
                  className="font-semibold text-[#5B6CFF] underline-offset-4 transition hover:text-[#8D3DFF] hover:underline"
                >
                  Créer un compte
                </Link>
              </p>
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  )
}
