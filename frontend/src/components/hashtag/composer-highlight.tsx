'use client'

import { forwardRef } from 'react'

import { parseHashtagSegments } from '@/lib/hashtags'
import { cn } from '@/lib/utils'

export const ComposerHighlight = forwardRef<HTMLDivElement, { text: string; className?: string }>(
  function ComposerHighlight({ text, className }, ref) {
  const segments = parseHashtagSegments(text)
  return (
    <div
      ref={ref}
      aria-hidden
      className={cn(
        'pointer-events-none absolute inset-0 overflow-hidden whitespace-pre-wrap break-words text-xl',
        className,
      )}
    >
      {segments.map((segment, index) =>
        segment.type === 'hashtag' ? (
          <span
            key={index}
            className="font-semibold text-[#1D9BF0] underline decoration-[#1D9BF0]/70 underline-offset-2"
          >
            {segment.raw}
          </span>
        ) : (
          <span key={index} className="text-foreground">
            {segment.text}
          </span>
        ),
      )}
      {text.endsWith('\n') && ' '}
    </div>
  )
})
