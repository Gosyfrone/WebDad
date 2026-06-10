'use client'

import { useRef, useState } from 'react'
import { Camera, Loader2 } from 'lucide-react'

import { cn } from '@/lib/utils'
import { mediaUrl, uploadMedia } from '@/lib/media'
import type { ProfilEditableFields } from '@/types'
import { useToast } from '@/hooks/use-toast'
import { useLanguage } from '@/components/language-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

/** Longueur maximale de la bio (alignée sur la convention X). */
const MAX_BIO = 160
/** Longueur maximale du nom affiché. */
const MAX_NAME = 50

interface EditProfilDialogProps {
  /** Élément déclencheur (ex. bouton « Éditer le profil »), rendu via `asChild`. */
  children: React.ReactNode
  /** Valeurs initiales du formulaire. */
  initial: ProfilEditableFields
  birthDateLocked: boolean
  genderLocked: boolean
  displayNameChangedAt: string
  saving?: boolean
  /** Appelé avec les valeurs validées après enregistrement. */
  onSave: (fields: ProfilEditableFields) => Promise<void>
}

/**
 * Popup d'édition du profil : bannière, avatar, nom affiché et bio.
 *
 * La bannière et l'avatar se changent via un sélecteur de fichier : le fichier
 * est **uploadé au media-service** (`uploadMedia`) et son URL est stockée puis
 * persistée par `PATCH /profils/me` → l'image survit au rechargement.
 * Le formulaire est réinitialisé aux valeurs courantes à chaque ouverture.
 */
