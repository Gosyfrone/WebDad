import { apiUrl } from '@/lib/config'

/**
 * Provisioning paresseux du user via l'email : `GET /users/me` crée la ligne
 * `users` (handle dérivé de l'email). Sert de repli (connexion, ou course au
 * register). NB : le profil (profil-service) n'a PAS de provisioning paresseux
 * — sa seule création est `POST /profils` (cf. provisionProfil), appelé au
 * register avec display_name=username, birth_date et gender.
 */
async function provisionFromEmail(token: string): Promise<void> {
  try {
    await fetch(apiUrl('/users/me'), {
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    })
  } catch {
    // Silencieux : provisioning différé au prochain /users/me.
  }
}

/** POST /profils : pose le profil initial issu du formulaire register. */
async function provisionProfil(
  token: string,
  options: { displayName: string; birthDate?: string; gender?: string },
): Promise<void> {
  await fetch(apiUrl('/profils'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      display_name: options.displayName,
      birth_date: options.birthDate
        ? new Date(options.birthDate).toISOString()
        : undefined,
      gender: options.gender || undefined,
    }),
    cache: 'no-store',
  })
}

/**
 * Provisioning de l'utilisateur après login / register, dans user-service ET
 * profil-service (un compte = une identité + un profil).
 *
 *   - Avec `username` (inscription) : `POST /users { username }` (handle choisi)
 *     puis `POST /profils` avec display_name=username, birth_date et gender.
 *     La disponibilité du username est pré-vérifiée côté page register ; en cas
 *     de course rarissime, on retombe sur les valeurs dérivées de l'email.
 *   - Sans `username` (connexion) : `GET /users/me` provisionne seulement
 *     l'identité dérivée de l'email. La création du profil complet passe par
 *     le register, car birth_date/gender viennent du formulaire.
 *
 * Best-effort : une erreur ici ne doit JAMAIS casser l'authentification.
 * À usage serveur uniquement (route handlers) : le token n'est pas exposé au client.
 */
export async function provisionUser(
  token: string,
  options?: { username?: string; birthDate?: string; gender?: string },
): Promise<void> {
  const username = options?.username?.trim()
  if (!username) {
    await provisionFromEmail(token)
    return
  }

  try {
    const response = await fetch(apiUrl('/users'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ username }),
      cache: 'no-store',
    })

    if (!response.ok) {
      // username pris/invalide (course après le pré-check) → repli dérivé email.
      await provisionFromEmail(token)
      return
    }

    // user OK → on pose le profil avec display_name = username choisi.
    await provisionProfil(token, {
      displayName: username,
      birthDate: options?.birthDate,
      gender: options?.gender,
    })
  } catch {
    await provisionFromEmail(token)
  }
}

/** Marque la dernière vraie connexion dans profil-service (best-effort). */
export async function markLoginActivity(token: string): Promise<void> {
  try {
    await fetch(apiUrl('/profils/me/activity'), {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    })
  } catch {
    // Silencieux : l'activité sera remise à jour à la prochaine connexion.
  }
}

/** Marque la déconnexion dans profil-service (best-effort). */
export async function markLogoutActivity(token: string): Promise<void> {
  try {
    await fetch(apiUrl('/profils/me/activity/offline'), {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    })
  } catch {
    // Silencieux : la déconnexion locale prime.
  }
}
