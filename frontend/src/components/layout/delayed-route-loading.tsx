'use client'

import Image from 'next/image'
import { useEffect, useState } from 'react'

const SHOW_DELAY_MS = 700

export function DelayedRouteLoading() {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setVisible(true)
    }, SHOW_DELAY_MS)

    return () => window.clearTimeout(timer)
  }, [])

  if (!visible) {
    return null
  }

  return (
    <div
      className="breezy-route-loading"
      style={{
        alignItems: 'center',
        background:
          'linear-gradient(140deg, #f8f3ff 0%, #eadcff 28%, #d9c6ff 62%, #ebe8ff 100%)',
        display: 'flex',
        justifyContent: 'center',
        minHeight: '100vh',
        overflow: 'hidden',
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

          .dark .breezy-route-loading {
            background:
              radial-gradient(circle at 50% 42%, rgba(141, 61, 255, 0.18), transparent 34%),
              linear-gradient(140deg, #0b0712 0%, #140c24 38%, #0e0a1c 100%) !important;
          }

          .breezy-route-loading-logo {
            animation: breezy-loader-pulse 1.35s ease-in-out infinite;
            transform-origin: center;
          }
        `}
      </style>
      <Image
        src="/logo_only.png"
        alt="Breezy"
        className="breezy-route-loading-logo"
        width={96}
        height={96}
        priority
        style={{
          display: 'block',
          filter: 'drop-shadow(0 16px 36px rgba(91, 108, 255, 0.28))',
          height: 96,
          objectFit: 'contain',
          width: 96,
        }}
      />
    </div>
  )
}
