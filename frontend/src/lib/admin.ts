/**
 * Client d'administration (réservé au rôle administrateur).
 *
 * L'annuaire a pour BASE l'auth-service (`GET /auth/users`), source faisant
 * autorité pour le rôle et l'état du compte (`is_active` = login autorisé).
 * L'identité visible (username, nom affiché, avatar) est ENRICHIE côté front
 * depuis user-service + profil-service (`resolveUser`, mémoïsé) — même pattern
 * cross-service que la recherche de comptes (cf. la doc d'architecture).
 *
 * « Bannir » = bloquer le login (auth) ET masquer le compte (user) : deux
 * écritures orchestrées ici (une donnée = un service). Le changement de rôle
 * ne prend effet dans le JWT qu'au prochain /refresh de l'utilisateur ciblé.
 *
 * ⚠️ À usage CLIENT uniquement (`apiFetch` lit le token en localStorage).
 */

import { apiFetch } from '@/lib/auth-client'
import { mapRole } from '@/lib/session'
import { resolveUser } from '@/lib/user-cache'
import type { UserCertification, UserRole } from '@/types'

/** Erreur d'appel API admin portant le code HTTP. */
export class AdminApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'AdminApiError'
    this.status = status
  }
}

/** Compte vu par l'admin : identité auth + enrichissement user/profil. */
export interface AdminUser {
  id: string
  email: string
  role: UserRole
  /** Compte actif = peut se connecter. `false` = banni (login bloqué). */
  isActive: boolean
  /** Date du bannissement (point de départ de la purge RGPD à 5 ans). */
  deactivatedAt?: string
  createdAt: string
  username: string
  displayName: string
  avatarUrl: string
  certification: UserCertification
}

interface ApiAuthUser {
  id: string
  email: string
  role: string
  is_active: boolean
  deactivated_at?: string
  created_at: string
}

/** Rôle front → rôle back (`administrator` ⇒ `admin` ; les autres identiques). */
function toBackendRole(role: UserRole): string {
  return role === 'administrator' ? 'admin' : role
}

async function unwrap<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => null)) as
    | { data?: T; error?: string }
    | null
  if (!res.ok) {
    throw new AdminApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
  return (body?.data ?? null) as T
}

async function expectOk(res: Response): Promise<void> {
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new AdminApiError(body?.error ?? `Erreur ${res.status}`, res.status)
  }
}

/** Données de création d'un compte par un admin. */
export interface CreateAccountInput {
  username: string
  email: string
  password: string
}

/** Résultat de la création : username EFFECTIF (éventuellement suffixé) + état. */
export interface CreatedAccount {
  id: string
  email: string
  username: string
  /** true si le username a dû être suffixé (handle demandé déjà pris). */
  usernamePending: boolean
}

interface ApiUserRow {
  id: string
  username: string
  username_pending?: boolean
}

/**
 * Crée un compte de force (admin) en orchestrant les trois services, comme
 * `hardDeleteUser` pour l'effacement :
 *   1. auth-service `POST /auth/users` : credentials (vérifié + mot de passe
 *      TEMPORAIRE), qui envoie le mot de passe par e-mail (best-effort côté back) ;
 *   2. user-service `POST /users/admin` : ligne `users` (id imposé), username
 *      suffixé + `username_pending` si le handle est déjà pris ;
 *   3. profil-service `POST /profils/admin` : profil (display_name = username
 *      effectif) pour que le gate d'onboarding ne se déclenche pas.
 *
 * Les étapes 1-2 sont critiques (erreur propagée) ; l'étape 3 est best-effort
 * (le profil sera sinon créé au provisioning paresseux). Renvoie le username
 * effectif et l'état `usernamePending`.
 */
export async function createAccount(input: CreateAccountInput): Promise<CreatedAccount> {
  const created = await unwrap<{ id: string; email: string }>(
    await apiFetch('/auth/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: input.email,
        password: input.password,
        username: input.username,
      }),
    }),
  )

  const user = await unwrap<ApiUserRow>(
    await apiFetch('/users/admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: created.id, username: input.username }),
    }),
  )

  try {
    await apiFetch('/profils/admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: created.id, display_name: user.username }),
    })
  } catch {
    /* best-effort : profil créable ensuite via provisioning paresseux */
  }

  return {
    id: created.id,
    email: created.email,
    username: user.username,
    usernamePending: user.username_pending ?? false,
  }
}

