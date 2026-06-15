export type UserRole = 'user' | 'moderator' | 'administrator'

export interface User {
  id: string
  username: string
  email: string
  role: UserRole
  createdAt: string
}

export interface Post {
  id: string
  authorId: string
  /** Contenu limité à 280 caractères (validation côté API). */
  content: string
  likesCount: number
  commentsCount: number
  createdAt: string
}

export interface Profil {
  userId: string
  bio: string
  avatarUrl: string
  followersCount: number
  followingCount: number
}

/**
 * Vue agrégée d'un profil pour la page de consultation : identité (issue de
 * User), données de profil (Profil) et compteurs. Renvoyée par le
 * profil-service via l'API Gateway (`GET /profils/me` ou `/profils/:username`).
 */
export interface ProfilDetails {
  userId: string
  /** Nom affiché (modifiable). */
  displayName: string
  /** Identifiant unique sans « @ » (immuable). */
  username: string
  role: UserRole
  /** Compte actif ? `false` = banni (login bloqué + compte masqué). */
  isActive: boolean
  bio: string
  avatarUrl: string
  /** Image de bannière (en-tête du profil). */
  bannerUrl: string
  website: string
  location: string
  birthDate: string
  gender: 'male' | 'female' | ''
  /** Code pays ISO 3166-1 alpha-2. */
  nationality: string
  /** Date d'inscription (ISO 8601). */
  joinedAt: string
  updatedAt: string
  displayNameChangedAt: string
  visibility: 'public' | 'private'
  likesVisibility: 'public' | 'private'
  followersCount: number
  followingCount: number
  postsCount: number
  profileExists: boolean
}

/** Champs modifiables d'un profil (formulaire d'édition). */
export interface ProfilEditableFields {
  displayName: string
  bio: string
  avatarUrl: string
  bannerUrl: string
  website: string
  location: string
  birthDate: string
  gender: 'male' | 'female' | ''
  nationality: string
}

/**
 * Utilisateur affiché dans une liste d'abonnés / d'abonnements (modale des
 * relations) ou dans les suggestions. L'identité (id, username) vient du
 * user-service ; le décoratif (displayName, bio, avatar) est enrichi depuis
 * profil-service à la lecture (repli sur le username si absent).
 */
export interface RelationUser {
  id: string
  username: string
  displayName: string
  bio: string
  avatarUrl: string
}
