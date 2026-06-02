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
  if (typeof error === 'string' && error.trim()) {
    return error
  }

  if (error instanceof Error && error.message.trim()) {
    return error.message
  }

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

    if (Object.keys(nextErrors).length > 0) {
      return
    }

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
              Reprends ton fil là où tu l’as laissé.
            </CardTitle>
            <CardDescription className="max-w-md text-sm text-slate-600 sm:text-base">
              Connecte-toi avec ton adresse e-mail et ton mot de passe.
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="px-5 pb-5 sm:px-8 sm:pb-8">
          <form className="space-y-4 sm:space-y-5" onSubmit={handleSubmit} noValidate>
            <div className="space-y-2">
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
              {errors.email ? (
                <p id="email-error" className="text-sm text-red-600">
                  {errors.email}
                </p>
              ) : null}
            </div>

            <div className="space-y-2">
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
                  autoComplete="current-password"
                  placeholder="••••••••"
                  className="h-11 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff]"
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
                <p id="password-error" className="text-sm text-red-600">
                  {errors.password}
                </p>
              ) : null}
            </div>

            {errors.form ? (
              <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                <p>{errors.form}</p>
              </div>
            ) : null}

            <Button
              type="submit"
              className="h-11 w-full rounded-xl bg-[#e053ff] text-white shadow-[0_18px_36px_rgba(224,83,255,0.28)] transition-transform hover:translate-y-[-1px] hover:bg-[#cf42f0]"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Connexion en cours…' : 'Se connecter'}
            </Button>

            <p className="pb-1 text-center text-sm text-slate-600 sm:pb-0">
              Pas encore de compte ?{' '}
              <Link
                href={ROUTES.register}
                className="font-medium text-[#e053ff] underline-offset-4 hover:underline"
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