export function EditProfilDialog({
  children,
  initial,
  birthDateLocked,
  genderLocked,
  displayNameChangedAt,
  saving = false,
  onSave,
}: EditProfilDialogProps) {
  const { t, locale } = useLanguage()
  const { toast } = useToast()
  const [open, setOpen] = useState(false)
  const [displayName, setDisplayName] = useState(initial.displayName)
  const [bio, setBio] = useState(initial.bio)
  const [avatarUrl, setAvatarUrl] = useState(initial.avatarUrl)
  const [bannerUrl, setBannerUrl] = useState(initial.bannerUrl)
  const [website, setWebsite] = useState(initial.website)
  const [location, setLocation] = useState(initial.location)
  const [birthDate, setBirthDate] = useState(initial.birthDate)
  const [gender, setGender] = useState<ProfilEditableFields['gender']>(initial.gender)

  /** Recharge le formulaire avec les valeurs courantes à chaque ouverture. */
  function handleOpenChange(next: boolean) {
    if (next) {
      setDisplayName(initial.displayName)
      setBio(initial.bio)
      setAvatarUrl(initial.avatarUrl)
      setBannerUrl(initial.bannerUrl)
      setWebsite(initial.website)
      setLocation(initial.location)
      setBirthDate(initial.birthDate)
      setGender(initial.gender)
    }
    setOpen(next)
  }

  const trimmedName = displayName.trim()
  const bioRemaining = MAX_BIO - bio.length
  const nameTooLong = displayName.length > MAX_NAME
  const bioTooLong = bioRemaining < 0
  const displayNameChanged = trimmedName !== initial.displayName
  const nextDisplayNameDate = getNextDisplayNameDate(displayNameChangedAt)
  const displayNameLocked = displayNameChanged && nextDisplayNameDate > new Date()
  const canSave = trimmedName.length > 0 && !nameTooLong && !bioTooLong && !displayNameLocked

  async function handleSubmit() {
    if (!canSave) return
    try {
      await onSave({
        displayName: trimmedName,
        bio: bio.trim(),
        avatarUrl,
        bannerUrl,
        website: website.trim(),
        location: location.trim(),
        birthDate: birthDateLocked ? '' : birthDate,
        gender: genderLocked ? '' : gender,
      })
      setOpen(false)
    } catch {
      // Le parent affiche déjà le toast d'erreur ; on garde la popup ouverte.
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="panel grid max-h-[calc(100dvh-1rem)] w-[calc(100vw-1rem)] grid-rows-[auto,minmax(0,1fr),auto] gap-0 overflow-hidden border p-0 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-h-[calc(100dvh-3rem)] sm:max-w-lg">
        <DialogHeader className="shrink-0 border-b p-4">
          <DialogTitle className="brand-text">{t('profil.edit')}</DialogTitle>
          <DialogDescription className="text-muted-foreground">
            {t('editprofil.desc')}
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 overflow-y-auto overscroll-contain">
          {/* Bannière éditable */}
          <ImagePicker
            label={t('editprofil.change_banner')}
            onPick={setBannerUrl}
            onError={() => toast({ title: t('editprofil.upload_failed'), variant: 'destructive' })}
            className={cn(
              'relative flex h-36 w-full items-center justify-center overflow-hidden bg-cover bg-center',
              !bannerUrl &&
                'bg-gradient-to-r from-[#8D3DFF]/35 via-[#EADCFF] to-[#47D9FF]/25 dark:from-[#8D3DFF]/45 dark:via-[#1c1338] dark:to-[#47D9FF]/35',
            )}
            style={bannerUrl ? { backgroundImage: `url(${bannerUrl})` } : undefined}
          />

          {/* Avatar éditable, superposé à la bannière */}
          <div className="px-4">
            <div className="-mt-12 w-fit">
              <ImagePicker
                label={t('editprofil.change_avatar')}
                onPick={setAvatarUrl}
                onError={() => toast({ title: t('editprofil.upload_failed'), variant: 'destructive' })}
                className="relative rounded-full"
              >
                <Avatar className="h-24 w-24 border-4 border-[#F8F3FF] shadow-[0_18px_44px_rgba(91,108,255,0.22)] dark:border-[#171026]">
                  {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
                  <AvatarFallback className="bg-gradient-to-br from-[#F8F3FF] via-white to-[#EEF9FF] text-2xl text-slate-950 dark:from-[#1c1338] dark:via-[#171026] dark:to-[#141a2e] dark:text-white">
                    {trimmedName.charAt(0).toUpperCase() || '?'}
                  </AvatarFallback>
                </Avatar>
              </ImagePicker>
            </div>
          </div>

          {/* Champs texte */}
          <div className="flex flex-col gap-4 p-4 pt-2">
            <Field label={t('editprofil.name_label')} htmlFor="profil-name">
              <Input
                id="profil-name"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder={t('editprofil.name_placeholder')}
                aria-invalid={nameTooLong}
                className="rounded-2xl border-white/70 bg-white/82 shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5 dark:placeholder:text-muted-foreground"
              />
              {nameTooLong && (
                <p className="text-xs text-destructive">
                  {t('editprofil.name_max', { count: MAX_NAME })}
                </p>
              )}
              {displayNameLocked && (
                <p className="text-xs text-destructive">
                  {t('editprofil.name_locked', { date: formatDate(nextDisplayNameDate, locale) })}
                </p>
              )}
            </Field>

            <Field label={t('editprofil.bio_label')} htmlFor="profil-bio">
              <textarea
                id="profil-bio"
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                placeholder={t('editprofil.bio_placeholder')}
                rows={3}
                className="flex w-full resize-none rounded-2xl border border-white/70 bg-white/82 px-3 py-2 text-sm shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15 dark:border-white/15 dark:bg-white/5 dark:placeholder:text-muted-foreground"
              />
              <span
                className={cn(
                  'self-end text-xs',
                  bioTooLong ? 'font-bold text-destructive' : 'text-muted-foreground',
                )}
              >
                {bioRemaining}
              </span>
            </Field>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t('editprofil.location_label')} htmlFor="profil-location">
                <Input
                  id="profil-location"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  placeholder={t('editprofil.location_placeholder')}
                  className="rounded-2xl border-white/70 bg-white/82 shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                />
              </Field>

              <Field label={t('editprofil.website_label')} htmlFor="profil-website">
                <Input
                  id="profil-website"
                  value={website}
                  onChange={(e) => setWebsite(e.target.value)}
                  placeholder="https://..."
                  className="rounded-2xl border-white/70 bg-white/82 shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
                />
              </Field>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t('auth.register.birthdate_label')} htmlFor="profil-birth-date">
                <Input
                  id="profil-birth-date"
                  type="date"
                  value={birthDate}
                  onChange={(e) => setBirthDate(e.target.value)}
                  disabled={birthDateLocked}
                  title={birthDateLocked ? t('editprofil.birthdate_locked') : undefined}
                  className={cn(
                    'rounded-2xl border-white/70 bg-white/82 shadow-sm shadow-slate-200/50 transition-all hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15',
                    birthDateLocked &&
                      'cursor-not-allowed border-slate-200 bg-slate-100 text-slate-500 hover:border-slate-200',
                  )}
                />
              </Field>

              <Field label={t('auth.register.gender_label')} htmlFor="profil-gender">
                <select
                  id="profil-gender"
                  value={gender}
                  onChange={(e) => setGender(e.target.value as ProfilEditableFields['gender'])}
                  disabled={genderLocked}
                  title={genderLocked ? t('editprofil.gender_locked') : undefined}
                  className={cn(
                    'flex h-10 w-full rounded-2xl border border-white/70 bg-white/82 px-3 py-2 text-sm shadow-sm shadow-slate-200/50 transition-all hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15',
                    genderLocked &&
                      'cursor-not-allowed border-slate-200 bg-slate-100 text-slate-500 hover:border-slate-200',
                  )}
                >
                  <option value="">{t('editprofil.gender_none')}</option>
                  <option value="female">{t('auth.register.gender_female')}</option>
                  <option value="male">{t('auth.register.gender_male')}</option>
                </select>
              </Field>
            </div>
          </div>
        </div>

        <DialogFooter className="shrink-0 border-t p-4">
          <Button
            className="w-full rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.01] sm:w-auto"
            disabled={!canSave || saving}
            onClick={handleSubmit}
          >
            {saving ? t('editprofil.saving') : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface ImagePickerProps {
  /** Libellé accessible du bouton de sélection. */
  label: string
  /** Reçoit l'URL (absolue, gateway) du média uploadé. */
  onPick: (url: string) => void
  /** Appelé si l'upload échoue (pour afficher un toast côté parent). */
  onError?: (message: string) => void
  className?: string
  style?: React.CSSProperties
  /** Contenu superposé (ex. l'avatar). */
  children?: React.ReactNode
}

/**
 * Zone cliquable qui ouvre un sélecteur de fichier image, **uploade** le fichier
 * choisi au media-service et restitue son URL (prête à l'affichage). Affiche un
 * voile « appareil photo » au survol et un spinner pendant l'upload.
 */
function ImagePicker({ label, onPick, onError, className, style, children }: ImagePickerProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)

  async function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    // Permet de re-sélectionner le même fichier deux fois de suite.
    e.target.value = ''
    if (!file) return

    setUploading(true)
    try {
      const { id } = await uploadMedia(file)
      onPick(mediaUrl(id))
    } catch (err) {
      onError?.(err instanceof Error ? err.message : 'upload failed')
    } finally {
      setUploading(false)
    }
  }

  return (
    <button
      type="button"
      aria-label={label}
      aria-busy={uploading}
      disabled={uploading}
      onClick={() => inputRef.current?.click()}
      className={cn('group cursor-pointer', className)}
      style={style}
    >
      {children}
      {/* Voile + icône appareil photo (ou spinner pendant l'upload) */}
      <span
        className={cn(
          'absolute inset-0 flex items-center justify-center rounded-[inherit] bg-black/30 transition-opacity',
          uploading ? 'opacity-100' : 'opacity-0 group-hover:opacity-100',
        )}
      >
        {uploading ? (
          <Loader2 className="h-6 w-6 animate-spin text-white" aria-hidden />
        ) : (
          <Camera className="h-6 w-6 text-white" aria-hidden />
        )}
      </span>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        onChange={handleChange}
        className="sr-only"
      />
    </button>
  )
}

function Field({
  label,
  htmlFor,
  children,
}: {
  label: string
  htmlFor: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={htmlFor} className="text-sm font-medium">
        {label}
      </label>
      {children}
    </div>
  )
}

function getNextDisplayNameDate(changedAt: string): Date {
  if (!changedAt) return new Date(0)
  const date = new Date(changedAt)
  if (Number.isNaN(date.getTime())) return new Date(0)
  date.setDate(date.getDate() + 7)
  return date
}

function formatDate(date: Date, locale: string): string {
  return new Intl.DateTimeFormat(locale === 'en' ? 'en-US' : 'fr-FR', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(date)
}
