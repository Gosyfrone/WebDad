'use client'

import Image from 'next/image'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { CircleAlert, Lock, Mail } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/components/ui/button'
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
    <main className="relative h-dvh w-full overflow-hidden bg-slate-50 px-4 py-4 sm:px-6 sm:py-6 lg:mx-auto lg:h-[calc(100vh-1.5rem)] lg:rounded-[2rem] lg:border lg:border-white/60 lg:bg-white/80 lg:shadow-[0_24px_80px_rgba(15,23,42,0.12)] lg:backdrop-blur-xl">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(224,83,255,0.14),_transparent_34%),radial-gradient(circle_at_bottom_right,_rgba(56,189,248,0.12),_transparent_28%)]" />
      <div className="relative grid h-full min-h-0 gap-5 xl:grid-cols-[1.02fr_0.98fr] xl:gap-0">
        <section className="flex min-h-0 xl:p-8">
          <div className="flex h-full w-full max-w-xl flex-col gap-4 sm:gap-5">
            <div className="flex justify-start xl:hidden">
              <Link href={ROUTES.home} className="inline-flex items-center">
                <Image
                  src="/logo_breezy.png"
                  alt="Breezy"
                  width={1106}
                  height={336}
                  className="h-10 w-auto object-contain sm:h-12"
                  priority
                />
              </Link>
            </div>

            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl xl:text-5xl">
              Reprends ton fil là où tu l’as laissé.
            </h1>

            <form className="space-y-4 sm:space-y-5" onSubmit={handleSubmit} noValidate>
              <div className="space-y-1 pt-0.5 sm:space-y-2">
                <h2 className="text-xl font-semibold tracking-tight text-slate-950 sm:text-2xl xl:text-3xl">
                  Connexion
                </h2>
                <p className="max-w-md text-sm text-slate-600 sm:text-base">
                  Connecte-toi avec ton adresse e-mail et ton mot de passe.
                </p>
              </div>

              <div className="space-y-3 sm:space-y-4">
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
                      className="h-10 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-11"
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
                      className="h-10 rounded-xl border-slate-200 bg-white pl-10 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#e053ff] sm:h-11"
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
                  className="h-10 w-full rounded-xl bg-[#e053ff] text-white shadow-[0_18px_36px_rgba(224,83,255,0.28)] transition-transform hover:translate-y-[-1px] hover:bg-[#cf42f0] sm:h-11"
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
              </div>
            </form>
          </div>
        </section>

        <section className="hidden min-h-[120px] items-center justify-center px-2 pb-0 sm:min-h-[150px] sm:px-4 sm:pb-1 xl:flex xl:min-h-0 xl:p-8">
          <Link
            href={ROUTES.home}
            className="group relative flex h-full w-full items-center justify-center overflow-hidden p-1 transition-all duration-300 sm:p-4 xl:p-10"
          >
            <div className="absolute inset-0 rounded-[2rem] bg-[radial-gradient(circle_at_center,_rgba(224,83,255,0.12),_transparent_58%)] opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
            <Image
              src="/logo_breezy.png"
              alt="Breezy"
              width={1106}
              height={336}
              className="relative h-auto w-[74%] max-w-[16rem] object-contain transition-transform duration-300 group-hover:scale-[1.03] group-hover:drop-shadow-[0_0_28px_rgba(224,83,255,0.30)] sm:w-[78%] sm:max-w-[22rem] xl:w-[92%] xl:max-w-[34rem]"
              priority
            />
          </Link>
        </section>
      </div>
    </main>
  )
}
