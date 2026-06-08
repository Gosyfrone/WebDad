'use client'

import Image from 'next/image'
import { usePathname } from 'next/navigation'
import { useCallback, useEffect, useRef, useState } from 'react'

const SHOW_DELAY_MS = 700
const MIN_VISIBLE_MS = 450
const MAX_VISIBLE_MS = 5000

function isModifiedClick(event: MouseEvent) {
  return event.metaKey || event.ctrlKey || event.shiftKey || event.altKey
}

function shouldShowLoader(anchor: HTMLAnchorElement) {
  const href = anchor.getAttribute('href')

  if (
    !href ||
    href.startsWith('#') ||
    href.startsWith('mailto:') ||
    href.startsWith('tel:') ||
    anchor.target === '_blank' ||
    anchor.hasAttribute('download')
  ) {
    return false
  }

  const destination = new URL(href, window.location.href)

  if (destination.origin !== window.location.origin) {
    return false
  }

  return destination.href !== window.location.href
}

export function NavigationLoader() {
  const pathname = usePathname()
  const [visible, setVisible] = useState(false)
  const visibleRef = useRef(false)
  const startedAtRef = useRef(0)
  const showTimerRef = useRef<number | null>(null)
  const hideTimerRef = useRef<number | null>(null)
  const fallbackTimerRef = useRef<number | null>(null)

  const clearTimers = useCallback(() => {
    if (showTimerRef.current !== null) {
      window.clearTimeout(showTimerRef.current)
      showTimerRef.current = null
    }

    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current)
      hideTimerRef.current = null
    }

    if (fallbackTimerRef.current !== null) {
      window.clearTimeout(fallbackTimerRef.current)
      fallbackTimerRef.current = null
    }
  }, [])

  const hideLoader = useCallback(() => {
    visibleRef.current = false
    setVisible(false)
    clearTimers()
  }, [clearTimers])

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (event.defaultPrevented || event.button !== 0 || isModifiedClick(event)) {
        return
      }

      const target = event.target instanceof Element ? event.target : null
      const anchor = target?.closest('a[href]')

      if (!(anchor instanceof HTMLAnchorElement) || !shouldShowLoader(anchor)) {
        return
      }

      clearTimers()
      visibleRef.current = false
      setVisible(false)

      showTimerRef.current = window.setTimeout(() => {
        startedAtRef.current = window.performance.now()
        visibleRef.current = true
        setVisible(true)

        fallbackTimerRef.current = window.setTimeout(() => {
          hideLoader()
        }, MAX_VISIBLE_MS)
      }, SHOW_DELAY_MS)
    }

    document.addEventListener('click', handleClick, true)

    return () => {
      document.removeEventListener('click', handleClick, true)
      clearTimers()
    }
  }, [clearTimers, hideLoader])

  useEffect(() => {
    if (!visibleRef.current && showTimerRef.current !== null) {
      hideLoader()
      return
    }

    if (!visibleRef.current) {
      return
    }

    const elapsed = window.performance.now() - startedAtRef.current
    const remaining = Math.max(MIN_VISIBLE_MS - elapsed, 0)

    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current)
    }

    hideTimerRef.current = window.setTimeout(() => {
      hideLoader()
    }, remaining)
  }, [hideLoader, pathname])

  if (!visible) {
    return null
  }

  return (
    <div
      aria-label="Chargement"
      aria-live="polite"
      className="breezy-navigation-loader"
      role="status"
      style={{
        alignItems: 'center',
        background:
          'linear-gradient(140deg, rgba(248, 243, 255, 0.94) 0%, rgba(234, 220, 255, 0.94) 32%, rgba(235, 248, 255, 0.94) 100%)',
        display: 'flex',
        inset: 0,
        justifyContent: 'center',
        position: 'fixed',
        zIndex: 2147483647,
      }}
    >
      <style>
        {`
          @keyframes breezy-loader-pulse {
            0%, 100% {
              opacity: 0.86;
              transform: scale(0.96);
            }
            50% {
              opacity: 1;
              transform: scale(1.08);
            }
          }

          .breezy-navigation-loader {
            transition: background 180ms ease;
          }

          .dark .breezy-navigation-loader {
            background:
              radial-gradient(circle at 50% 42%, rgba(141, 61, 255, 0.18), transparent 34%),
              linear-gradient(140deg, rgba(11, 7, 18, 0.96) 0%, rgba(20, 12, 36, 0.96) 38%, rgba(14, 10, 28, 0.96) 100%) !important;
          }

          .breezy-navigation-loader-logo {
            animation: breezy-loader-pulse 1.35s ease-in-out infinite;
            transform-origin: center;
          }
        `}
      </style>
      <Image
        src="/logo_only.png"
        alt="Breezy"
        className="breezy-navigation-loader-logo"
        width={112}
        height={112}
        priority
        style={{
          display: 'block',
          filter: 'drop-shadow(0 18px 42px rgba(91, 108, 255, 0.34))',
          height: 112,
          objectFit: 'contain',
          width: 112,
        }}
      />
    </div>
  )
}
