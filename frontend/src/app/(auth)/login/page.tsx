'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { CircleAlert, Lock, Mail } from 'lucide-react'
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
    <main
      className="relative flex min-h-dvh w-full items-center justify-center overflow-hidden px-4 py-3 sm:px-6"
      style={{
        background:
          'linear-gradient(140deg, #f8f3ff 0%, #eadcff 28%, #d9c6ff 62%, #ebe8ff 100%)',
      }}
    >
      <div className="pointer-events-none absolute -left-28 top-[-80px] h-[560px] w-[560px] rounded-full bg-[#a855f7]/18 blur-[140px]" />
      <div className="pointer-events-none absolute left-[18%] top-[10%] h-[340px] w-[340px] rounded-full bg-[#c084fc]/20 blur-[110px]" />
      <div className="pointer-events-none absolute right-[-120px] top-[18%] h-[300px] w-[300px] rounded-full bg-[#47D9FF]/10 blur-[120px]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(255,255,255,0.28),transparent_62%)]" />

      <Card className="relative w-full max-w-md overflow-hidden rounded-[26px] border border-white/25 bg-white/82 shadow-[0_28px_80px_rgba(0,0,0,0.22)] backdrop-blur-2xl">
        <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]" />

        <CardHeader className="space-y-3 px-5 pb-2 pt-5 sm:px-7 sm:pt-6">
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
              className="relative h-8 w-auto object-contain drop-shadow-sm sm:h-9"
              priority
            />
          </Link>

          <div className="space-y-1.5">
            <CardTitle className="max-w-md bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-[28px] font-semibold leading-tight tracking-[-0.04em] text-transparent sm:text-[32px]">
              Reprends ton fil là où tu l’as laissé.
            </CardTitle>

            <CardDescription className="max-w-sm text-sm leading-relaxed text-slate-500">
              Connecte-toi à Breezy avec ton adresse e-mail et ton mot de passe.
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="px-5 pb-5 sm:px-7 sm:pb-6">
          <form className="space-y-3" onSubmit={handleSubmit} noValidate>
            <div className="space-y-1.5">
              <label htmlFor="email" className="text-sm font-medium text-slate-700">
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
                  className="h-11 rounded-2xl border-white/70 bg-white/90 pl-11 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
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
              <label htmlFor="password" className="text-sm font-medium text-slate-700">
                Mot de passe
              </label>

              <div className="group relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition group-focus-within:text-[#5B6CFF]" />

                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder="••••••••"
                  className="h-11 rounded-2xl border-white/70 bg-white/90 pl-11 text-[15px] shadow-sm shadow-slate-200/60 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
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
              <div className="flex items-start gap-3 rounded-2xl border border-red-200 bg-red-50/90 px-4 py-2.5 text-sm text-red-700 shadow-sm">
                <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                <p>{errors.form}</p>
              </div>
            ) : null}

            <Button
              type="submit"
              className="h-11 w-full rounded-2xl bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] text-base font-semibold text-white shadow-[0_18px_44px_rgba(91,108,255,0.34)] transition duration-300 hover:scale-[1.015] hover:shadow-[0_24px_56px_rgba(91,108,255,0.42)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Connexion en cours…' : 'Se connecter'}
            </Button>

            <div className="space-y-3 pt-1">
              <div className="flex items-center gap-3">
                <div className="h-px flex-1 bg-gradient-to-r from-transparent via-slate-300 to-transparent" />

                <span className="text-xs font-medium text-slate-500">
                  Ou se connecter avec
                </span>

                <div className="h-px flex-1 bg-gradient-to-r from-transparent via-slate-300 to-transparent" />
              </div>

              <div className="grid grid-cols-2 gap-2.5">
                <button
                  type="button"
                  className="flex h-10 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md"
                >
                  <Image
                    src="/google-logo.jpg"
                    alt="Google"
                    width={18}
                    height={18}
                  />
                  <span className="text-sm font-medium text-slate-700">
                    Google
                  </span>
                </button>

                <button
                  type="button"
                  className="flex h-10 items-center justify-center gap-2 rounded-2xl border border-white/70 bg-white/85 transition hover:scale-[1.01] hover:bg-white hover:shadow-md"
                >
                  <Image
                    src="/microsoft-logo.png"
                    alt="Microsoft"
                    width={18}
                    height={18}
                  />
                  <span className="text-sm font-medium text-slate-700">
                    Microsoft
                  </span>
                </button>
              </div>
            </div>

            <p className="text-center text-sm text-slate-500">
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
    </main>
  )
}