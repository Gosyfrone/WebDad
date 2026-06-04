'use client'

import { useRef, useState } from 'react'
import { Camera } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { ProfilEditableFields } from '@/types'
import { useToast } from '@/hooks/use-toast'
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
  /** Appelé avec les valeurs validées après enregistrement. */
  onSave: (fields: ProfilEditableFields) => void
}

/**
 * Popup d'édition du profil : bannière, avatar, nom affiché et bio.
 *
 * La bannière et l'avatar se changent via un sélecteur de fichier avec aperçu
 * immédiat (lu en data URL, sans dépendance). Tant que le stockage côté serveur
 * n'existe pas, l'aperçu est local : il ne survit pas à un rechargement.
 * Le formulaire est réinitialisé aux valeurs courantes à chaque ouverture.
 */
export function EditProfilDialog({ children, initial, onSave }: EditProfilDialogProps) {
  const { toast } = useToast()
  const [open, setOpen] = useState(false)
  const [displayName, setDisplayName] = useState(initial.displayName)
  const [bio, setBio] = useState(initial.bio)
  const [avatarUrl, setAvatarUrl] = useState(initial.avatarUrl)
  const [bannerUrl, setBannerUrl] = useState(initial.bannerUrl)

  /** Recharge le formulaire avec les valeurs courantes à chaque ouverture. */
  function handleOpenChange(next: boolean) {
    if (next) {
      setDisplayName(initial.displayName)
      setBio(initial.bio)
      setAvatarUrl(initial.avatarUrl)
      setBannerUrl(initial.bannerUrl)
    }
    setOpen(next)
  }

  const trimmedName = displayName.trim()
  const bioRemaining = MAX_BIO - bio.length
  const nameTooLong = displayName.length > MAX_NAME
  const bioTooLong = bioRemaining < 0
  const canSave = trimmedName.length > 0 && !nameTooLong && !bioTooLong

  function handleSubmit() {
    if (!canSave) return
    // TODO (issue profil) : uploader les fichiers image vers le stockage via
    // l'API Gateway, puis envoyer les URLs persistées avec PATCH /profils/me.
    onSave({
      displayName: trimmedName,
      bio: bio.trim(),
      avatarUrl,
      bannerUrl,
    })
    setOpen(false)
    toast({ title: 'Profil mis à jour' })
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="gap-0 overflow-hidden border-[#D9C6FF]/70 bg-gradient-to-br from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] p-0 shadow-[0_28px_80px_rgba(91,108,255,0.24)] sm:max-w-lg">
        <DialogHeader className="border-b border-[#D9C6FF]/70 bg-gradient-to-r from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] p-4">
          <DialogTitle className="bg-gradient-to-r from-slate-950 via-[#5B6CFF] to-[#8D3DFF] bg-clip-text text-transparent">
            Éditer le profil
          </DialogTitle>
          <DialogDescription className="text-slate-600">
            Mettez à jour les informations visibles sur votre profil public.
          </DialogDescription>
        </DialogHeader>

        {/* Bannière éditable */}
        <ImagePicker
          label="Changer la bannière"
          onPick={setBannerUrl}
          className={cn(
            'relative flex h-36 w-full items-center justify-center overflow-hidden bg-cover bg-center',
            !bannerUrl &&
              'bg-gradient-to-r from-[#8D3DFF]/35 via-[#EADCFF] to-[#47D9FF]/25',
          )}
          style={bannerUrl ? { backgroundImage: `url(${bannerUrl})` } : undefined}
        />

        {/* Avatar éditable, superposé à la bannière */}
        <div className="px-4">
          <div className="-mt-12 w-fit">
            <ImagePicker
              label="Changer la photo de profil"
              onPick={setAvatarUrl}
              className="relative rounded-full"
            >
              <Avatar className="h-24 w-24 border-4 border-[#F8F3FF] shadow-[0_18px_44px_rgba(91,108,255,0.22)]">
                {avatarUrl && <AvatarImage src={avatarUrl} alt="" />}
                <AvatarFallback className="bg-gradient-to-br from-[#F8F3FF] via-white to-[#EEF9FF] text-2xl text-slate-950">
                  {trimmedName.charAt(0).toUpperCase() || '?'}
                </AvatarFallback>
              </Avatar>
            </ImagePicker>
          </div>
        </div>

        {/* Champs texte */}
        <div className="flex flex-col gap-4 p-4 pt-2">
          <Field label="Nom" htmlFor="profil-name">
            <Input
              id="profil-name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Votre nom"
              aria-invalid={nameTooLong}
              className="rounded-2xl border-white/70 bg-white/82 shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
            />
            {nameTooLong && (
              <p className="text-xs text-destructive">{MAX_NAME} caractères maximum.</p>
            )}
          </Field>

          <Field label="Bio" htmlFor="profil-bio">
            <textarea
              id="profil-bio"
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Parlez de vous en quelques mots…"
              rows={3}
              className="flex w-full resize-none rounded-2xl border border-white/70 bg-white/82 px-3 py-2 text-sm shadow-sm shadow-slate-200/50 transition-all placeholder:text-slate-400 hover:border-[#47D9FF]/70 focus-visible:border-[#5B6CFF] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#5B6CFF]/15"
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
        </div>

        <DialogFooter className="border-t border-[#D9C6FF]/70 bg-gradient-to-r from-[#F8F3FF] via-[#EADCFF] to-[#EEF9FF] p-4">
          <Button
            className="w-full rounded-full bg-gradient-to-r from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF] font-bold text-white shadow-[0_18px_44px_rgba(91,108,255,0.3)] transition hover:scale-[1.01] sm:w-auto"
            disabled={!canSave}
            onClick={handleSubmit}
          >
            Enregistrer
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface ImagePickerProps {
  /** Libellé accessible du bouton de sélection. */
  label: string
  /** Reçoit l'aperçu (data URL) du fichier choisi. */
  onPick: (dataUrl: string) => void
  className?: string
  style?: React.CSSProperties
  /** Contenu superposé (ex. l'avatar). */
  children?: React.ReactNode
}

/**
 * Zone cliquable qui ouvre un sélecteur de fichier image et restitue le
 * fichier choisi en data URL (aperçu local). Affiche une pastille « appareil
 * photo » au survol, par-dessus un contenu optionnel.
 */
function ImagePicker({ label, onPick, className, style, children }: ImagePickerProps) {
  const inputRef = useRef<HTMLInputElement>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => onPick(reader.result as string)
    reader.readAsDataURL(file)
    // Permet de re-sélectionner le même fichier deux fois de suite.
    e.target.value = ''
  }

  return (
    <button
      type="button"
      aria-label={label}
      onClick={() => inputRef.current?.click()}
      className={cn('group cursor-pointer', className)}
      style={style}
    >
      {children}
      {/* Voile + icône appareil photo */}
      <span className="absolute inset-0 flex items-center justify-center rounded-[inherit] bg-black/30 opacity-0 transition-opacity group-hover:opacity-100">
        <Camera className="h-6 w-6 text-white" aria-hidden />
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
