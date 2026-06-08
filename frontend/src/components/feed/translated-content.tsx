'use client'

import { useEffect, useState } from 'react'
import { Languages, Loader2 } from 'lucide-react'

import {
  getPageLanguage,
  translatePostContent,
  type PostTranslation,
} from '@/lib/post-translation'
import { cn } from '@/lib/utils'

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
  const [translation, setTranslation] = useState<PostTranslation | null>(null)
  const [translating, setTranslating] = useState(false)
  const [showOriginal, setShowOriginal] = useState(false)

  const displayedContent = translation && !showOriginal ? translation.translatedText : content

  useEffect(() => {
    let active = true
    const targetLanguage = getPageLanguage()

    setTranslation(null)
    setShowOriginal(false)
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
  }, [contentId, content])

  return (
    <>
      <p className={className}>{displayedContent}</p>

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
                  ? 'Texte original'
                  : `Traduit automatiquement${translation.detectedSourceLanguage ? ` depuis ${translation.detectedSourceLanguage.toUpperCase()}` : ''}`}
              </span>
              <button
                type="button"
                onClick={() => setShowOriginal((value) => !value)}
                className="font-semibold text-primary transition hover:underline"
              >
                {showOriginal ? 'Voir la traduction' : "Voir l'original"}
              </button>
            </>
          ) : (
            <span className="inline-flex items-center gap-1">
              <Loader2 className="h-3 w-3 animate-spin" />
              Traduction...
            </span>
          )}
        </div>
      )}
    </>
  )
}
