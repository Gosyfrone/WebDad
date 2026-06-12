'use client'

import { useCallback, useEffect, useState } from 'react'
import { ChevronLeft, RotateCcw } from 'lucide-react'

import { hslTripletToHex } from '@/lib/color'
import {
  applyCustomTheme,
  EMPTY_CUSTOM_THEME,
  readCustomTheme,
  writeCustomTheme,
  type CustomTheme,
  type CustomThemeTarget,
} from '@/lib/custom-theme'
import { useT } from '@/components/language-provider'
import { Button } from '@/components/ui/button'
import { ColorWheel } from '@/components/ui/color-wheel'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

/**
 * Popup de thème personnalisé : trois rectangles affichant les couleurs Breezy
 * courantes (mode clair OU sombre). Cliquer un rectangle ouvre l'éditeur (roue
 * HSV + hex + RGB) pour cette cible :
 *   1. Fond d'écran      (--background / --bg-page)
 *   2. Couleur du texte  (--foreground …)   « Pour toi », « Abonnements »…
 *   3. Boutons d'action  (--primary …)      « Breezer », « Suivre »…
 *
 * Application EN DIRECT (on voit le changement immédiatement derrière la popup)
 * + persistance localStorage. Bouton « Réinitialiser » pour revenir au thème
 * Breezy d'origine. Composant contrôlé via `open` / `onOpenChange`.
 */

interface CustomThemeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/** Variable CSS à relire pour afficher la couleur courante de chaque cible. */
const READ_VAR: Record<CustomThemeTarget, string> = {
  background: '--background',
  text: '--foreground',
  primary: '--primary',
}

const TARGET_LABEL: Record<CustomThemeTarget, string> = {
  background: 'theme.target_background',
  text: 'theme.target_text',
  primary: 'theme.target_primary',
}

const TARGET_HINT: Record<CustomThemeTarget, string> = {
  background: 'theme.target_background_hint',
  text: 'theme.target_text_hint',
  primary: 'theme.target_primary_hint',
}

const TARGETS: CustomThemeTarget[] = ['background', 'text', 'primary']

/** Lit la couleur effective d'une cible (custom prioritaire, sinon thème courant). */
function readEffectiveColor(target: CustomThemeTarget, theme: CustomTheme): string {
  const custom = theme[target]
  if (custom) return custom
  if (typeof window !== 'undefined') {
    const triplet = getComputedStyle(document.documentElement)
      .getPropertyValue(READ_VAR[target])
      .trim()
    const hex = hslTripletToHex(triplet)
    if (hex) return hex
  }
  return '#000000'
}

export function CustomThemeDialog({ open, onOpenChange }: CustomThemeDialogProps) {
  const t = useT()
  const [theme, setTheme] = useState<CustomTheme>(EMPTY_CUSTOM_THEME)
  const [editing, setEditing] = useState<CustomThemeTarget | null>(null)
  // Couleur effective affichée sur chaque rectangle (recalculée à l'ouverture).
  const [swatches, setSwatches] = useState<Record<CustomThemeTarget, string>>({
    background: '#ffffff',
    text: '#000000',
    primary: '#e053ff',
  })

  const refreshSwatches = useCallback((current: CustomTheme) => {
    setSwatches({
      background: readEffectiveColor('background', current),
      text: readEffectiveColor('text', current),
      primary: readEffectiveColor('primary', current),
    })
  }, [])

  // À l'ouverture : relit le thème persisté + les couleurs courantes (le mode
  // clair/sombre a pu changer depuis la dernière fois).
  useEffect(() => {
    if (!open) {
      setEditing(null)
      return
    }
    const stored = readCustomTheme()
    setTheme(stored)
    refreshSwatches(stored)
  }, [open, refreshSwatches])

  /** Applique + persiste une nouvelle couleur pour la cible en cours d'édition. */
  function handleColorChange(hex: string) {
    if (!editing) return
    const next = { ...theme, [editing]: hex }
    setTheme(next)
    applyCustomTheme(next)
    writeCustomTheme(next)
    setSwatches((s) => ({ ...s, [editing]: hex }))
  }

  /** Réinitialise tout au thème Breezy d'origine. */
  function handleReset() {
    setTheme(EMPTY_CUSTOM_THEME)
    applyCustomTheme(EMPTY_CUSTOM_THEME)
    writeCustomTheme(EMPTY_CUSTOM_THEME)
    setEditing(null)
    // Laisse le DOM revenir à la feuille de style avant de relire les couleurs.
    requestAnimationFrame(() => refreshSwatches(EMPTY_CUSTOM_THEME))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        {editing ? (
          <>
            <DialogHeader>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setEditing(null)}
                  aria-label={t('theme.back')}
                  className="rounded-full p-1 transition-colors hover:bg-accent"
                >
                  <ChevronLeft className="h-5 w-5" />
                </button>
                <DialogTitle>{t(TARGET_LABEL[editing])}</DialogTitle>
              </div>
              <DialogDescription>{t(TARGET_HINT[editing])}</DialogDescription>
            </DialogHeader>
            <ColorWheel value={swatches[editing]} onChange={handleColorChange} />
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{t('theme.customize')}</DialogTitle>
              <DialogDescription>{t('theme.customize_hint')}</DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-3">
              {TARGETS.map((target) => (
                <button
                  key={target}
                  type="button"
                  onClick={() => setEditing(target)}
                  className="flex items-center gap-3 rounded-xl border p-3 text-left transition-colors hover:bg-accent"
                >
                  <span
                    className="h-12 w-16 shrink-0 rounded-lg border shadow-inner"
                    style={{ backgroundColor: swatches[target] }}
                    aria-hidden
                  />
                  <span className="flex min-w-0 flex-col">
                    <span className="text-sm font-medium">{t(TARGET_LABEL[target])}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {t(TARGET_HINT[target])}
                    </span>
                  </span>
                  <span className="ml-auto font-mono text-xs uppercase text-muted-foreground">
                    {swatches[target]}
                  </span>
                </button>
              ))}
            </div>

            <Button
              type="button"
              variant="outline"
              onClick={handleReset}
              className="w-full gap-2"
            >
              <RotateCcw className="h-4 w-4" />
              {t('theme.reset')}
            </Button>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
