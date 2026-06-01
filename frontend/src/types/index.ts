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
