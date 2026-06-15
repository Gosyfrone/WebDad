'use client'

import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from '@/components/ui/toast'
import { useT } from '@/components/language-provider'
import { useToast } from '@/hooks/use-toast'

export function Toaster() {
  const { toasts } = useToast()
  const t = useT()

  return (
    // duration : auto-fermeture des toasts après 3 s.
    <ToastProvider duration={3000}>
      {toasts.map(function ({ id, title, description, action, ...props }) {
        return (
          <Toast key={id} {...props}>
            <div className="grid gap-1">
              {title && <ToastTitle>{title}</ToastTitle>}
              {description && <ToastDescription>{description}</ToastDescription>}
            </div>
            {action}
            <ToastClose aria-label={t('common.close')} />
          </Toast>
        )
      })}
      <ToastViewport />
    </ToastProvider>
  )
}
