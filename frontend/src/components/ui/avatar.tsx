'use client'

import * as React from 'react'
import * as AvatarPrimitive from '@radix-ui/react-avatar'

import { cn } from '@/lib/utils'

const AvatarImage = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Image>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Image>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Image
    ref={ref}
    className={cn('aspect-square h-full w-full rounded-full object-cover', className)}
    {...props}
  />
))
AvatarImage.displayName = AvatarPrimitive.Image.displayName

const Avatar = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Root>
>(({ className, children, ...props }, ref) => {
  const image = React.Children.toArray(children).find(
    (child) => React.isValidElement(child) && child.type === AvatarImage,
  ) as React.ReactElement<{ src?: string }> | undefined

  // Radix conserve sinon l'état "loaded" quand AvatarImage est retiré.
  const imageIdentity = image?.props.src || 'fallback'

  return (
    <AvatarPrimitive.Root
      key={imageIdentity}
      ref={ref}
      className={cn(
        // Pas d'`overflow-hidden` ici : l'image et le fallback portent déjà
        // `rounded-full` (donc restent circulaires), et le clip racine rognait
        // la pastille de présence (`ActivityPresenceDot`, enfant en position
        // absolue débordant volontairement le cercle) en un mince croissant.
        'relative flex h-10 w-10 shrink-0 rounded-full',
        className,
      )}
      {...props}
    >
      {children}
    </AvatarPrimitive.Root>
  )
})
Avatar.displayName = AvatarPrimitive.Root.displayName

const AvatarFallback = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Fallback>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Fallback>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Fallback
    ref={ref}
    className={cn(
      'flex h-full w-full items-center justify-center rounded-full bg-muted',
      className
    )}
    {...props}
  />
))
AvatarFallback.displayName = AvatarPrimitive.Fallback.displayName

export { Avatar, AvatarImage, AvatarFallback }
