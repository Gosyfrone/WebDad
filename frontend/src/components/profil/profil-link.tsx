'use client'

import Link from 'next/link'

import { ROUTES, profilHref } from '@/lib/routes'
import { currentUserId } from '@/lib/posts'
import { useAuthGate } from '@/components/auth-prompt-provider'
import { ProfileHoverCard } from '@/components/profil/profile-hover-card'

interface ProfilLinkProps {
  /** Auteur ciblé : on a besoin de son id (pour détecter « moi ») + username. */
  author: { id: string; username: string }
  className?: string
  /** Destination personnalisée, utile pour conserver un contexte de retour. */
  href?: string
  /** Désactive la carte de survol pour les liens invisibles/étirés. */
  preview?: boolean
  children: React.ReactNode
}

type AnchorProps = Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, 'href'>

/**
 * Enveloppe ses enfants d'un lien vers le profil de l'auteur. Pointe vers
 * `/profil` si c'est l'utilisateur courant, sinon `/profil/<username>`. Si le
 * username est inconnu (auteur non résolu), rend un simple span (pas de lien
 * mort) en conservant la mise en page.
 */
export function ProfilLink({
  author,
  className,
  href,
  preview = true,
  children,
  ...linkProps
}: ProfilLinkProps & AnchorProps) {
  const { isVisitor } = useAuthGate()
  if (!author.username) {
    return <span className={className}>{children}</span>
  }
  const targetHref =
    href ?? (currentUserId() === author.id ? ROUTES.profil : profilHref(author.username))
  // Visiteur : pas d'aperçu au survol (la carte charge le graphe social
  // authentifié). Le lien reste, mais le middleware le renverra vers /login.
  if (!preview || isVisitor) {
    return (
      <Link href={targetHref} className={className} {...linkProps}>
        {children}
      </Link>
    )
  }
  return (
    <ProfileHoverCard author={author} href={targetHref} className={className} {...linkProps}>
      {children}
    </ProfileHoverCard>
  )
}
