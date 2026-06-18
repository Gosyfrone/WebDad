'use client'

import { useCallback, useEffect, useState } from 'react'
import { Loader2, Save } from 'lucide-react'

import { getReportSettings, updateReportSettings } from '@/lib/reports'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

/**
 * Paramètres de modération réglables par l'administrateur. Pour l'instant : le
 * SEUIL d'auto-masquage — nombre de signalements à partir duquel un post est
 * automatiquement masqué (et mis en attente d'une décision de modérateur). 0
 * désactive l'auto-masquage. Lecture mod/admin, écriture admin (gating back).
 */
export function ModerationSettings() {
  const t = useT()
  const { toast } = useToast()
  const [threshold, setThreshold] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const s = await getReportSettings()
      setThreshold(String(s.autoHideThreshold))
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function onSave() {
    const n = Number(threshold)
    if (!Number.isInteger(n) || n < 0) {
      toast({ title: t('admin.settings.invalid'), variant: 'brand' })
      return
    }
    setSaving(true)
    try {
      const s = await updateReportSettings(n)
      setThreshold(String(s.autoHideThreshold))
      toast({ title: t('admin.settings.saved') })
    } catch {
      toast({ title: t('admin.settings.save_failed'), variant: 'brand' })
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-4 px-4 py-4">
      <div className="flex flex-col gap-3 rounded-xl border p-4">
        <div>
          <h2 className="text-sm font-bold">{t('admin.settings.auto_hide_title')}</h2>
          <p className="text-xs text-muted-foreground">{t('admin.settings.auto_hide_desc')}</p>
        </div>

        {loading ? (
          <div className="flex items-center py-4 text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
          </div>
        ) : error ? (
          <p className="py-2 text-sm text-muted-foreground">{t('admin.settings.error')}</p>
        ) : (
          <div className="flex items-end gap-3">
            <label className="flex flex-col gap-1 text-xs font-medium text-muted-foreground">
              {t('admin.settings.auto_hide_label')}
              <Input
                type="number"
                min={0}
                value={threshold}
                onChange={(e) => setThreshold(e.target.value)}
                className="h-9 w-28"
              />
            </label>
            <Button size="sm" disabled={saving} onClick={() => void onSave()}>
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="mr-1.5 h-4 w-4" />}
              {t('common.save')}
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
