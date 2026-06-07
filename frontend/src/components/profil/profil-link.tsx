'use client'

import Link from 'next/link'

import { ROUTES, profilHref } from '@/lib/routes'
import { currentUserId } from '@/lib/posts'

interface ProfilLinkProps {
  /** Auteur ciblé : on a besoin de son id (pour détecter « moi ») + username. */
  author: { id: string; username: string }
  className?: string
  children: React.ReactNode
}

/**
 * Enveloppe ses enfants d'un lien vers le profil de l'auteur. Pointe vers
 * `/profil` si c'est l'utilisateur courant, sinon `/profil/<username>`. Si le
 * username est inconnu (auteur non résolu), rend un simple span (pas de lien
 * mort) en conservant la mise en page.
 */
export function ProfilLink({ author, className, children }: ProfilLinkProps) {
  if (!author.username) {
    return <span className={className}>{children}</span>
  }
  const href = currentUserId() === author.id ? ROUTES.profil : profilHref(author.username)
  return (
    <Link href={href} className={className}>
      {children}
    </Link>
  )
}
