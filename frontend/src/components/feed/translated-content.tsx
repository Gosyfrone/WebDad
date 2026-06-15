'use client'

import { useEffect, useState } from 'react'
import { Languages, Loader2 } from 'lucide-react'

import {
  getPageLanguage,
  isTranslationCandidate,
  translatePostContent,
  type PostTranslation,
} from '@/lib/post-translation'
import { cn } from '@/lib/utils'
import { MentionText } from '@/components/mention/mention-text'
import { useLanguage } from '@/components/language-provider'

interface TranslatedContentProps {
  contentId: string
  content: string
  className?: string
  indicatorClassName?: string
}

export function TranslatedContent({
  contentId,
  content,
  className,
  indicatorClassName,
}: TranslatedContentProps) {
  const { locale, t } = useLanguage()
  const [translation, setTranslation] = useState<PostTranslation | null>(null)
  const [translating, setTranslating] = useState(false)
  const [showOriginal, setShowOriginal] = useState(false)

  const displayedContent = translation && !showOriginal ? translation.translatedText : content

  useEffect(() => {
    let active = true
    const targetLanguage = locale || getPageLanguage()

    setTranslation(null)
    setShowOriginal(false)

    if (!isTranslationCandidate(content, targetLanguage)) {
      setTranslating(false)
      return () => {
        active = false
      }
    }

    setTranslating(true)

    translatePostContent(contentId, content, targetLanguage)
      .then((result) => {
        if (!active) return
        setTranslation(result)
      })
      .finally(() => {
        if (!active) return
        setTranslating(false)
      })

    return () => {
      active = false
    }
  }, [contentId, content, locale])

  return (
    <>
      <MentionText text={displayedContent} className={cn('block', className)} />

      {(translation || translating) && (
        <div
          className={cn(
            'mt-1 flex min-h-6 flex-wrap items-center gap-2 text-xs text-muted-foreground',
            indicatorClassName,
          )}
        >
          <Languages className="h-3.5 w-3.5 shrink-0" />
          {translation ? (
            <>
              <span>
                {showOriginal
                  ? t('translation.original')
                  : `${t('translation.automatic')}${translation.detectedSourceLanguage ? ` ${t('translation.from', { language: translation.detectedSourceLanguage.toUpperCase() })}` : ''}`}
              </span>
              <button
                type="button"
                onClick={() => setShowOriginal((value) => !value)}
                className="font-semibold text-primary transition hover:underline"
              >
                {showOriginal
                  ? t('translation.show_translation')
                  : t('translation.show_original')}
              </button>
            </>
          ) : (
            <span className="inline-flex items-center gap-1">
              <Loader2 className="h-3 w-3 animate-spin" />
              {t('translation.loading')}
            </span>
          )}
        </div>
      )}
    </>
  )
}