/** Annuaire des comptes (admin). `query` filtre par email. */
export async function listAdminUsers(
  query = '',
  limit = 50,
  offset = 0,
): Promise<AdminUser[]> {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
    offset: String(offset),
  })
  const users = await unwrap<ApiAuthUser[]>(await apiFetch(`/auth/users?${params}`))

  return Promise.all(
    (users ?? []).map(async (u): Promise<AdminUser> => {
      const resolved = await resolveUser(u.id)
      return {
        id: u.id,
        email: u.email,
        role: mapRole(u.role),
        isActive: u.is_active,
        deactivatedAt: u.deactivated_at,
        createdAt: u.created_at,
        username: resolved.username,
        displayName: resolved.displayName,
        avatarUrl: resolved.avatarUrl,
        certification: resolved.certification,
      }
    }),
  )
}

/** Change le rôle d'un compte (effet au prochain refresh JWT de la cible). */
export async function updateUserRole(id: string, role: UserRole): Promise<void> {
  await expectOk(
    await apiFetch(`/auth/users/${id}/role`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role: toBackendRole(role) }),
    }),
  )
}

/** Attribue ou retire une certification décorative (modérateur/admin). */
export async function updateUserCertification(
  id: string,
  certification: UserCertification,
): Promise<void> {
  await expectOk(
    await apiFetch(`/profils/${id}/certification`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ certification }),
    }),
  )
}

/**
 * Bannit (`banned=true`) ou réactive (`banned=false`) un compte. Orchestre les
 * deux services : auth (bloque le login) PUIS user (masque le compte). L'ordre
 * met l'effet critique (accès) en premier ; la visibilité suit.
 */
export async function setUserBanned(id: string, banned: boolean): Promise<void> {
  const active = !banned
  // 1) auth-service = verrou d'accès (login/refresh). Critique : on propage l'erreur.
  await expectOk(
    await apiFetch(`/auth/users/${id}/status`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_active: active }),
    }),
  )
  // 2) user-service = visibilité publique. Un 404 est toléré : un compte jamais
  // provisionné côté user (inscrit mais jamais passé par /users/me) n'a ni
  // posts ni profil → rien à masquer. Les autres erreurs sont propagées.
  const res = await apiFetch(`/users/${id}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_active: active }),
  })
  if (res.status !== 404) {
    await expectOk(res)
  }
}

// Étapes de l'effacement RGPD, dans l'ordre : on purge d'abord les données
// applicatives et les IDENTIFIANTS (auth) EN DERNIER — ainsi, si une étape
// échoue, le compte existe encore et l'opération est rejouable. Chaque service
// possède SA donnée (une donnée = un service), orchestré ici côté admin comme
// le bannissement. Un 404 est toléré (donnée déjà absente / jamais créée).
const ERASE_STEPS: { label: string; path: (id: string) => string }[] = [
  { label: 'profil', path: (id) => `/profils/${id}` },
  { label: 'posts', path: (id) => `/posts/by-author/${id}` },
  { label: 'messages', path: (id) => `/messages/users/${id}` },
  { label: 'media', path: (id) => `/media/owners/${id}` },
  { label: 'user', path: (id) => `/users/${id}/hard` },
  { label: 'auth', path: (id) => `/auth/users/${id}` }, // identifiants en dernier
]

/**
 * Efface DÉFINITIVEMENT un compte et toutes ses données à travers les services
 * (effacement RGPD, admin). Best-effort + 404 toléré ; lève une erreur listant
 * les services en échec (effacement partiel à rejouer).
 */
export async function hardDeleteUser(id: string): Promise<void> {
  const failed: string[] = []
  for (const step of ERASE_STEPS) {
    try {
      const res = await apiFetch(step.path(id), { method: 'DELETE' })
      if (!res.ok && res.status !== 404) failed.push(step.label)
    } catch {
      failed.push(step.label)
    }
  }
  if (failed.length > 0) {
    throw new AdminApiError(`Effacement partiel : ${failed.join(', ')}`, 500)
  }
}
